package tonalt1arms

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

const (
	// ScorerRelTol is the relative tolerance for numeric comparison (primary metric).
	ScorerRelTol = 1e-4
	// ScorerAbsTol is the absolute tolerance for numeric comparison (primary metric).
	ScorerAbsTol = 1e-6
)

// ScorerRule is the frozen scoring policy as a frozen artifact.
type ScorerRule struct {
	SchemaVersion        string  `json:"schema_version"`
	FrozenTimestamp      string  `json:"frozen_timestamp"`
	Description          string  `json:"description"`
	RelativeTolerance    float64 `json:"relative_tolerance"`
	AbsoluteTolerance    float64 `json:"absolute_tolerance"`
	ScorerHash           string  `json:"scorer_hash"`
	IntegerExactRequired bool    `json:"integer_exact_required"`
}

// ScoreResult scores a predicted value against a gold record.
// Returns (semanticCorrect, exactCorrect, parseFailure).
// semanticCorrect is the primary metric (uses tolerance).
// exactCorrect is strict normalized equality, tracked separately.
// parseFailure is true if the prediction could not be parsed as a finite number.
func ScoreResult(predicted float64, goldRecord Gold) (semantic bool, exact bool, parseFailure bool) {
	if math.IsNaN(predicted) || math.IsInf(predicted, 0) {
		return false, false, true
	}

	gold := goldRecord.FinalExpectedValue
	if math.IsNaN(gold) || math.IsInf(gold, 0) {
		// Malformed gold record (shouldn't happen)
		return false, false, true
	}

	// Exact equality (strict, no tolerance)
	exact = predicted == gold

	// Semantic correctness: use tolerance rule
	// If gold is an integer, require exact numeric equality; otherwise use isClose
	if isInteger(gold) {
		semantic = predicted == gold
	} else {
		semantic = isClose(predicted, gold, ScorerRelTol, ScorerAbsTol)
	}

	return semantic, exact, false
}

// isInteger checks if a float64 is an integer value (within epsilon).
func isInteger(x float64) bool {
	return x == math.Trunc(x)
}

// isClose implements the Go 1.22 math.IsClose semantics locally.
// Returns true if a and b are close within relative and absolute tolerances.
func isClose(a, b, relTol, absTol float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	if diff <= absTol {
		return true
	}
	return diff <= relTol*math.Max(math.Abs(a), math.Abs(b))
}

// WriteScorerRule writes the frozen scorer rule as an artifact.
// This must be done before the first model call.
func WriteScorerRule(outPath string) error {
	rule := ScorerRule{
		SchemaVersion:        "tonal.t1.scorer.r1",
		FrozenTimestamp:      "", // Caller should set this
		Description:          "Parse model output to a finite number. If gold is an integer, require exact equality. Otherwise use IsClose(rel_tol=1e-4, abs_tol=1e-6) for primary semantic_correct. exact_correct tracked separately (strict normalized equality). Parse/format failures never rescued by tolerance.",
		RelativeTolerance:    ScorerRelTol,
		AbsoluteTolerance:    ScorerAbsTol,
		ScorerHash:           "frozen-pre-inference",
		IntegerExactRequired: true,
	}

	data, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scorer rule: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write scorer rule: %w", err)
	}

	return nil
}
