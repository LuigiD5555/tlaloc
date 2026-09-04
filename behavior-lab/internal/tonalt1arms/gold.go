package tonalt1arms

import (
	"fmt"
	"math"
)

const (
	// ReconciliationTolerance is the frozen tolerance for the RECONCILIATION_CHAIN shape.
	// Recovered from the gold-generation script's default constructor argument.
	ReconciliationTolerance = 0.05
)

// ComputeGold reproduces the exact frozen gold formula for a workflow.
// This is a self-check: if this doesn't match the frozen gold record,
// the experiment's arithmetic has not been understood correctly.
func ComputeGold(wf Workflow) (finalValue float64, status string, err error) {
	if len(wf.Operands) == 0 {
		return 0, "", fmt.Errorf("workflow %q has no operands", wf.WorkflowID)
	}

	switch wf.Shape {
	case "READ_AND_CHECK":
		if len(wf.Operands) != 1 {
			return 0, "", fmt.Errorf("READ_AND_CHECK must have 1 operand, got %d", len(wf.Operands))
		}
		return wf.Operands[0].NumericValue, "SUCCESS", nil

	case "COMPARE_TWO_VALUES":
		if len(wf.Operands) != 2 {
			return 0, "", fmt.Errorf("COMPARE_TWO_VALUES must have 2 operands, got %d", len(wf.Operands))
		}
		roleMap := makeRoleMap(wf.Operands)
		a, ok := roleMap["A"]
		if !ok {
			return 0, "", fmt.Errorf("COMPARE_TWO_VALUES missing role A")
		}
		b, ok := roleMap["B"]
		if !ok {
			return 0, "", fmt.Errorf("COMPARE_TWO_VALUES missing role B")
		}
		return a - b, "SUCCESS", nil

	case "DIFFERENCE_THEN_VERIFY":
		if len(wf.Operands) != 2 {
			return 0, "", fmt.Errorf("DIFFERENCE_THEN_VERIFY must have 2 operands, got %d", len(wf.Operands))
		}
		roleMap := makeRoleMap(wf.Operands)
		a, ok := roleMap["A"]
		if !ok {
			return 0, "", fmt.Errorf("DIFFERENCE_THEN_VERIFY missing role A")
		}
		b, ok := roleMap["B"]
		if !ok {
			return 0, "", fmt.Errorf("DIFFERENCE_THEN_VERIFY missing role B")
		}
		return a - b, "SUCCESS", nil

	case "RATIO_OF_DIFFERENCE":
		if len(wf.Operands) != 3 {
			return 0, "", fmt.Errorf("RATIO_OF_DIFFERENCE must have 3 operands, got %d", len(wf.Operands))
		}
		roleMap := makeRoleMap(wf.Operands)
		a, ok := roleMap["A"]
		if !ok {
			return 0, "", fmt.Errorf("RATIO_OF_DIFFERENCE missing role A")
		}
		b, ok := roleMap["B"]
		if !ok {
			return 0, "", fmt.Errorf("RATIO_OF_DIFFERENCE missing role B")
		}
		c, ok := roleMap["C"]
		if !ok {
			return 0, "", fmt.Errorf("RATIO_OF_DIFFERENCE missing role C")
		}
		diff := a - b
		if c == 0 {
			return 0, "INVALID_INPUT_DENOMINATOR_ZERO", nil
		}
		return diff / c, "SUCCESS", nil

	case "RECONCILIATION_CHAIN":
		if len(wf.Operands) != 4 {
			return 0, "", fmt.Errorf("RECONCILIATION_CHAIN must have 4 operands, got %d", len(wf.Operands))
		}
		roleMap := makeRoleMap(wf.Operands)
		a, ok := roleMap["A"]
		if !ok {
			return 0, "", fmt.Errorf("RECONCILIATION_CHAIN missing role A")
		}
		aLower, ok := roleMap["a"]
		if !ok {
			return 0, "", fmt.Errorf("RECONCILIATION_CHAIN missing role a")
		}
		b, ok := roleMap["B"]
		if !ok {
			return 0, "", fmt.Errorf("RECONCILIATION_CHAIN missing role B")
		}
		bLower, ok := roleMap["b"]
		if !ok {
			return 0, "", fmt.Errorf("RECONCILIATION_CHAIN missing role b")
		}

		subA := a - aLower
		subB := b - bLower
		avg := (math.Abs(subA) + math.Abs(subB)) / 2.0
		var pctDiff float64
		if avg == 0 {
			pctDiff = 0
		} else {
			pctDiff = math.Abs(subA-subB) / avg * 100
		}
		fraction := pctDiff / 100
		margin := fraction - ReconciliationTolerance
		return margin, "SUCCESS", nil

	default:
		return 0, "", fmt.Errorf("unknown shape: %q", wf.Shape)
	}
}

// makeRoleMap builds a map from operand role to its numeric value.
func makeRoleMap(operands []Operand) map[string]float64 {
	m := make(map[string]float64)
	for _, op := range operands {
		m[op.Role] = op.NumericValue
	}
	return m
}
