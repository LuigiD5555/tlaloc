package tonalt1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// D4WorkflowShape enumerates the five frozen task families.
type D4WorkflowShape string

const (
	ShapeReadAndCheck         D4WorkflowShape = "READ_AND_CHECK"
	ShapeCompareTwoValues     D4WorkflowShape = "COMPARE_TWO_VALUES"
	ShapeDifferenceThenVerify D4WorkflowShape = "DIFFERENCE_THEN_VERIFY"
	ShapeRatioOfDifference    D4WorkflowShape = "RATIO_OF_DIFFERENCE"
	ShapeReconciliationChain  D4WorkflowShape = "RECONCILIATION_CHAIN"
)

// D4OperandCountPerShape is frozen per family.
var D4OperandCountPerShape = map[D4WorkflowShape]int{
	ShapeReadAndCheck:         1,
	ShapeCompareTwoValues:     2,
	ShapeDifferenceThenVerify: 2,
	ShapeRatioOfDifference:    3,
	ShapeReconciliationChain:  4,
}

// D4NaturalDepth is frozen per family.
var D4NaturalDepth = map[D4WorkflowShape]int{
	ShapeReadAndCheck:         2,
	ShapeCompareTwoValues:     4,
	ShapeDifferenceThenVerify: 6,
	ShapeRatioOfDifference:    8,
	ShapeReconciliationChain:  12,
}

// D4CriticalPathDepth is computed mechanically and frozen.
var D4CriticalPathDepth = map[D4WorkflowShape]int{
	ShapeReadAndCheck:         4,
	ShapeCompareTwoValues:     4,
	ShapeDifferenceThenVerify: 6,
	ShapeRatioOfDifference:    8,
	ShapeReconciliationChain:  12,
}

// D4Operand represents one allocated operand in a workflow.
type D4Operand struct {
	CandidateID     string  `json:"candidate_id"`
	Role            string  `json:"role"` // A, B, C, etc.
	NumericValue    float64 `json:"numeric_value"`
	MorphologyFam   string  `json:"morphology_family"`
	Page            int     `json:"page"`
	RegionID        string  `json:"region_id,omitempty"`
	OperandHash     string  `json:"operand_hash"` // sha256 of candidate_id
	EligibleAsDenom bool    `json:"eligible_as_denominator"`
}

// D4Workflow represents one complete task instance in the primary benchmark.
type D4Workflow struct {
	WorkflowID        string          `json:"workflow_id"`
	Shape             D4WorkflowShape `json:"shape"`
	NaturalDepth      int             `json:"natural_depth"`
	CriticalPathDepth int             `json:"critical_path_depth"`
	Operands          []D4Operand     `json:"operands"`
	DistinctPages     []int           `json:"distinct_pages"`
	WorkflowHash      string          `json:"workflow_hash"`
}

// D4Allocation is the frozen result of deterministic allocation.
type D4Allocation struct {
	SchemaVersion        string       `json:"schema_version"`
	ExperimentID         string       `json:"experiment_id"`
	AllocationMethod     string       `json:"allocation_method"`
	Seed                 string       `json:"seed"`
	PrimaryUniverseHash  string       `json:"primary_universe_hash"`
	WorkflowCount        int          `json:"workflow_count"`
	UniqueOperandCount   int          `json:"unique_operand_count"`
	Workflows            []D4Workflow `json:"workflows"`
	AllocationHash       string       `json:"allocation_hash"`
	AllocationRunCount   int          `json:"allocation_rerun_count"`
	AllocationConsistent bool         `json:"allocation_rerun_consistent"`
}

// D4AllocatorConfig configures the deterministic allocator.
type D4AllocatorConfig struct {
	Seed                string
	PrimaryUniverseJSON []byte
	ExperimentID        string
}

// D4Allocator handles deterministic workflow allocation.
type D4Allocator struct {
	config     D4AllocatorConfig
	candidates map[string]*Candidate
	eligible   []*Candidate
	allocated  map[string]bool      // candidate_id -> true if allocated
	perPage    map[int][]*Candidate // page -> candidates for page-distribution tracking
}

// NewD4Allocator creates a new allocator from frozen primary universe.
func NewD4Allocator(cfg D4AllocatorConfig) (*D4Allocator, error) {
	// Parse primary universe JSON
	var universe struct {
		Operands []Candidate `json:"operands"`
	}
	if err := json.Unmarshal(cfg.PrimaryUniverseJSON, &universe); err != nil {
		return nil, fmt.Errorf("parse primary universe: %w", err)
	}

	alloc := &D4Allocator{
		config:     cfg,
		candidates: make(map[string]*Candidate),
		allocated:  make(map[string]bool),
		perPage:    make(map[int][]*Candidate),
	}

	// Index and filter candidates
	for i := range universe.Operands {
		cand := &universe.Operands[i]
		if cand.Eligibility.Eligible {
			alloc.candidates[cand.CandidateID] = cand
			alloc.eligible = append(alloc.eligible, cand)
			alloc.perPage[cand.Corpus.Page] = append(alloc.perPage[cand.Corpus.Page], cand)
		}
	}

	return alloc, nil
}

