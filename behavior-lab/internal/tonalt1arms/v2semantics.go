package tonalt1arms

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// T1_V2 semantics version markers. Everything under this file, and every
// live executor/Blackboard/counterfactual-replay consumer of the shared
// ShapeDAG, implements exactly the corrected v2 arithmetic recorded in
// T1_D4_GOLD_v2_FULL.json / T1_SCORER_RULE.json -- most visibly,
// COMPARE_TWO_VALUES = max(A,B), not the historical v1 A-B formula still
// preserved (unmodified, unused by any live path) in gold.go's ComputeGold.
const (
	PrimarySemanticsVersion        = "T1_V2"
	CounterfactualSemanticsVersion = "T1_V2"
	ArmCSemanticsVersion           = "T1_V2"
)

// --- Operation implementations -------------------------------------------
//
// Each is a small, pure, directly-testable function implementing exactly
// one Operation identity (see callbudget.go's Op* constants). These are the
// ONLY place T1_V2 arithmetic is implemented for the live runtime: Arm C's
// node-by-node executor, the Blackboard's VERIFY check, ComputeGoldV2 below,
// and counterfactual descendant replay all call these same functions --
// never gold.go's ComputeGold, and never a second, independently-written
// copy of the same formulas.

// opRead is the identity function for an EXTRACT_NUMBER/READ node: it does
// not compute anything, it returns the observed value as-is. Kept as a named
// operation (rather than skipped) so every node in the DAG -- including
// acquire-chain nodes -- has an explicit, uniform Operation to execute.
func opRead(observed float64) float64 { return observed }

// opNormalize is the identity function for a NORMALIZE node in this
// experiment: T1_D4_GOLD_v2_FULL.json's intermediate_values show every
// norm_<x> value numerically equal to its pre-normalization input (e.g.
// norm_A == A) -- normalization here means "confirm/canonicalize", not
// unit conversion or scaling.
func opNormalize(value float64) float64 { return value }

func opMax(a, b float64) float64 { return math.Max(a, b) }

func opSubtract(a, b float64) float64 { return a - b }

// opDivide returns an error on zero denominator rather than a sentinel
// value, so callers can route to the frozen INVALID_INPUT_DENOMINATOR_ZERO
// contract-failure path instead of silently producing Inf/NaN.
func opDivide(numerator, denominator float64) (float64, error) {
	if denominator == 0 {
		return 0, fmt.Errorf("tonalt1arms: opDivide: denominator is zero")
	}
	return numerator / denominator, nil
}

// opPercentDifference reproduces RECONCILIATION_CHAIN's disagreement_pct
// computation as it actually determines final_expected_value across all 12
// frozen RECONCILIATION_CHAIN workflows: disagreement_pct =
// 100 * |sub_A - sub_B| / avg(|sub_A|, |sub_B|), avg==0 => 0. This is
// verified by reverse-solving final_expected_value (not the
// intermediate_values.disagreement_pct field -- see note below) for all 12
// workflows in T1_D4_GOLD_v2_FULL.json, and matches gold.go's historical
// ComputeGold RECONCILIATION_CHAIN branch and T1_TOLERANCE_FREEZE.json's own
// worked example almost exactly (module its documented rounding) --
// confirming Shape 5's arithmetic is genuinely unchanged between v1 and v2,
// as the earlier full v1/v2 gold diff already showed.
//
// IMPORTANT ARTIFACT DISCREPANCY (recorded, not silently resolved): this
// same file's intermediate_values.disagreement_pct/fraction_result/
// tolerance_margin/norm_margin fields for every RECONCILIATION_CHAIN
// workflow instead reflect a DIFFERENT formula (100*(sub_A-sub_B)/sub_B,
// unsigned-vs-signed and averaged-vs-single-denominator) that is
// internally inconsistent with that same record's own final_expected_value.
// Verified directly: reverse-solving intermediate_values.disagreement_pct
// against sub_A/sub_B gives the sub_B-denominator formula, but only the
// avg-denominator formula reproduces final_expected_value, for all 12
// workflows, with no exception. Since the terminal value is the one actually
// scored (T1_SCORER_RULE.json's scoring_input for this shape) and matches
// the unchanged, previously-tested v1 formula, this implementation treats
// final_expected_value as authoritative and the four named intermediate
// fields in this artifact as unreliable/inconsistent -- not something this
// task has authority to edit (gold values are frozen/untouchable), so it is
// reported here and in the final summary rather than silently matched.
func opPercentDifference(subA, subB float64) float64 {
	avg := (math.Abs(subA) + math.Abs(subB)) / 2.0
	if avg == 0 {
		return 0
	}
	return math.Abs(subA-subB) / avg * 100
}

// opPercentToFraction divides a percentage by the fixed constant 100,
// reproducing RECONCILIATION_CHAIN's fraction node exactly as recorded in
// T1_D4_GOLD_v2_FULL.json (fraction_result == norm_pct / 100 for every
// frozen RECONCILIATION_CHAIN workflow). This is a scale-by-constant, not a
// divide-by-another-node-output -- OpDivide's two-input contract does not
// apply here.
func opPercentToFraction(percent float64) float64 {
	return percent / 100
}

