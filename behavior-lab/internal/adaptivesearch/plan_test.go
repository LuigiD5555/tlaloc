package adaptivesearch

import (
	"math"
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

func boolp(v bool) *bool        { return &v }
func floatp(v float64) *float64 { return &v }

func TestRealT2FailuresPrioritizeT2MutationFamilies(t *testing.T) {
	events := []learningmemory.Event{
		{EventID: "e1", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, ModelID: "m1", SpecimenID: "s1", QuestionID: "Q3", ScoreLayer: "S_SEMANTIC", Pass: boolp(false), LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"},
		{EventID: "e2", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, ModelID: "m2", SpecimenID: "s1", QuestionID: "Q4", ScoreLayer: "S_SEMANTIC", Pass: boolp(false), LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"},
		{EventID: "e3", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, ModelID: "m1", SpecimenID: "s1", QuestionID: "Q1", ScoreLayer: "P_PERCEPTION", Pass: boolp(true)},
	}
	plan := BuildPlan("/memory", events)
	if !plan.Adaptive {
		t.Fatal("expected adaptive plan")
	}
	if plan.NextDebugTarget != "T2_NAVIGATION" {
		t.Fatalf("target=%s", plan.NextDebugTarget)
	}
	if len(plan.MutationPriorities) == 0 || plan.MutationPriorities[0].Kind != visualsearch.MutationLayout {
		t.Fatalf("top priority=%+v", plan.MutationPriorities)
	}
	if len(plan.ParentEvidenceIDs) != 2 {
		t.Fatalf("parents=%v", plan.ParentEvidenceIDs)
	}
	sum := 0.0
	for _, p := range plan.MutationPriorities {
		sum += p.Weight
		if p.Weight <= 0 {
			t.Fatalf("exploration floor missing for %s", p.Kind)
		}
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("weights sum=%f", sum)
	}
}

func TestSyntheticFailureDoesNotDriveAdaptiveFocus(t *testing.T) {
	events := []learningmemory.Event{{EventID: "s1", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceSynthetic, ModelID: "SYNTHETIC", QuestionID: "Q3", ScoreLayer: "S_SEMANTIC", Pass: boolp(false), LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"}}
	plan := BuildPlan("/memory", events)
	if plan.Adaptive {
		t.Fatal("synthetic evidence must not drive empirical adaptive focus")
	}
	if plan.NextDebugTarget != "" {
		t.Fatalf("unexpected target %q", plan.NextDebugTarget)
	}
}

func TestHistoricalSignalIsBounded(t *testing.T) {
	events := []learningmemory.Event{
		{EventID: "e1", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, ModelID: "m", SpecimenID: "s", QuestionID: "Q3", ScoreLayer: "S_SEMANTIC", Pass: boolp(false), LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"},
		{Schema: learningmemory.EventSchema, EventType: learningmemory.EventChange, EvidenceClass: learningmemory.EvidenceManual, CandidateID: "layout-win", ParentEventIDs: []string{"e1"}, ChangeSummary: "layout", Tags: []string{"mutation:LAYOUT"}},
		{Schema: learningmemory.EventSchema, EventType: learningmemory.EventOutcome, EvidenceClass: learningmemory.EvidenceManual, CandidateID: "layout-win", ParentEventIDs: []string{"change", "post"}, BeforeScore: floatp(0), AfterScore: floatp(1), Delta: floatp(1)},
	}
	plan := BuildPlan("/memory", events)
	if len(plan.HistoricalSignals) != 1 {
		t.Fatalf("signals=%+v", plan.HistoricalSignals)
	}
	if plan.HistoricalSignals[0].Adjustment != 0.25 {
		t.Fatalf("adjustment=%f", plan.HistoricalSignals[0].Adjustment)
	}
}

func TestPrioritizeQueueDoesNotClaimPromotion(t *testing.T) {
	events := []learningmemory.Event{{EventID: "e1", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, ModelID: "m", SpecimenID: "s", QuestionID: "Q3", ScoreLayer: "S_SEMANTIC", Pass: boolp(false), LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"}}
	plan := BuildPlan("/memory", events)
	candidates := []visualsearch.Candidate{
		{ID: "temporal", Mutations: []visualsearch.Mutation{{Kind: visualsearch.MutationTemporalStructure, Target: "timeline", Value: "x", Experimental: true}}},
		{ID: "layout", Mutations: []visualsearch.Mutation{{Kind: visualsearch.MutationLayout, Target: "T1_TO_T2", Value: "anchor", Experimental: true}}},
	}
	q := Prioritize(plan, candidates, 0)
	if len(q.CandidateOrder) != 2 || q.CandidateOrder[0].CandidateID != "layout" {
		t.Fatalf("order=%+v", q.CandidateOrder)
	}
	if q.Authority != "SEARCH_PRIORITY_ONLY_FINAL_VISUAL_TOURNAMENT_REMAINS_EVIDENCE_GATED" {
		t.Fatalf("authority=%s", q.Authority)
	}
}

func TestChangeAttemptsLinkBackToFailureEvidence(t *testing.T) {
	events := []learningmemory.Event{{EventID: "e1", Schema: learningmemory.EventSchema, EventType: learningmemory.EventObservation, EvidenceClass: learningmemory.EvidenceRealModel, ModelID: "m", SpecimenID: "s", QuestionID: "Q3", ScoreLayer: "S_SEMANTIC", Pass: boolp(false), LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"}}
	plan := BuildPlan("/memory", events)
	candidates := []visualsearch.Candidate{{ID: "layout", Mutations: []visualsearch.Mutation{{Kind: visualsearch.MutationLayout, Target: "T1_TO_T2", Value: "anchor", Experimental: true}}}}
	q := Prioritize(plan, candidates, 1)
	attempts := ChangeAttemptEvents(q, candidates)
	if len(attempts) != 1 {
		t.Fatalf("attempts=%d", len(attempts))
	}
	if len(attempts[0].ParentEventIDs) != 1 || attempts[0].ParentEventIDs[0] != "e1" {
		t.Fatalf("parents=%v", attempts[0].ParentEventIDs)
	}
	found := false
	for _, tag := range attempts[0].Tags {
		if tag == "mutation:LAYOUT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tags=%v", attempts[0].Tags)
	}
}
