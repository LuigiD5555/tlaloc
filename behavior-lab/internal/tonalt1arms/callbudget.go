package tonalt1arms

import "fmt"

// dagStep is one node of a frozen TaskFamily DAG, as defined in
// tonal-origami's runtime/tonal/families.go. Only the Capability matters for
// call-budget derivation; LocalID is kept for readability/debugging.
type dagStep struct {
	LocalID    string
	Capability string
}

// acquireSteps reproduces runtime/tonal/families.go's acquire(role): the
// three-node LOCATE_REGION -> EXTRACT_NUMBER -> NORMALIZE chain shared by
// every operand role across all five frozen shapes.
func acquireSteps(role string) []dagStep {
	return []dagStep{
		{"locate_" + role, "LOCATE_REGION"},
		{"read_" + role, "EXTRACT_NUMBER"},
		{"norm_" + role, "NORMALIZE"},
	}
}

// FrozenShapeDAG reproduces the exact step list of one frozen TaskFamily
// (runtime/tonal/families.go, shapeReadAndCheck/shapeCompareTwoValues/
// shapeDifferenceThenVerify/shapeRatioOfDifference/shapeReconciliationChain).
// This is a one-time verified transcription of the frozen Go source (not a
// live import, since tonal.local/runtime depends on this module, not the
// reverse) -- ShapeDAGStepCounts below is checked against the frozen
// critical_path_depth node counts (4/7/9/14/22) recorded in D1/D2 as an
// independent cross-check that the transcription is faithful.
func FrozenShapeDAG(shape string) ([]dagStep, error) {
	switch shape {
	case "READ_AND_CHECK":
		steps := acquireSteps("A")
		steps = append(steps, dagStep{"check", "COMPARE_NUMBERS"})
		return steps, nil

	case "COMPARE_TWO_VALUES":
		steps := append(acquireSteps("A"), acquireSteps("B")...)
		steps = append(steps, dagStep{"cmp", "COMPARE_NUMBERS"})
		return steps, nil

	case "DIFFERENCE_THEN_VERIFY":
		steps := append(acquireSteps("A"), acquireSteps("B")...)
		steps = append(steps,
			dagStep{"diff", "ARITHMETIC"},
			dagStep{"norm_diff", "NORMALIZE"},
			dagStep{"verify", "VERIFY"},
		)
		return steps, nil

	case "RATIO_OF_DIFFERENCE":
		steps := acquireSteps("A")
		steps = append(steps, acquireSteps("B")...)
		steps = append(steps, acquireSteps("C")...)
		steps = append(steps,
			dagStep{"diff", "ARITHMETIC"},
			dagStep{"norm_diff", "NORMALIZE"},
			dagStep{"ratio", "ARITHMETIC"},
			dagStep{"norm_ratio", "NORMALIZE"},
			dagStep{"verify", "VERIFY"},
		)
		return steps, nil

	case "RECONCILIATION_CHAIN":
		steps := acquireSteps("A")
		steps = append(steps, acquireSteps("a")...)
		steps = append(steps, acquireSteps("B")...)
		steps = append(steps, acquireSteps("b")...)
		steps = append(steps,
			dagStep{"sub_A", "ARITHMETIC"},
			dagStep{"sub_B", "ARITHMETIC"},
			dagStep{"disagreement_pct", "ARITHMETIC"},
			dagStep{"norm_pct", "NORMALIZE"},
			dagStep{"fraction", "ARITHMETIC"},
			dagStep{"norm_fraction", "NORMALIZE"},
			dagStep{"tolerance_margin", "ARITHMETIC"},
			dagStep{"norm_margin", "NORMALIZE"},
			dagStep{"cmp_zero", "COMPARE_NUMBERS"},
			dagStep{"verify", "VERIFY"},
		)
		return steps, nil

	default:
		return nil, fmt.Errorf("tonalt1arms: unknown shape %q", shape)
	}
}