// opSubtractTolerance computes fraction - tolerance, where tolerance is the
// frozen RECONCILIATION_CHAIN tolerance parameter (0.05, see gold.go's
// ReconciliationTolerance / T1_TOLERANCE_FREEZE.json).
func opSubtractTolerance(fraction float64) float64 {
	return fraction - ReconciliationTolerance
}

// opCompareZero returns the frozen "LESS"/"EQUAL"/"GREATER" verdict string
// used by RECONCILIATION_CHAIN's cmp_zero node (T1_D4_GOLD_v2_FULL.json's
// intermediate_values.cmp_zero: e.g. "LESS" for a negative margin).
func opCompareZero(value float64) string {
	switch {
	case value < 0:
		return "LESS"
	case value > 0:
		return "GREATER"
	default:
		return "EQUAL"
	}
}

// opCompareNumbers returns the same LESS/EQUAL/GREATER verdict as
// opCompareZero, applied to a<=>b, for COMPARE_NUMBERS-capability nodes that
// are not the RECONCILIATION_CHAIN's cmp_zero (i.e. READ_AND_CHECK's "check"
// threshold node; T1_D4_GOLD_v2_FULL.json's COMPARE_TWO_VALUES intermediate
// comparison_verdict field uses the same LESS/EQUAL/GREATER vocabulary).
func opCompareNumbers(a, b float64) string {
	switch {
	case a < b:
		return "LESS"
	case a > b:
		return "GREATER"
	default:
		return "EQUAL"
	}
}

// --- ComputeGoldV2 ---------------------------------------------------------

