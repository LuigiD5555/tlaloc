package actionflow

import (
	"context"
	"fmt"
	"testing"

	"tlaloc.local/behaviorlab/internal/action"
	"tlaloc.local/behaviorlab/internal/executor"
	"tlaloc.local/behaviorlab/internal/intent"
	"tlaloc.local/behaviorlab/internal/sentinel"
	"tlaloc.local/behaviorlab/internal/verify"
)

type fakeSemantic struct{ agree bool }

func (f fakeSemantic) AgreesWith(context.Context, string, string) (bool, float64, error) {
	return f.agree, 0.9, nil
}

func compiledIntent(t *testing.T, constraints ...intent.Constraint) intent.CompiledIntent {
	t.Helper()
	compiled, err := intent.Compile(intent.IntentIR{
		Version:         "1",
		RequiredOutputs: []string{"ORGANIZE_DOCUMENTS"},
		Constraints:     constraints,
		Risk:            intent.RiskProfile{Level: "medium"},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return compiled
}

// A non-grounded answer proposes nothing — the proposer will not act on an
// answer the swarm could not verify.
func TestMarkerProposer_IgnoresUngroundedAnswer(t *testing.T) {
	candidates, _ := MarkerProposer{}.Propose(context.Background(), ProposeInput{
		Answer:   "PROPOSE FILE.MOVE source=/a/x.pdf destination=/b/x.pdf",
		Grounded: false,
	})
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates from an ungrounded answer, got %d", len(candidates))
	}
}

// Prose without a PROPOSE marker yields nothing — the deterministic
// proposer never infers an action from free text.
func TestMarkerProposer_NoMarkerNoAction(t *testing.T) {
	candidates, _ := MarkerProposer{}.Propose(context.Background(), ProposeInput{
		Answer:   "You should probably move those invoices into per-supplier folders.",
		Grounded: true,
	})
	if len(candidates) != 0 {
		t.Fatalf("prose must not become an action, got %d", len(candidates))
	}
}

// Full flow: a grounded answer with a reversible action inside the sandbox
// auto-executes and verifies.
func TestRun_ReversibleInSandboxAutoExecutes(t *testing.T) {
	fs := newFakeFS("/home/u/Facturas/inbox/acme.pdf")
	exec := executor.Executor{
		Impls: map[string]executor.Impl{"FILE.MOVE": fs.move},
		Check: fs.check,
	}
	compiled := compiledIntent(t,
		intent.Constraint{Kind: "stay_inside", Value: "/home/u/Facturas"},
		intent.Constraint{Kind: "max_action_risk", Value: "R2_LOCAL_IRREVERSIBLE"},
	)

	in := ProposeInput{
		Grounded: true,
		Answer:   "The inbox invoice belongs under acme.\nPROPOSE FILE.MOVE source=/home/u/Facturas/inbox/acme.pdf destination=/home/u/Facturas/acme/acme.pdf",
	}
	result, err := Run(context.Background(), MarkerProposer{}, nil, nil, compiled, action.DefaultCatalog(), exec, in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Proposed != 1 || result.AutoExecuted != 1 {
		t.Fatalf("expected 1 proposed + 1 auto-executed: %+v", result)
	}
	if !result.Actions[0].Execution.Verified {
		t.Errorf("execution should have verified: %+v", result.Actions[0].Execution)
	}
	if fs.exists["/home/u/Facturas/inbox/acme.pdf"] || !fs.exists["/home/u/Facturas/acme/acme.pdf"] {
		t.Errorf("world not moved: %+v", fs.exists)
	}
}

// An irreversible action is authorized but held for confirmation, never run.
func TestRun_IrreversibleNeedsConfirmation(t *testing.T) {
	compiled := compiledIntent(t, intent.Constraint{Kind: "max_action_risk", Value: "R2_LOCAL_IRREVERSIBLE"})
	in := ProposeInput{Grounded: true, Answer: "PROPOSE FILE.DELETE path=/home/u/tmp/old.log"}

	result, _ := Run(context.Background(), MarkerProposer{}, nil, nil, compiled, action.DefaultCatalog(), executor.Executor{}, in)
	if result.NeedsConfirmation != 1 || result.AutoExecuted != 0 {
		t.Fatalf("an R2 action must need confirmation, not run: %+v", result)
	}
	if result.Actions[0].Authorized == nil {
		t.Error("it should still be authorized (just not run)")
	}
}

// Prompt-injection shape: the grounded answer carries a hostile PROPOSE
// line for a path outside the sandbox. It is proposed, then refused.
func TestRun_HostileProposalRefused(t *testing.T) {
	compiled := compiledIntent(t,
		intent.Constraint{Kind: "stay_inside", Value: "/home/u/Facturas"},
		intent.Constraint{Kind: "max_action_risk", Value: "R2_LOCAL_IRREVERSIBLE"},
	)
	in := ProposeInput{
		Grounded: true,
		Answer:   "PROPOSE FILE.DELETE path=/home/u/.ssh/id_rsa",
	}
	result, _ := Run(context.Background(), MarkerProposer{}, nil, nil, compiled, action.DefaultCatalog(), executor.Executor{}, in)
	if result.Refused != 1 || result.AutoExecuted != 0 || result.NeedsConfirmation != 0 {
		t.Fatalf("a proposal outside the sandbox must be refused: %+v", result)
	}
	if result.Actions[0].RefusedReason == "" {
		t.Error("a refusal should say why")
	}
}

// When a verify.Spine is supplied, its verdict — not the plain Grounded
// flag — gates proposal. A structurally-broken / semantically-unsupported
// answer proposes nothing even if it carries a valid PROPOSE line.
func TestRun_SpineGatesProposal(t *testing.T) {
	compiled := compiledIntent(t,
		intent.Constraint{Kind: "stay_inside", Value: "/home/u/Facturas"},
		intent.Constraint{Kind: "max_action_risk", Value: "R2_LOCAL_IRREVERSIBLE"},
	)
	answer := "PROPOSE FILE.MOVE source=/home/u/Facturas/a.pdf destination=/home/u/Facturas/x/a.pdf"
	in := ProposeInput{
		Answer:     answer,
		AnswerJSON: []byte(`{"answer":"` + answer + `","grounded":true}`),
		Evidence:   "the page discusses invoice filing",
	}

	// Semantic verifier disagrees -> UNVERIFIED -> nothing proposed.
	disagree := &verify.Spine{Semantic: fakeSemantic{agree: false}, SemanticMinConfidence: 0.6}
	result, err := Run(context.Background(), MarkerProposer{}, disagree, nil, compiled, action.DefaultCatalog(), executor.Executor{}, in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.AnswerVerification == nil || result.AnswerVerification.Verdict != verify.Unverified {
		t.Fatalf("expected an UNVERIFIED spine report, got %+v", result.AnswerVerification)
	}
	if result.Proposed != 0 {
		t.Errorf("an unverified answer must propose nothing, got %d", result.Proposed)
	}

	// Semantic verifier agrees + structural ok -> VERIFIED -> proposes.
	agree := &verify.Spine{Semantic: fakeSemantic{agree: true}, SemanticMinConfidence: 0.6}
	result, _ = Run(context.Background(), MarkerProposer{}, agree, nil, compiled, action.DefaultCatalog(), executor.Executor{}, in)
	if result.AnswerVerification.Verdict != verify.Verified {
		t.Fatalf("expected VERIFIED, got %s", result.AnswerVerification.Verdict)
	}
	if result.Proposed != 1 {
		t.Errorf("a verified answer with a PROPOSE line should propose 1, got %d", result.Proposed)
	}
}

// With a panel, answer-level concerns (a number not in the evidence) are
// surfaced but advisory — they do not stop a clean, in-scope action.
func TestRun_SentinelAnswerConcernsAreAdvisory(t *testing.T) {
	compiled := compiledIntent(t,
		intent.Constraint{Kind: "stay_inside", Value: "/home/u/Facturas"},
		intent.Constraint{Kind: "max_action_risk", Value: "R2_LOCAL_IRREVERSIBLE"},
	)
	panel := sentinel.DefaultPanel()
	in := ProposeInput{
		Grounded: true,
		Evidence: "The 2019 filing lives in the Facturas folder.",
		Answer:   "Filed in 2099.\nPROPOSE FILE.DELETE path=/home/u/Facturas/dupe.pdf",
	}
	result, err := Run(context.Background(), MarkerProposer{}, nil, &panel, compiled, action.DefaultCatalog(), executor.Executor{}, in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	foundNumeric := false
	for _, concern := range result.AnswerConcerns {
		if concern.Kind == "unsupported_number" {
			foundNumeric = true
		}
	}
	if !foundNumeric {
		t.Errorf("expected an unsupported_number answer concern, got %+v", result.AnswerConcerns)
	}
	// The R2 delete is still authorized (advisory concern did not block it),
	// it just needs confirmation.
	if result.NeedsConfirmation != 1 {
		t.Errorf("the in-scope R2 action should still be authorized -> NEEDS_CONFIRMATION: %+v", result)
	}
}

// A sentinel BLOCK concern on an authorized action refuses it — the panel
// is a second independent gate after the envelope. PermissionSentinel
// re-derives the risk-vs-ceiling check; here the policy is mutated after
// authorization to simulate a stale/inconsistent policy path.
func TestRun_SentinelBlocksOnPermission(t *testing.T) {
	compiled := compiledIntent(t, intent.Constraint{Kind: "max_action_risk", Value: "R4_PRIVILEGED"})
	panel := sentinel.Panel{Sentinels: []sentinel.Sentinel{tightPermissionSentinel{}}}
	in := ProposeInput{Grounded: true, Answer: "PROPOSE SERVICE.RESTART unit=nginx"}

	result, err := Run(context.Background(), MarkerProposer{}, nil, &panel, compiled, action.DefaultCatalog(), executor.Executor{}, in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Refused != 1 || len(result.Actions[0].Concerns) == 0 {
		t.Fatalf("a BLOCK concern must refuse the action with its concerns attached: %+v", result)
	}
}

// tightPermissionSentinel blocks any action above R1 regardless of policy —
// a stand-in for an independent policy source stricter than the envelope's.
type tightPermissionSentinel struct{}

func (tightPermissionSentinel) Name() string { return "tight-permission" }
func (tightPermissionSentinel) Inspect(_ context.Context, s sentinel.Subject) ([]sentinel.Concern, error) {
	if s.ProposedAction != nil && s.ProposedAction.Risk > action.R1LocalReversible {
		return []sentinel.Concern{{Kind: "too_risky", Severity: sentinel.Block, Detail: "above R1"}}, nil
	}
	return nil, nil
}

// --- in-memory fake filesystem for the executor ---

type fakeFS struct{ exists map[string]bool }

func newFakeFS(paths ...string) *fakeFS {
	fs := &fakeFS{exists: map[string]bool{}}
	for _, p := range paths {
		fs.exists[p] = true
	}
	return fs
}

func (fs *fakeFS) check(_ context.Context, kind, arg string, args map[string]string) (bool, error) {
	path := args[arg]
	switch kind {
	case "path_exists":
		return fs.exists[path], nil
	case "path_absent":
		return !fs.exists[path], nil
	default:
		return false, fmt.Errorf("unknown check %q", kind)
	}
}

func (fs *fakeFS) move(_ context.Context, args map[string]string) error {
	src, dst := args["source"], args["destination"]
	if !fs.exists[src] {
		return fmt.Errorf("source missing")
	}
	delete(fs.exists, src)
	fs.exists[dst] = true
	return nil
}
