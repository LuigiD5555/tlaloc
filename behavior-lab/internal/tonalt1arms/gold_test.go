package tonalt1arms

import (
	"math"
	"testing"
)

func TestComputeGold_ReadAndCheck(t *testing.T) {
	wf := Workflow{
		WorkflowID: "test-rea",
		Shape:      "READ_AND_CHECK",
		Operands: []Operand{
			{Role: "A", NumericValue: 95},
		},
	}
	final, status, err := ComputeGold(wf)
	if err != nil {
		t.Fatalf("ComputeGold failed: %v", err)
	}
	if final != 95 || status != "SUCCESS" {
		t.Errorf("expected (95, SUCCESS), got (%v, %s)", final, status)
	}
}

func TestComputeGold_CompareTwoValues(t *testing.T) {
	wf := Workflow{
		WorkflowID: "test-com",
		Shape:      "COMPARE_TWO_VALUES",
		Operands: []Operand{
			{Role: "A", NumericValue: 60},
			{Role: "B", NumericValue: 420},
		},
	}
	final, status, err := ComputeGold(wf)
	if err != nil {
		t.Fatalf("ComputeGold failed: %v", err)
	}
	if final != -360 || status != "SUCCESS" {
		t.Errorf("expected (-360, SUCCESS), got (%v, %s)", final, status)
	}
}

func TestComputeGold_RatioOfDifference(t *testing.T) {
	wf := Workflow{
		WorkflowID: "test-rat",
		Shape:      "RATIO_OF_DIFFERENCE",
		Operands: []Operand{
			{Role: "A", NumericValue: 0.5},
			{Role: "B", NumericValue: 420},
			{Role: "C", NumericValue: 0.85},
		},
	}
	final, status, err := ComputeGold(wf)
	if err != nil {
		t.Fatalf("ComputeGold failed: %v", err)
	}
	expected := -493.5294117647059
	if math.Abs(final-expected) > 1e-10 || status != "SUCCESS" {
		t.Errorf("expected (%v, SUCCESS), got (%v, %s)", expected, final, status)
	}
}

func TestComputeGold_ReconciliationChain(t *testing.T) {
	// tonal-t1-workflow-REC-49: A=4.8, a=1, B=420, b=45 → 1.9098732840549102
	wf := Workflow{
		WorkflowID: "test-rec",
		Shape:      "RECONCILIATION_CHAIN",
		Operands: []Operand{
			{Role: "A", NumericValue: 4.8},
			{Role: "a", NumericValue: 1},
			{Role: "B", NumericValue: 420},
			{Role: "b", NumericValue: 45},
		},
	}
	final, status, err := ComputeGold(wf)
	if err != nil {
		t.Fatalf("ComputeGold failed: %v", err)
	}
	expected := 1.9098732840549102
	if math.Abs(final-expected) > 1e-10 || status != "SUCCESS" {
		t.Errorf("expected (%v, SUCCESS), got (%v, %s)", expected, final, status)
	}
}

func TestScoreResult_Integer(t *testing.T) {
	gold := Gold{FinalExpectedValue: 95}

	// Exact match
	semantic, exact, failed := ScoreResult(95, gold)
	if !semantic || !exact || failed {
		t.Errorf("exact match: expected (true, true, false), got (%v, %v, %v)", semantic, exact, failed)
	}

	// Off by epsilon but within tolerance (should still be exact for integers)
	semantic, exact, failed = ScoreResult(95.0000001, gold)
	if semantic || exact || failed {
		t.Errorf("epsilon off for integer: expected (false, false, false), got (%v, %v, %v)", semantic, exact, failed)
	}
}

func TestScoreResult_NonInteger(t *testing.T) {
	gold := Gold{FinalExpectedValue: 1.9098732840549102}

	// Exact match
	semantic, exact, failed := ScoreResult(1.9098732840549102, gold)
	if !semantic || !exact || failed {
		t.Errorf("exact match: expected (true, true, false), got (%v, %v, %v)", semantic, exact, failed)
	}

	// Within relative tolerance (1e-4)
	predicted := gold.FinalExpectedValue * (1 + 1e-5) // 0.001% off
	semantic, exact, failed = ScoreResult(predicted, gold)
	if !semantic || exact || failed {
		t.Errorf("within tolerance: expected (true, false, false), got (%v, %v, %v)", semantic, exact, failed)
	}
}

func TestScoreResult_ParseFailure(t *testing.T) {
	gold := Gold{FinalExpectedValue: 95}

	semantic, exact, failed := ScoreResult(math.NaN(), gold)
	if semantic || exact || !failed {
		t.Errorf("NaN: expected (false, false, true), got (%v, %v, %v)", semantic, exact, failed)
	}

	semantic, exact, failed = ScoreResult(math.Inf(1), gold)
	if semantic || exact || !failed {
		t.Errorf("Inf: expected (false, false, true), got (%v, %v, %v)", semantic, exact, failed)
	}
}