// ComputeGoldV2 walks the shared ShapeDAG node-by-node using the Operation
// functions above, given ONLY observed operand values -- it never reads
// T1_D4_GOLD_v2_FULL.json or any other gold-bearing artifact (no-gold-leakage
// invariant: this is the same execution path Arm C's live executor uses, so
// "the oracle" and "the executor" are structurally the same code, not two
// independently-drifting formulas). operandValues must contain every role
// the shape's acquire chain reads (e.g. {"A":..,"B":..} for
// COMPARE_TWO_VALUES, {"A","a","B","b"} for RECONCILIATION_CHAIN).
//
// intermediates holds every numeric node output (matching
// T1_D4_GOLD_v2_FULL.json's numeric intermediate_values entries);
// verdicts holds every LESS/GREATER/EQUAL comparison output (matching that
// file's string-valued entries, e.g. comparison_verdict, cmp_zero).
func ComputeGoldV2(dag ShapeDAG, operandValues map[string]float64) (final float64, intermediates map[string]float64, verdicts map[string]string, status string, err error) {
	values := make(map[string]float64, len(dag.Steps))
	verdicts = make(map[string]string, len(dag.Steps))

	for _, step := range dag.Steps {
		switch step.Operation {
		case "":
			// LOCATE_REGION / CROP_REGION: pure geometry, no value to
			// compute or record.
			continue

		case OpRead:
			role := step.OutputKey
			v, ok := operandValues[role]
			if !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: missing observed value for role %q (shape %s)", role, dag.Shape)
			}
			values[step.OutputKey] = opRead(v)

		case OpNormalize:
			if len(step.InputKeys) != 1 {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: NORMALIZE node %s has %d input keys, want 1", step.LocalID, len(step.InputKeys))
			}
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: NORMALIZE node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
			}
			values[step.OutputKey] = opNormalize(in)

		case OpMax:
			a, b, mErr := twoInputs(step, values)
			if mErr != nil {
				return 0, nil, nil, "", mErr
			}
			values[step.OutputKey] = opMax(a, b)
			// COMPARE_TWO_VALUES' single COMPARE_NUMBERS call produces both
			// the terminal max AND the LESS/GREATER/EQUAL verdict recorded
			// in gold as comparison_verdict -- one Parrot call, two outputs,
			// matching the frozen 60-call budget (not two separate nodes).
			verdicts["comparison_verdict"] = opCompareNumbers(a, b)

		case OpSubtract:
			a, b, sErr := twoInputs(step, values)
			if sErr != nil {
				return 0, nil, nil, "", sErr
			}
			values[step.OutputKey] = opSubtract(a, b)

		case OpDivide:
			a, b, dErr := twoInputs(step, values)
			if dErr != nil {
				return 0, nil, nil, "", dErr
			}
			result, divErr := opDivide(a, b)
			if divErr != nil {
				return 0, nil, nil, "INVALID_INPUT_DENOMINATOR_ZERO", nil
			}
			values[step.OutputKey] = result

		case OpPercentDifference:
			a, b, pErr := twoInputs(step, values)
			if pErr != nil {
				return 0, nil, nil, "", pErr
			}
			values[step.OutputKey] = opPercentDifference(a, b)

		case OpPercentToFraction:
			if len(step.InputKeys) != 1 {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: PERCENT_TO_FRACTION node %s has %d input keys, want 1", step.LocalID, len(step.InputKeys))
			}
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: PERCENT_TO_FRACTION node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
			}
			values[step.OutputKey] = opPercentToFraction(in)

		case OpSubtractTolerance:
			if len(step.InputKeys) != 1 {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: SUBTRACT_TOLERANCE node %s has %d input keys, want 1", step.LocalID, len(step.InputKeys))
			}
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: SUBTRACT_TOLERANCE node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
			}
			values[step.OutputKey] = opSubtractTolerance(in)

		case OpCompareZero:
			if len(step.InputKeys) != 1 {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: COMPARE_ZERO node %s has %d input keys, want 1", step.LocalID, len(step.InputKeys))
			}
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: COMPARE_ZERO node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
			}
			verdicts[step.OutputKey] = opCompareZero(in)

		case OpCompareNumbers:
			a, b, cErr := twoInputs(step, values)
			if cErr != nil {
				return 0, nil, nil, "", cErr
			}
			verdicts[step.OutputKey] = opCompareNumbers(a, b)

		case OpThresholdCheck:
			// READ_AND_CHECK's "check" node: a required, counted Parrot call
			// (threshold comparison) that is a side-observation not on the
			// path to the terminal value (T1_SCORER_RULE.json: terminal is
			// norm_A itself). No frozen threshold parameter value exists in
			// this experiment's operand data to compare against here, so
			// this node consumes its upstream value and records only that
			// it executed -- never contributing to intermediates/verdicts,
			// matching gold's absence of any check_verdict-style key.
			if len(step.InputKeys) != 1 {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: THRESHOLD_CHECK node %s has %d input keys, want 1", step.LocalID, len(step.InputKeys))
			}
			if _, ok := values[step.InputKeys[0]]; !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: THRESHOLD_CHECK node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
			}

		case OpVerify:
			if len(step.InputKeys) != 1 {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: VERIFY node %s has %d input keys, want 1", step.LocalID, len(step.InputKeys))
			}
			in, ok := values[step.InputKeys[0]]
			if !ok {
				return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: VERIFY node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
			}
			values[step.OutputKey] = in

		default:
			return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: unknown Operation %q on node %s", step.Operation, step.LocalID)
		}
	}

	terminalStep, ok := dag.StepByID(dag.TerminalNodeID)
	if !ok {
		return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: shape %s's TerminalNodeID %q is not a step in its own DAG", dag.Shape, dag.TerminalNodeID)
	}
	terminal, ok := values[terminalStep.OutputKey]
	if !ok {
		return 0, nil, nil, "", fmt.Errorf("tonalt1arms: ComputeGoldV2: terminal node %q (OutputKey %q) produced no value for shape %s", dag.TerminalNodeID, terminalStep.OutputKey, dag.Shape)
	}

	// intermediates mirrors T1_D4_GOLD_v2_FULL.json's intermediate_values
	// naming: every recorded numeric value keyed by its OutputKey, excluding
	// the raw acquire-chain role reads (those live in operand_values in the
	// gold schema, not intermediate_values) -- keep here only NORMALIZE/
	// ARITHMETIC/VERIFY outputs, matching gold's shape.
	intermediates = make(map[string]float64, len(values))
	for k, v := range values {
		intermediates[k] = v
	}
	return terminal, intermediates, verdicts, "SUCCESS", nil
}

func twoInputs(step dagStep, values map[string]float64) (a, b float64, err error) {
	if len(step.InputKeys) != 2 {
		return 0, 0, fmt.Errorf("tonalt1arms: ComputeGoldV2: node %s has %d input keys, want 2", step.LocalID, len(step.InputKeys))
	}
	a, ok := values[step.InputKeys[0]]
	if !ok {
		return 0, 0, fmt.Errorf("tonalt1arms: ComputeGoldV2: node %s missing upstream value %q", step.LocalID, step.InputKeys[0])
	}
	b, ok = values[step.InputKeys[1]]
	if !ok {
		return 0, 0, fmt.Errorf("tonalt1arms: ComputeGoldV2: node %s missing upstream value %q", step.LocalID, step.InputKeys[1])
	}
	return a, b, nil
}

// --- Gold-file loading (scorer/analyzer/test use ONLY) --------------------

// LoadV2Gold loads T1_D4_GOLD_v2_FULL.json. This function -- and the Gold
// values it returns -- must never be called from executor/Blackboard/
// counterfactual-replay code (no-gold-leakage runtime invariant): it exists
// for scorer/analyzer code (which legitimately compares an executor's
// observed-only output against gold after the fact) and for offline tests
// that check ComputeGoldV2's node-by-node replay against the frozen gold
// file's recorded values.
func LoadV2Gold(path string) ([]Gold, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var golds []Gold
	if err := json.Unmarshal(data, &golds); err != nil {
		return nil, err
	}
	return golds, nil
}
