package tonalt1arms

import "fmt"

// Frozen operation identities. Every dagStep whose Capability requires
// generative or arithmetic work carries exactly one of these, naming the
// SPECIFIC computation it performs -- not just which capability class
// (Parrot-eligible vs deterministic) it belongs to. This is what lets Arm C
// (and counterfactual replay) execute the shared DAG node-by-node without
// ever calling or mirroring the historical v1 ComputeGold: each node's
// Operation has exactly one small, pure, directly-testable implementation
// (see v2semantics.go's op* functions), used identically by every consumer
// of the shared DAG.
//
// These identities and the shape each applies to are read directly off
// T1_D4_GOLD_v2_FULL.json's intermediate_values naming (opRead/opNormalize
// for the acquire chain's read_<role>/norm_<role> steps; the shape-specific
// arithmetic chain nodes below), not invented ad hoc.
const (
	OpRead              = "READ"
	OpNormalize         = "NORMALIZE"
	OpMax               = "MAX"
	OpSubtract          = "SUBTRACT"
	OpDivide            = "DIVIDE"
	OpPercentDifference = "PERCENT_DIFFERENCE"
	OpPercentToFraction = "PERCENT_TO_FRACTION" // RECONCILIATION_CHAIN's fraction node: divide-by-100 (fixed constant, not a second DAG-node input) turning a percentage into a fraction
	OpSubtractTolerance = "SUBTRACT_TOLERANCE"
	OpCompareZero       = "COMPARE_ZERO"
	OpCompareNumbers    = "COMPARE_NUMBERS" // COMPARE_TWO_VALUES-style two-operand comparison (kept for adapter routing/prompt-template lookup parity with the ARM_B_POLICY.json capability name; not used directly by any frozen shape below -- OpMax already produces COMPARE_TWO_VALUES' verdict)
	OpThresholdCheck    = "THRESHOLD_CHECK" // READ_AND_CHECK's "check" node: compares the single read value against a frozen threshold parameter, not against a second operand -- a required, counted Parrot call that is a side-observation, not on the path to the terminal value
	OpVerify            = "VERIFY"
)

// dagStep is one node of a frozen TaskFamily DAG, as defined in
// tonal-origami's runtime/tonal/families.go. Capability is the node's
// capability class (what callbudget derivation and Arm-B/C routing key off
// of); Operation is the specific computation this node performs, used by the
// v2 executors/replay to actually run the node. DependsOn/InputKeys/OutputKey
// give the shared DAG real edges and I/O naming so it can be walked (not
// just counted) by Arm B, Arm C, counterfactual descendant closure, and
// call-budget derivation alike -- one representation, several consumers.
type dagStep struct {
	LocalID    string
	Capability string
	Operation  string
	DependsOn  []string
	InputKeys  []string
	OutputKey  string
}

