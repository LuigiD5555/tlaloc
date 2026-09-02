package actionflow

import (
	"context"
	"strings"

	"tlaloc.local/behaviorlab/internal/action"
	"tlaloc.local/behaviorlab/internal/envelope"
	"tlaloc.local/behaviorlab/internal/executor"
	"tlaloc.local/behaviorlab/internal/intent"
	"tlaloc.local/behaviorlab/internal/sentinel"
	"tlaloc.local/behaviorlab/internal/verify"
)

// ProposedAction is one candidate carried through the flow with its
// verdict.
type ProposedAction struct {
	Candidate     action.ActionCandidate `json:"candidate"`
	Decision      Decision               `json:"decision"`
	RefusedReason string                 `json:"refused_reason,omitempty"`
	Authorized    *action.ActionIR       `json:"authorized,omitempty"`
	Concerns      []sentinel.Concern     `json:"concerns,omitempty"`
	Execution     *executor.Result       `json:"execution,omitempty"`
}

// FlowResult is the account of turning one answer into zero or more
// authorized/executed/deferred actions.
type FlowResult struct {
	// AnswerVerification is the verify.Spine report on the answer itself,
	// when Run was given a spine. If its verdict is not VERIFIED, no
	// candidate is proposed.
	AnswerVerification *verify.Report `json:"answer_verification,omitempty"`
	// AnswerConcerns are answer-level sentinel findings (numeric
	// inconsistency, OOD, observation conflict) — advisory, surfaced for a
	// human; they do not by themselves stop the flow.
	AnswerConcerns    []sentinel.Concern `json:"answer_concerns,omitempty"`
	Actions           []ProposedAction   `json:"actions"`
	Proposed          int                `json:"proposed"`
	Refused           int                `json:"refused"`
	AutoExecuted      int                `json:"auto_executed"`
	NeedsConfirmation int                `json:"needs_confirmation"`
}

// Run is the full connection: verify -> propose -> authorize -> (maybe)
// execute. spine may be nil (then the proposer's plain in.Grounded gates
// proposal); exec may be the zero Executor{} (then authorized auto-
// executable actions come back as AUTHORIZED_NOT_RUN instead of running).
func Run(
	ctx context.Context,
	proposer Proposer,
	spine *verify.Spine,
	panel *sentinel.Panel,
	compiled intent.CompiledIntent,
	catalog action.Catalog,
	exec executor.Executor,
	in ProposeInput,
) (FlowResult, error) {
	result := FlowResult{}

	policy, policyErr := envelope.PolicyFor(compiled)
	if policyErr != nil {
		return FlowResult{}, policyErr
	}

	if spine != nil {
		report := spine.Verify(ctx, verify.Input{
			Output:   in.AnswerJSON,
			Claim:    in.Answer,
			Evidence: in.Evidence,
		})
		result.AnswerVerification = &report
		// The spine is now the authoritative grounding verdict.
		in.Grounded = report.Verdict == verify.Verified
		if !in.Grounded {
			return result, nil // an unverified answer proposes nothing
		}
	}

	if panel != nil {
		answerReview := panel.Review(ctx, sentinel.Subject{
			Question: in.Question, Answer: in.Answer, Evidence: in.Evidence,
		})
		result.AnswerConcerns = answerReview.Concerns
		if answerReview.Blocked {
			return result, nil // an answer-level BLOCK stops here
		}
	}

	candidates, err := proposer.Propose(ctx, in)
	if err != nil {
		return FlowResult{}, err
	}

	result.Proposed = len(candidates)
	haveExecutor := exec.Impls != nil && exec.Check != nil

	for _, candidate := range candidates {
		entry := ProposedAction{Candidate: candidate}

		authorized, authErr := action.Compile(candidate, catalog, policy)
		if authErr != nil {
			entry.Decision = Refused
			entry.RefusedReason = authErr.Error()
			result.Refused++
			result.Actions = append(result.Actions, entry)
			continue
		}
		authorizedCopy := authorized
		entry.Authorized = &authorizedCopy

		// Independent sentinel review of the authorized action — a BLOCK
		// concern overrides the authorization (belt and suspenders).
		if panel != nil {
			review := panel.Review(ctx, sentinel.Subject{
				Question:       in.Question,
				Answer:         in.Answer,
				Evidence:       in.Evidence,
				ProposedAction: &authorizedCopy,
				Policy:         &policy,
				AllowedPaths:   compiled.StayInside,
			})
			entry.Concerns = review.Concerns
			if review.Blocked {
				entry.Decision = Refused
				entry.RefusedReason = "sentinel: " + blockReasons(review.Concerns)
				result.Refused++
				result.Actions = append(result.Actions, entry)
				continue
			}
		}

		if !autoExecutable(authorized.Risk) {
			entry.Decision = NeedsConfirmation
			result.NeedsConfirmation++
			result.Actions = append(result.Actions, entry)
			continue
		}

		if !haveExecutor {
			entry.Decision = AuthorizedNotRun
			result.Actions = append(result.Actions, entry)
			continue
		}

		execResult, execErr := exec.Execute(ctx, authorized)
		if execErr != nil {
			// Protocol failure (unknown impl) — treat like a refusal so the
			// flow never claims an action ran when it did not.
			entry.Decision = Refused
			entry.RefusedReason = execErr.Error()
			result.Refused++
			result.Actions = append(result.Actions, entry)
			continue
		}
		execCopy := execResult
		entry.Execution = &execCopy
		entry.Decision = AutoExecuted
		result.AutoExecuted++
		result.Actions = append(result.Actions, entry)
	}

	return result, nil
}

func blockReasons(concerns []sentinel.Concern) string {
	reasons := []string{}
	for _, concern := range concerns {
		if concern.Severity == sentinel.Block {
			reasons = append(reasons, concern.Kind)
		}
	}
	return strings.Join(reasons, ", ")
}
