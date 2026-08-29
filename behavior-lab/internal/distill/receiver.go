package distill

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const ContractID = "tlaloc.origami-receiver-distillation.r0"

// SwarmStep is an observed semantic transition produced while a richer Tlaloc
// swarm solves a receiver task. Distillation preserves the externally relevant
// trigger/transition/effect and intentionally discards agent-internal reasoning.
type SwarmStep struct {
	Agent     string `json:"agent"`
	State     string `json:"state"`
	Token     string `json:"token"`
	Action    string `json:"action"`
	NextState string `json:"next_state"`
	Emit      string `json:"emit,omitempty"`
}

// MicroRule is the simple deterministic target form intended for Origami's
// carrier/runtime. Tlaloc proposes these rules; Origami remains authoritative
// for validating their semantics before promotion.
type MicroRule struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Token     string `json:"token"`
	Action    string `json:"action"`
	NextState string `json:"next_state"`
	Emit      string `json:"emit,omitempty"`
}

type Candidate struct {
	Schema             string      `json:"schema"`
	ID                 string      `json:"id"`
	Prompt             string      `json:"prompt"`
	BootStrategy       []string    `json:"boot_strategy"`
	RosettaConstraints []string    `json:"rosetta_constraints"`
	Program            []MicroRule `json:"program"`
	SourceTraceSHA256  string      `json:"source_trace_sha256"`
}

// Distill converts a successful swarm trace into a bounded candidate program.
// Identical semantic transitions collapse to one rule. Conflicting transitions
// are rejected instead of being hidden behind ordering or model judgment.
func Distill(prompt string, trace []SwarmStep) (Candidate, error) {
	if len(trace) == 0 {
		return Candidate{}, fmt.Errorf("swarm trace is required")
	}

	seen := map[string]MicroRule{}
	program := make([]MicroRule, 0, len(trace))
	for _, step := range trace {
		if step.State == "" || step.Token == "" || step.Action == "" || step.NextState == "" {
			return Candidate{}, fmt.Errorf("swarm step requires state, token, action and next_state")
		}
		key := step.State + "\x00" + step.Token
		rule := MicroRule{
			ID:        fmt.Sprintf("m%04d", len(program)),
			State:     step.State,
			Token:     step.Token,
			Action:    step.Action,
			NextState: step.NextState,
			Emit:      step.Emit,
		}
		if prior, ok := seen[key]; ok {
			if prior.Action != rule.Action || prior.NextState != rule.NextState || prior.Emit != rule.Emit {
				return Candidate{}, fmt.Errorf("non-deterministic swarm behavior for state=%q token=%q", step.State, step.Token)
			}
			continue
		}
		seen[key] = rule
		program = append(program, rule)
	}

	traceBytes, err := json.Marshal(trace)
	if err != nil {
		return Candidate{}, err
	}
	sum := sha256.Sum256(traceBytes)
	traceHash := hex.EncodeToString(sum[:])
	candidateID := traceHash[:16]

	return Candidate{
		Schema: ContractID,
		ID:     candidateID,
		Prompt: prompt,
		BootStrategy: []string{
			"locate carrier BOOT before payload interpretation",
			"discover carrier-local ROSETTA from BOOT",
			"initialize declared micro-program before memory traversal",
		},
		RosettaConstraints: []string{
			"physical symbol meaning is carrier-local",
			"do not assume glyph semantics from prior carriers",
			"unknown symbol fails closed",
		},
		Program:           program,
		SourceTraceSHA256: traceHash,
	}, nil
}