// acquireSteps reproduces runtime/tonal/families.go's acquire(role): the
// three-node LOCATE_REGION -> EXTRACT_NUMBER -> NORMALIZE chain shared by
// every operand role across all five frozen shapes. EXTRACT_NUMBER's
// Operation is READ (it reads/observes the operand's value; LOCATE_REGION
// has no Operation of its own -- it is purely deterministic geometry, not an
// arithmetic/generative step).
func acquireSteps(role string) []dagStep {
	locate := "locate_" + role
	read := "read_" + role
	norm := "norm_" + role
	return []dagStep{
		{LocalID: locate, Capability: "LOCATE_REGION"},
		{LocalID: read, Capability: "EXTRACT_NUMBER", Operation: OpRead, DependsOn: []string{locate}, OutputKey: role},
		{LocalID: norm, Capability: "NORMALIZE", Operation: OpNormalize, DependsOn: []string{read}, InputKeys: []string{role}, OutputKey: "norm_" + role},
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
//
// Operation assignments implement T1_V2 semantics throughout (see
// v2semantics.go): COMPARE_TWO_VALUES's terminal node is OpMax
// (max(A,B)), matching T1_D4_GOLD_v2_FULL.json/T1_SCORER_RULE.json's
// corrected rule -- NOT the historical v1 A-B formula, which lives on,
// untouched and unused by this DAG, only in the old tonalt1arms.ComputeGold.
// All other four shapes' arithmetic is unchanged between v1 and v2 (verified
// by diffing T1_D4_GOLD.json against T1_D4_GOLD_v2_FULL.json: only
// COMPARE_TWO_VALUES workflows differ).
func FrozenShapeDAG(shape string) ([]dagStep, error) {
	switch shape {
	case "READ_AND_CHECK":
		// T1_SCORER_RULE.json's scoring_contract_per_shape: has_verify=false,
		// scoring_input="RunRecord.FinalValue (from last node observation)".
		// The terminal value is norm_A itself (T1_D4_GOLD_v2_FULL.json:
		// final_expected_value == intermediate_values.norm_A exactly, for
		// every READ_AND_CHECK workflow) -- "check" is a required, counted
		// Parrot call (the Arm-A prompt is "read one value and check it
		// against a threshold") but it is a side-observation, not on the
		// path to the terminal value, and there is no VERIFY/Fact-promotion
		// node for this shape at all ("VERIFY node should not appear").
		steps := acquireSteps("A")
		steps = append(steps, dagStep{
			LocalID: "check", Capability: "COMPARE_NUMBERS", Operation: OpThresholdCheck,
			DependsOn: []string{"norm_A"}, InputKeys: []string{"norm_A"}, OutputKey: "check_verdict",
		})
		return steps, nil

	case "COMPARE_TWO_VALUES":
		// has_verify=false too, but here (unlike READ_AND_CHECK) the
		// comparison node's own output IS the terminal value. One
		// COMPARE_NUMBERS Parrot call (per the frozen 60-call budget: one
		// comparison per workflow, not two) produces BOTH the LESS/GREATER/
		// EQUAL verdict recorded in T1_D4_GOLD_v2_FULL.json's
		// intermediate_values.comparison_verdict AND the terminal numeric
		// value -- T1_V2's corrected rule is expected=max(A,B), never the
		// historical v1 A-B formula. OpMax computes both in one call.
		steps := append(acquireSteps("A"), acquireSteps("B")...)
		steps = append(steps, dagStep{
			LocalID: "cmp", Capability: "COMPARE_NUMBERS", Operation: OpMax,
			DependsOn: []string{"norm_A", "norm_B"}, InputKeys: []string{"norm_A", "norm_B"}, OutputKey: "final",
		})
		return steps, nil

	case "DIFFERENCE_THEN_VERIFY":
		steps := append(acquireSteps("A"), acquireSteps("B")...)
		steps = append(steps,
			dagStep{LocalID: "diff", Capability: "ARITHMETIC", Operation: OpSubtract,
				DependsOn: []string{"norm_A", "norm_B"}, InputKeys: []string{"norm_A", "norm_B"}, OutputKey: "diff_result"},
			dagStep{LocalID: "norm_diff", Capability: "NORMALIZE", Operation: OpNormalize,
				DependsOn: []string{"diff"}, InputKeys: []string{"diff_result"}, OutputKey: "norm_diff"},
			dagStep{LocalID: "verify", Capability: "VERIFY", Operation: OpVerify,
				DependsOn: []string{"norm_diff"}, InputKeys: []string{"norm_diff"}, OutputKey: "final"},
		)
		return steps, nil

	case "RATIO_OF_DIFFERENCE":
		steps := acquireSteps("A")
		steps = append(steps, acquireSteps("B")...)
		steps = append(steps, acquireSteps("C")...)
		steps = append(steps,
			dagStep{LocalID: "diff", Capability: "ARITHMETIC", Operation: OpSubtract,
				DependsOn: []string{"norm_A", "norm_B"}, InputKeys: []string{"norm_A", "norm_B"}, OutputKey: "diff_result"},
			dagStep{LocalID: "norm_diff", Capability: "NORMALIZE", Operation: OpNormalize,
				DependsOn: []string{"diff"}, InputKeys: []string{"diff_result"}, OutputKey: "norm_diff"},
			dagStep{LocalID: "ratio", Capability: "ARITHMETIC", Operation: OpDivide,
				DependsOn: []string{"norm_diff", "norm_C"}, InputKeys: []string{"norm_diff", "norm_C"}, OutputKey: "ratio_result"},
			dagStep{LocalID: "norm_ratio", Capability: "NORMALIZE", Operation: OpNormalize,
				DependsOn: []string{"ratio"}, InputKeys: []string{"ratio_result"}, OutputKey: "norm_ratio"},
			dagStep{LocalID: "verify", Capability: "VERIFY", Operation: OpVerify,
				DependsOn: []string{"norm_ratio"}, InputKeys: []string{"norm_ratio"}, OutputKey: "final"},
		)
		return steps, nil

	case "RECONCILIATION_CHAIN":
		steps := acquireSteps("A")
		steps = append(steps, acquireSteps("a")...)
		steps = append(steps, acquireSteps("B")...)
		steps = append(steps, acquireSteps("b")...)
		steps = append(steps,
			dagStep{LocalID: "sub_A", Capability: "ARITHMETIC", Operation: OpSubtract,
				DependsOn: []string{"norm_A", "norm_a"}, InputKeys: []string{"norm_A", "norm_a"}, OutputKey: "sub_A"},
			dagStep{LocalID: "sub_B", Capability: "ARITHMETIC", Operation: OpSubtract,
				DependsOn: []string{"norm_B", "norm_b"}, InputKeys: []string{"norm_B", "norm_b"}, OutputKey: "sub_B"},
			dagStep{LocalID: "disagreement_pct", Capability: "ARITHMETIC", Operation: OpPercentDifference,
				DependsOn: []string{"sub_A", "sub_B"}, InputKeys: []string{"sub_A", "sub_B"}, OutputKey: "disagreement_pct"},
			dagStep{LocalID: "norm_pct", Capability: "NORMALIZE", Operation: OpNormalize,
				DependsOn: []string{"disagreement_pct"}, InputKeys: []string{"disagreement_pct"}, OutputKey: "norm_pct"},
			dagStep{LocalID: "fraction", Capability: "ARITHMETIC", Operation: OpPercentToFraction,
				DependsOn: []string{"norm_pct"}, InputKeys: []string{"norm_pct"}, OutputKey: "fraction_result"},
			dagStep{LocalID: "norm_fraction", Capability: "NORMALIZE", Operation: OpNormalize,
				DependsOn: []string{"fraction"}, InputKeys: []string{"fraction_result"}, OutputKey: "norm_fraction"},
			dagStep{LocalID: "tolerance_margin", Capability: "ARITHMETIC", Operation: OpSubtractTolerance,
				DependsOn: []string{"norm_fraction"}, InputKeys: []string{"norm_fraction"}, OutputKey: "tolerance_margin"},
			dagStep{LocalID: "norm_margin", Capability: "NORMALIZE", Operation: OpNormalize,
				DependsOn: []string{"tolerance_margin"}, InputKeys: []string{"tolerance_margin"}, OutputKey: "norm_margin"},
			dagStep{LocalID: "cmp_zero", Capability: "COMPARE_NUMBERS", Operation: OpCompareZero,
				DependsOn: []string{"norm_margin"}, InputKeys: []string{"norm_margin"}, OutputKey: "cmp_zero"},
			// T1_SCORER_RULE.json: "VERIFY single-target on norm_margin;
			// cmp_zero verdict lives on Blackboard but not promoted" --
			// VERIFY depends only on norm_margin, not on cmp_zero, even
			// though cmp_zero is itself a required, counted node.
			dagStep{LocalID: "verify", Capability: "VERIFY", Operation: OpVerify,
				DependsOn: []string{"norm_margin"}, InputKeys: []string{"norm_margin"}, OutputKey: "final"},
		)
		return steps, nil

	default:
		return nil, fmt.Errorf("tonalt1arms: unknown shape %q", shape)
	}
}

// ShapeDAG is the shared, operation-aware DAG representation for one frozen
// TaskFamily -- the single representation consumed by call-budget
// derivation, Arm B traversal, Arm C traversal, and counterfactual
// descendant-closure replay (task requirement: "one shared executable DAG
// representation", not four independent copies of the experiment logic).
//
// TerminalNodeID/HasVerify are read directly off T1_SCORER_RULE.json's
// scoring_contract_per_shape, not inferred from OutputKey=="final" naming
// conventions in FrozenShapeDAG's step list (that naming happens to hold for
// every shape below, but TerminalNodeID/HasVerify are the actual frozen
// authority a consumer should read).
type ShapeDAG struct {
	Shape          string
	Steps          []dagStep
	TerminalNodeID string // LocalID of the node whose OutputKey holds the terminal value
	HasVerify      bool   // true iff scoring requires a promoted VERIFY Fact (Shapes 3/4/5); false means RunRecord.FinalValue straight from the terminal node's observation (Shapes 1/2)
}

// shapeTerminals is read directly from T1_SCORER_RULE.json's
// scoring_contract_per_shape (has_verify / scoring_input fields), not
// invented: Shapes 1/2 score RunRecord.FinalValue from the last node's own
// observation (no VERIFY, no Fact promotion); Shapes 3/4/5 score the VERIFY
// node's promoted Fact.
var shapeTerminals = map[string]struct {
	terminalNodeID string
	hasVerify      bool
}{
	"READ_AND_CHECK":         {"norm_A", false},
	"COMPARE_TWO_VALUES":     {"cmp", false},
	"DIFFERENCE_THEN_VERIFY": {"verify", true},
	"RATIO_OF_DIFFERENCE":    {"verify", true},
	"RECONCILIATION_CHAIN":   {"verify", true},
}

// BuildShapeDAG wraps FrozenShapeDAG into the shared ShapeDAG type. It does
// not alter FrozenShapeDAG's step list, order, or capability values in any
// way -- those remain exactly what the existing call-budget tests
// (TestDeriveArmBCallBudget_Real et al.) already pin.
func BuildShapeDAG(shape string) (ShapeDAG, error) {
	steps, err := FrozenShapeDAG(shape)
	if err != nil {
		return ShapeDAG{}, err
	}
	terminal, ok := shapeTerminals[shape]
	if !ok {
		return ShapeDAG{}, fmt.Errorf("tonalt1arms: BuildShapeDAG: no frozen terminal-node mapping for shape %q", shape)
	}
	return ShapeDAG{
		Shape:          shape,
		Steps:          steps,
		TerminalNodeID: terminal.terminalNodeID,
		HasVerify:      terminal.hasVerify,
	}, nil
}

// StepByID returns the step with the given LocalID, or false if not found.
func (d ShapeDAG) StepByID(id string) (dagStep, bool) {
	for _, s := range d.Steps {
		if s.LocalID == id {
			return s, true
		}
	}
	return dagStep{}, false
}

// Descendants returns the transitive closure of nodes that (directly or
// indirectly) DependOn nodeID, computed by a real graph walk over the
// shared DAG's DependsOn edges -- not a hand-maintained per-shape table.
// Used by counterfactual replay (§12/D) to know exactly which nodes must be
// replayed after a POISON/REMOVE mutation, and by nothing else, so that the
// same graph structure that drives execution also drives replay.
func (d ShapeDAG) Descendants(nodeID string) []string {
	children := make(map[string][]string, len(d.Steps))
	for _, s := range d.Steps {
		for _, dep := range s.DependsOn {
			children[dep] = append(children[dep], s.LocalID)
		}
	}

	visited := make(map[string]bool)
	var order []string
	var walk func(id string)
	walk = func(id string) {
		for _, child := range children[id] {
			if visited[child] {
				continue
			}
			visited[child] = true
			order = append(order, child)
			walk(child)
		}
	}
	walk(nodeID)
	return order
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