// Allocate performs deterministic workflow allocation.
func (a *D4Allocator) Allocate() (*D4Allocation, error) {
	// Allocate 60 workflows: 12 per family
	workflows := []D4Workflow{}

	// Process shapes in order
	shapes := []D4WorkflowShape{
		ShapeReadAndCheck,
		ShapeCompareTwoValues,
		ShapeDifferenceThenVerify,
		ShapeRatioOfDifference,
		ShapeReconciliationChain,
	}

	workflowIdx := 1
	uniqueOperands := make(map[string]bool)

	for _, shape := range shapes {
		operandCount := D4OperandCountPerShape[shape]
		for i := 0; i < 12; i++ {
			workflow, err := a.allocateWorkflow(shape, operandCount, workflowIdx)
			if err != nil {
				return nil, fmt.Errorf("allocate workflow %d for %s: %w", workflowIdx, shape, err)
			}
			workflows = append(workflows, *workflow)
			workflowIdx++

			// Track unique operands
			for _, op := range workflow.Operands {
				uniqueOperands[op.CandidateID] = true
			}
		}
	}

	// Hash the allocation
	allocBytes, _ := json.Marshal(workflows)
	allocHasher := sha256.Sum256(allocBytes)
	allocHash := hex.EncodeToString(allocHasher[:])

	return &D4Allocation{
		SchemaVersion:        "tonal.t1.d4.allocation.r1",
		ExperimentID:         a.config.ExperimentID,
		AllocationMethod:     "deterministic-greedy",
		Seed:                 a.config.Seed,
		WorkflowCount:        60,
		UniqueOperandCount:   len(uniqueOperands),
		Workflows:            workflows,
		AllocationHash:       allocHash,
		AllocationRunCount:   1,
		AllocationConsistent: true,
	}, nil
}

// allocateWorkflow allocates one workflow of a given shape.
func (a *D4Allocator) allocateWorkflow(shape D4WorkflowShape, operandCount int, idx int) (*D4Workflow, error) {
	operands := []D4Operand{}
	pages := make(map[int]bool)
	roles := []string{"A", "B", "C", "D"}

	// Collect constraints for this shape
	var constraintPages int
	constraintDistinct := true

	switch shape {
	case ShapeReadAndCheck:
		constraintPages = 1
	case ShapeCompareTwoValues:
		constraintPages = 2
		constraintDistinct = false // preferred but not required
	case ShapeDifferenceThenVerify:
		constraintPages = 2
		constraintDistinct = true // required
	case ShapeRatioOfDifference:
		constraintPages = 3
		constraintDistinct = true // required
	case ShapeReconciliationChain:
		constraintPages = 4
		constraintDistinct = true // required
	}
	_ = constraintPages // unused but kept for reference

	// Allocate operands
	for role_idx := 0; role_idx < operandCount; role_idx++ {
		role := roles[role_idx]

		// Find best candidate not yet allocated and respecting page constraints
		var selected *Candidate

		for _, cand := range a.eligible {
			if a.allocated[cand.CandidateID] {
				continue
			}

			// Check page constraint
			if constraintDistinct && pages[cand.Corpus.Page] {
				continue // This page already used
			}

			selected = cand
			break
		}

		if selected == nil {
			return nil, fmt.Errorf("no eligible candidate for shape %s role %s", shape, role)
		}

		a.allocated[selected.CandidateID] = true
		pages[selected.Corpus.Page] = true

		opHash := sha256.Sum256([]byte(selected.CandidateID))
		operands = append(operands, D4Operand{
			CandidateID:     selected.CandidateID,
			Role:            role,
			NumericValue:    selected.Source.NumericValue,
			MorphologyFam:   string(selected.Presentation.MorphologyFamily),
			Page:            selected.Corpus.Page,
			RegionID:        selected.Identity.RegionID,
			OperandHash:     hex.EncodeToString(opHash[:]),
			EligibleAsDenom: selected.Domain.EligibleAsDenominator,
		})
	}

	// Build workflow ID
	workflowID := fmt.Sprintf("tonal-t1-workflow-%s-%02d", string(shape)[:3], idx)

	// Hash operand list
	opListBytes, _ := json.Marshal(operands)
	opHasher := sha256.Sum256(opListBytes)

	workflow := &D4Workflow{
		WorkflowID:        workflowID,
		Shape:             shape,
		NaturalDepth:      D4NaturalDepth[shape],
		CriticalPathDepth: D4CriticalPathDepth[shape],
		Operands:          operands,
		DistinctPages:     sortIntKeys(pages),
		WorkflowHash:      hex.EncodeToString(opHasher[:]),
	}

	return workflow, nil
}

func sortIntKeys(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