// ShapeDAGNodeCounts is the frozen total node count per shape, verified
// directly against runtime/tonal/families_test.go's wantNodeCount map
// (tonal-origami repo, branch claude/tonal-t1, line ~15-20): 4/7/9/14/22.
// natural_depth (the logical-composition label) is separately 2/4/6/8/12
// and not used here. Used only as an independent cross-check that
// FrozenShapeDAG's transcription below has the right node count per shape.
var ShapeDAGNodeCounts = map[string]int{
	"READ_AND_CHECK":         4,
	"COMPARE_TWO_VALUES":     7,
	"DIFFERENCE_THEN_VERIFY": 9,
	"RATIO_OF_DIFFERENCE":    14,
	"RECONCILIATION_CHAIN":   22,
}

// ArmBCallBudgetRow is one shape's call-budget breakdown for Arm B.
type ArmBCallBudgetRow struct {
	Family           string
	Workflows        int
	ExtractPerWF     int
	NormalizePerWF   int
	ComparePerWF     int
	ArithmeticPerWF  int
	OtherPerWF       int
	ParrotCallsPerWF int
	ParrotCallsTotal int
}

// DeriveArmBCallBudget mechanically derives Arm B's total Parrot call count
// by walking each of the five frozen shape DAGs and classifying every step
// against the frozen Arm-B policy's parrot_adapters set (EXTRACT_NUMBER,
// NORMALIZE, COMPARE_NUMBERS, ARITHMETIC) vs its deterministic_nodes set
// (LOCATE_REGION, CROP_REGION, VERIFY). workflowsPerShape is 12 for the
// frozen T1 allocation (60 workflows / 5 shapes).
func DeriveArmBCallBudget(policy *ArmBPolicy, workflowsPerShape int) ([]ArmBCallBudgetRow, int, error) {
	parrotCaps := make(map[string]bool, len(policy.ParrotAdapters))
	for cap := range policy.ParrotAdapters {
		parrotCaps[cap] = true
	}
	deterministic := make(map[string]bool, len(policy.DeterministicNodes))
	for _, cap := range policy.DeterministicNodes {
		deterministic[cap] = true
	}

	shapes := []string{
		"READ_AND_CHECK",
		"COMPARE_TWO_VALUES",
		"DIFFERENCE_THEN_VERIFY",
		"RATIO_OF_DIFFERENCE",
		"RECONCILIATION_CHAIN",
	}

	var rows []ArmBCallBudgetRow
	total := 0
	for _, shape := range shapes {
		steps, err := FrozenShapeDAG(shape)
		if err != nil {
			return nil, 0, err
		}
		if want, ok := ShapeDAGNodeCounts[shape]; ok && len(steps) != want {
			return nil, 0, fmt.Errorf("tonalt1arms: shape %s has %d transcribed nodes, want %d (cross-check against frozen critical_path_depth failed)", shape, len(steps), want)
		}

		row := ArmBCallBudgetRow{Family: shape, Workflows: workflowsPerShape}
		for _, step := range steps {
			switch {
			case step.Capability == "EXTRACT_NUMBER":
				row.ExtractPerWF++
			case step.Capability == "NORMALIZE":
				row.NormalizePerWF++
			case step.Capability == "COMPARE_NUMBERS":
				row.ComparePerWF++
			case step.Capability == "ARITHMETIC":
				row.ArithmeticPerWF++
			case deterministic[step.Capability]:
				// LOCATE_REGION, CROP_REGION, VERIFY: no Parrot call.
			case parrotCaps[step.Capability]:
				row.OtherPerWF++
			default:
				return nil, 0, fmt.Errorf("tonalt1arms: step %s capability %q is neither in frozen deterministic_nodes nor parrot_adapters -- Arm-B routing is underspecified for this node", step.LocalID, step.Capability)
			}
		}
		row.ParrotCallsPerWF = row.ExtractPerWF + row.NormalizePerWF + row.ComparePerWF + row.ArithmeticPerWF + row.OtherPerWF
		row.ParrotCallsTotal = row.ParrotCallsPerWF * workflowsPerShape
		total += row.ParrotCallsTotal
		rows = append(rows, row)
	}
	return rows, total, nil
}
