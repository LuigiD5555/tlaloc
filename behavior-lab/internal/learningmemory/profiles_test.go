package learningmemory

import (
	"math"
	"testing"
)

func boolPointer(value bool) *bool { return &value }

func profileObservation(model, condition, question string, pass bool) Event {
	return Event{
		Schema:        EventSchema,
		EventType:     EventObservation,
		EvidenceClass: EvidenceRealModel,
		BenchmarkID:   "bench",
		SpecimenID:    "specimen",
		QuestionID:    question,
		ScoreLayer:    "semantic",
		ModelID:       model,
		Condition:     condition,
		Pass:          boolPointer(pass),
	}
}

func closeEnough(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestBuildEmpiricalProfilesMeasuresFailureComplementarity(t *testing.T) {
	events := []Event{
		profileObservation("A", "protocol-a", "q1", true),
		profileObservation("A", "protocol-a", "q2", true),
		profileObservation("A", "protocol-a", "q3", false),
		profileObservation("A", "protocol-a", "q4", false),
		profileObservation("B", "protocol-b", "q1", true),
		profileObservation("B", "protocol-b", "q2", false),
		profileObservation("B", "protocol-b", "q3", true),
		profileObservation("B", "protocol-b", "q4", false),
	}

	models, pairs := BuildEmpiricalProfiles(events)
	if len(models) != 2 {
		t.Fatalf("model profiles=%+v", models)
	}
	if models[0].ModelID != "A" || models[0].Cases != 4 || models[0].Passes != 2 || !closeEnough(models[0].Accuracy, 0.5) {
		t.Fatalf("model A=%+v", models[0])
	}
	if len(pairs) != 1 {
		t.Fatalf("pairs=%+v", pairs)
	}
	pair := pairs[0]
	if pair.SharedCases != 4 || pair.BothPass != 1 || pair.BothFail != 1 || pair.ARecoversB != 1 || pair.BRecoversA != 1 {
		t.Fatalf("pair counts=%+v", pair)
	}
	if pair.FailureUnion != 3 || !closeEnough(pair.FailureOverlap, 1.0/3.0) || !closeEnough(pair.Complementarity, 2.0/3.0) || !closeEnough(pair.OracleSuccess, 0.75) {
		t.Fatalf("pair metrics=%+v", pair)
	}
}

func TestEmpiricalProfilesKeepConditionsDistinct(t *testing.T) {
	events := []Event{
		profileObservation("same-model", "prompt-a", "q1", true),
		profileObservation("same-model", "prompt-a", "q2", false),
		profileObservation("same-model", "prompt-b", "q1", true),
		profileObservation("same-model", "prompt-b", "q2", true),
	}
	models, pairs := BuildEmpiricalProfiles(events)
	if len(models) != 2 {
		t.Fatalf("conditions must create separate profiles: %+v", models)
	}
	if models[0].Condition != "prompt-a" || models[1].Condition != "prompt-b" {
		t.Fatalf("conditions=%+v", models)
	}
	if len(pairs) != 1 || pairs[0].SharedCases != 2 || pairs[0].BRecoversA != 1 {
		t.Fatalf("pair=%+v", pairs)
	}
}

func TestDuplicateObservationsAreFailureDominantAndOrderIndependent(t *testing.T) {
	pass := profileObservation("A", "cfg", "q1", true)
	fail := profileObservation("A", "cfg", "q1", false)
	first, _ := BuildEmpiricalProfiles([]Event{pass, fail})
	second, _ := BuildEmpiricalProfiles([]Event{fail, pass})
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("profiles first=%+v second=%+v", first, second)
	}
	if first[0].Passes != 0 || first[0].Failures != 1 || second[0] != first[0] {
		t.Fatalf("duplicate reduction must be deterministic: first=%+v second=%+v", first[0], second[0])
	}
}

func TestEmpiricalProfilesIgnoreSyntheticAndUnaddressableCases(t *testing.T) {
	real := profileObservation("A", "cfg", "q1", true)
	synthetic := profileObservation("B", "cfg", "q1", false)
	synthetic.EvidenceClass = EvidenceSynthetic
	unaddressable := profileObservation("C", "cfg", "", false)
	unaddressable.BenchmarkID = ""
	unaddressable.SpecimenID = ""
	models, pairs := BuildEmpiricalProfiles([]Event{real, synthetic, unaddressable})
	if len(models) != 1 || models[0].ModelID != "A" || len(pairs) != 0 {
		t.Fatalf("models=%+v pairs=%+v", models, pairs)
	}
}

func TestBuildSummaryIncludesEmpiricalProfiles(t *testing.T) {
	events := []Event{
		profileObservation("A", "cfg-a", "q1", false),
		profileObservation("B", "cfg-b", "q1", true),
	}
	summary := BuildSummary("memory", events)
	if len(summary.ModelProfiles) != 2 || len(summary.PairwiseProfiles) != 1 {
		t.Fatalf("summary profiles=%+v pairwise=%+v", summary.ModelProfiles, summary.PairwiseProfiles)
	}
	if summary.PairwiseProfiles[0].BRecoversA != 1 || !closeEnough(summary.PairwiseProfiles[0].OracleSuccess, 1) {
		t.Fatalf("summary pair=%+v", summary.PairwiseProfiles[0])
	}
}

func TestNoFailuresDoesNotInventComplementarity(t *testing.T) {
	events := []Event{
		profileObservation("A", "cfg-a", "q1", true),
		profileObservation("B", "cfg-b", "q1", true),
	}
	_, pairs := BuildEmpiricalProfiles(events)
	if len(pairs) != 1 {
		t.Fatalf("pairs=%+v", pairs)
	}
	if pairs[0].FailureUnion != 0 || pairs[0].FailureOverlap != 0 || pairs[0].Complementarity != 0 || pairs[0].OracleSuccess != 1 {
		t.Fatalf("perfect pair should not fabricate diversity: %+v", pairs[0])
	}
}
