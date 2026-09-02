package learningpolicy

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
)

func bp(v bool) *bool       { return &v }
func fp(v float64) *float64 { return &v }

func TestDeriveProtectsPositiveOutcomeAndRequiresParity(t *testing.T) {
	events := []learningmemory.Event{
		{Schema: learningmemory.EventSchema, EventID: "fail-1", EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, BenchmarkID: "b", TrialID: "t", QuestionID: "q", Pass: bp(false), LastCompletedStage: "T2_RULE_MICROGRAMMAR", FailureCode: "TEMPORAL_EXECUTION_INCOMPLETE", ScoreLayer: "T_TEMPORAL"},
		{Schema: learningmemory.EventSchema, EventID: "out-1", EventType: learningmemory.EventOutcome, EvidenceClass: learningmemory.EvidenceManual, CandidateID: "temporal-grammar-visible-r1", ParentEventIDs: []string{"a", "b"}, BeforeScore: fp(.30), AfterScore: fp(.72), Delta: fp(.42)},
		{Schema: learningmemory.EventSchema, EventID: "invalid-1", EventType: learningmemory.EventChange, EvidenceClass: learningmemory.EvidenceManual, CandidateID: "bad-render", ParentEventIDs: []string{"fail-1"}, ChangeSummary: "invalid specimen", Tags: []string{"invalid-specimen", "semantic-drift"}},
	}
	p := Derive(events)
	if p.Target != "EXECUTION_POLICY_COMPLIANCE" {
		t.Fatalf("target=%q", p.Target)
	}
	if len(p.Invariants) == 0 {
		t.Fatal("expected positive outcome invariant")
	}
	foundParity := false
	foundCrossModel := false
	for _, r := range p.Rules {
		if r.Kind == RuleRequire && r.Target == "SEMANTIC_PARITY_GATE" {
			foundParity = true
		}
		if r.Kind == RuleRequire && r.Target == "CROSS_MODEL_COMPATIBILITY_GATE" {
			foundCrossModel = true
		}
	}
	if !foundParity {
		t.Fatal("expected semantic parity requirement")
	}
	if !foundCrossModel {
		t.Fatal("expected cross-model compatibility requirement")
	}
	if len(p.AntiPatterns) == 0 || p.AntiPatterns[0].ID != "GENERATIVE_REWRITE_OF_EXACT_SEMANTICS" {
		t.Fatalf("anti-pattern missing: %#v", p.AntiPatterns)
	}
}
