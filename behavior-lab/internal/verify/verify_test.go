package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"tlaloc.local/behaviorlab/internal/executor"
)

func TestStructuralChecks(t *testing.T) {
	if !JSONValid([]byte(`{"a":1}`)).Passed || JSONValid([]byte(`{bad`)).Passed {
		t.Error("JSONValid wrong")
	}

	data := []byte("hello")
	full := hex.EncodeToString(func() []byte { s := sha256.Sum256(data); return s[:] }())
	if !HashMatches(data, full).Passed {
		t.Error("full hash should match")
	}
	if !HashMatches(data, full[:12]).Passed {
		t.Error("short hash prefix should match")
	}
	if HashMatches(data, "deadbeef").Passed {
		t.Error("wrong hash must not match")
	}

	if !RequiredFields([]byte(`{"x":1,"y":2}`), []string{"x", "y"}).Passed {
		t.Error("present fields should pass")
	}
	if RequiredFields([]byte(`{"x":1,"y":null}`), []string{"x", "y"}).Passed {
		t.Error("null field must fail")
	}

	if !InRange("score", 0.5, 0, 1).Passed || InRange("score", 1.5, 0, 1).Passed {
		t.Error("InRange wrong")
	}
	if !OneOf("verdict", "VERIFIED", []string{"VERIFIED", "UNVERIFIED"}).Passed {
		t.Error("OneOf wrong")
	}
	if !DependenciesSatisfied([]string{"a", "b"}, []string{"a", "b", "c"}).Passed ||
		DependenciesSatisfied([]string{"a", "z"}, []string{"a"}).Passed {
		t.Error("DependenciesSatisfied wrong")
	}
}

type fakeSemantic struct {
	agree      bool
	confidence float64
	err        error
}

func (f fakeSemantic) AgreesWith(context.Context, string, string) (bool, float64, error) {
	return f.agree, f.confidence, f.err
}

func TestSpine_AllLevelsMustPass(t *testing.T) {
	ctx := context.Background()
	spine := Spine{Semantic: fakeSemantic{agree: true, confidence: 0.9}, SemanticMinConfidence: 0.6}

	good := spine.Verify(ctx, Input{
		Output:         []byte(`{"answer":"x","grounded":true}`),
		RequiredFields: []string{"answer", "grounded"},
		Claim:          "x is the answer",
		Evidence:       "page 3 says x",
		Execution: &executor.Result{
			Executed:       true,
			Postconditions: []executor.CheckResult{{Kind: "path_exists", Passed: true}},
			Verified:       true,
		},
	})
	if good.Verdict != Verified {
		t.Fatalf("expected VERIFIED, got %s (%+v)", good.Verdict, good.Checks)
	}

	// Semantic disagreement sinks the whole verdict even though structural
	// and world pass.
	spine.Semantic = fakeSemantic{agree: false, confidence: 0.95}
	bad := spine.Verify(ctx, Input{
		Output: []byte(`{"answer":"x"}`),
		Claim:  "x is the answer",
	})
	if bad.Verdict != Unverified {
		t.Fatalf("expected UNVERIFIED on semantic disagreement, got %s", bad.Verdict)
	}
	if len(bad.FailedLevels) != 1 || bad.FailedLevels[0] != Semantic {
		t.Errorf("expected SEMANTIC in failed levels, got %v", bad.FailedLevels)
	}
}

func TestSpine_LowConfidenceIsNotAgreement(t *testing.T) {
	spine := Spine{Semantic: fakeSemantic{agree: true, confidence: 0.3}, SemanticMinConfidence: 0.6}
	report := spine.Verify(context.Background(), Input{Claim: "c", Evidence: "e"})
	if report.Verdict != Unverified {
		t.Errorf("a low-confidence agreement must not verify: %s", report.Verdict)
	}
}

func TestSpine_NoInputsIsInconclusiveNotVerified(t *testing.T) {
	report := Spine{}.Verify(context.Background(), Input{})
	if report.Verdict != Inconclusive {
		t.Errorf("nothing to check must be INCONCLUSIVE, got %s", report.Verdict)
	}
}

func TestSpine_SemanticErrorFailsClosed(t *testing.T) {
	spine := Spine{Semantic: fakeSemantic{err: errors.New("service down")}, SemanticMinConfidence: 0.6}
	report := spine.Verify(context.Background(), Input{Claim: "c"})
	if report.Verdict != Unverified {
		t.Errorf("a semantic verifier error must fail closed, got %s", report.Verdict)
	}
}

func TestFromExecution_RolledBack(t *testing.T) {
	checks := FromExecution(executor.Result{
		Executed:         true,
		Postconditions:   []executor.CheckResult{{Kind: "path_exists", Passed: false}},
		RolledBack:       true,
		RollbackVerified: true,
	})
	// executed=true passes, postcondition fails, rolled_back(verified)=true passes.
	report := verdictFrom(checks)
	if report.Verdict != Unverified {
		t.Errorf("a rolled-back action is UNVERIFIED overall, got %s", report.Verdict)
	}
}
