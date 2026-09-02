// Package actionflow connects a swarm's answer to the deterministic
// action boundary. The swarm (swarmask) produces a verified answer; that
// answer may imply an action ("move these files", "restart nginx"). This
// package takes the indeterministic proposal, runs every candidate through
// envelope.Authorize, and decides — from the action's risk class — whether
// it may auto-execute or must go back to the human.
//
// Invariant #1 in one flow: a Proposer PROPOSES (untrusted), the envelope
// AUTHORIZES (deterministic, against the intent's policy), the executor
// VERIFIES (world state). A refused candidate is recorded, never run.
package actionflow

import (
	"context"
	"regexp"
	"strings"

	"tlaloc.local/behaviorlab/internal/action"
)

// ProposeInput is what a Proposer sees: the question, the swarm's answer,
// and the material to verify it. A Proposer must not invent actions the
// answer does not support.
type ProposeInput struct {
	Question string
	Answer   string
	// Grounded is the swarm consolidator's plain grounding verdict. Used
	// only when Run is given no verify.Spine.
	Grounded bool
	// AnswerJSON and Evidence feed the verify.Spine when Run is given one:
	// AnswerJSON is the structured answer output (STRUCTURAL level),
	// Evidence is the page/source content the answer should be grounded in
	// (SEMANTIC level, as the claim<-Answer / evidence<-Evidence pair).
	AnswerJSON []byte
	Evidence   string
}

// Proposer is the indeterministic slot: today a marker parser, tomorrow a
// small classifier or an LLM constrained to emit ActionCandidates. Its
// output is untrusted data.
type Proposer interface {
	Propose(ctx context.Context, in ProposeInput) ([]action.ActionCandidate, error)
}

// markerLine matches "PROPOSE FILE.MOVE source=/a/x dest=/b/x" style lines.
var markerLine = regexp.MustCompile(`(?i)^\s*PROPOSE\s+([A-Z][A-Z0-9_.]+)\s*(.*)$`)
var argPair = regexp.MustCompile(`([A-Za-z0-9_]+)\s*=\s*("[^"]*"|\S+)`)

// MarkerProposer is the deterministic default Proposer: it only proposes an
// action when the answer contains an explicit, well-formed PROPOSE line,
// and only when the answer was grounded. It never infers an action from
// prose — that is a model's job, and a model plugs in here later.
type MarkerProposer struct{}

func (MarkerProposer) Propose(_ context.Context, in ProposeInput) ([]action.ActionCandidate, error) {
	if !in.Grounded {
		return nil, nil
	}
	var candidates []action.ActionCandidate
	for _, line := range strings.Split(in.Answer, "\n") {
		match := markerLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		candidate := action.ActionCandidate{
			Capability: strings.ToUpper(match[1]),
			Arguments:  map[string]string{},
			ProposedBy: "actionflow-marker-proposer",
		}
		for _, pair := range argPair.FindAllStringSubmatch(match[2], -1) {
			candidate.Arguments[pair[1]] = strings.Trim(pair[2], `"`)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// Decision is what the flow concluded for one proposed action.
type Decision string

const (
	// Refused: the envelope rejected the candidate (unknown capability, over
	// the risk ceiling, outside the sandbox, bad args).
	Refused Decision = "REFUSED"
	// AutoExecuted: authorized and low-risk enough to run without asking;
	// the executor result says whether it verified.
	AutoExecuted Decision = "AUTO_EXECUTED"
	// NeedsConfirmation: authorized but too consequential to run
	// automatically — handed back for a human yes/no.
	NeedsConfirmation Decision = "NEEDS_CONFIRMATION"
	// AuthorizedNotRun: authorized and auto-executable, but no executor was
	// supplied to the flow.
	AuthorizedNotRun Decision = "AUTHORIZED_NOT_RUN"
)

// autoExecutable is the safety line: read-only and reversible-local actions
// may run automatically; anything that cannot be undone, leaves the
// machine, or is privileged waits for a human.
func autoExecutable(risk action.RiskClass) bool {
	return risk <= action.R1LocalReversible
}
