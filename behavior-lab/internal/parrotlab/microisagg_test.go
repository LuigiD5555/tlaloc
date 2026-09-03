package parrotlab

import "testing"

func microISATestThresholds() Thresholds {
	return Thresholds{CapabilityClass: CapabilityClassCuts{
		StrongLowerCI: 0.85, UsableLowerCI: 0.70, WeakUpperCI: 0.70, UnusableUpperCI: 0.50,
	}}
}

func microISARecord(sub, dim, condition, capability, baseID string, correct bool) RunRecord {
	return RunRecord{
		Stage: StageMicroISAVisual, SubExperiment: sub, VariedDim: dim, Condition: condition,
		Capabilities: []string{capability}, BaseID: baseID, TaskFamily: "choice",
		Actual: Actual{Raw: "x"},
		Score:  Score{SemanticCorrect: correct, FormatValid: true, ContractSuccess: correct},
	}
}

func TestMicroISAClassificationUsesA1Only(t *testing.T) {
	var records []RunRecord
	// A1: READ_SHORT_TEXT strong (28/30 correct → Wilson-low above the usable cut).
	for index := 0; index < 30; index++ {
		records = append(records, microISARecord("A1", "", "canonical", "READ_SHORT_TEXT",
			"mi-a1-read_short_text-"+itoa(index), index < 28))
	}
	// A2: same capability but terrible at every rung — must NOT change the class.
	for _, rung := range []string{"chars=2", "chars=4", "chars=8", "chars=16", "chars=32"} {
		for index := 0; index < 10; index++ {
			records = append(records, microISARecord("A2", "visual_text_chars", rung, "READ_SHORT_TEXT",
				"mi-a2-visual_text_chars-"+itoa(index), false))
		}
	}
	result := buildMicroISA(records, microISATestThresholds())
	verdict := result.IntrinsicVerdict["READ_SHORT_TEXT"]
	if verdict.BaseStimuli != 30 || verdict.Observations != 30 {
		t.Fatalf("intrinsic should count A1 only: base %d obs %d", verdict.BaseStimuli, verdict.Observations)
	}
	if verdict.Class != "STRONG" && verdict.Class != "USABLE" {
		t.Fatalf("A1 18/20 should be STRONG/USABLE, got %s", verdict.Class)
	}
}

func TestMicroISALadderMaxSafeRung(t *testing.T) {
	var records []RunRecord
	// rungs 2 and 4 solid; rung 8 collapses on every base (significant drop).
	for baseIndex := 0; baseIndex < 12; baseIndex++ {
		base := "mi-a2-choice_width-" + itoa(baseIndex)
		records = append(records, microISARecord("A2", "choice_width", "width=2", "SELECT_ONE_OF_N", base, true))
		records = append(records, microISARecord("A2", "choice_width", "width=4", "SELECT_ONE_OF_N", base, true))
		records = append(records, microISARecord("A2", "choice_width", "width=8", "SELECT_ONE_OF_N", base, false))
	}
	result := buildMicroISA(records, microISATestThresholds())
	limit := result.Limits.ChoiceWidth
	if limit == nil || *limit != 4 {
		got := "nil"
		if limit != nil {
			got = itoa(*limit)
		}
		t.Fatalf("choice_width max safe rung: want 4, got %s", got)
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
