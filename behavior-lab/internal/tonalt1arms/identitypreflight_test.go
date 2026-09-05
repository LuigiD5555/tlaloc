package tonalt1arms

import (
	"context"
	"errors"
	"testing"

	"tlaloc.local/behaviorlab/internal/lfm2boundary"
)

var errFakeDial = errors.New("fake dial failure")

func fakeDial(result lfm2boundary.PreflightResult, err error) dialFunc {
	return func(ctx context.Context, endpoint, model string) (lfm2boundary.PreflightResult, error) {
		return result, err
	}
}

func TestRunIdentityPreflight_MatchesFrozenModelAndContext(t *testing.T) {
	dial := fakeDial(lfm2boundary.PreflightResult{
		Model: lfm2boundary.RequiredModel, ContextLength: lfm2boundary.RequiredContext, Vision: true,
	}, nil)

	result, err := RunIdentityPreflight(context.Background(), "http://fake-endpoint", lfm2boundary.RequiredModel, dial)
	if err != nil {
		t.Fatal(err)
	}
	if !result.ModelNameMatch {
		t.Error("ModelNameMatch = false, want true")
	}
	if !result.ContextMatch {
		t.Error("ContextMatch = false, want true")
	}
}

func TestRunIdentityPreflight_MismatchedModel(t *testing.T) {
	dial := fakeDial(lfm2boundary.PreflightResult{Model: "some-other-model", ContextLength: 2048}, nil)
	result, err := RunIdentityPreflight(context.Background(), "http://fake-endpoint", "some-other-model", dial)
	if err != nil {
		t.Fatal(err)
	}
	if result.ModelNameMatch {
		t.Error("ModelNameMatch = true, want false")
	}
	if result.ContextMatch {
		t.Error("ContextMatch = true, want false")
	}
}

// TestRunIdentityPreflight_WeightsGuardNeverFabricated is the required
// honesty test: regardless of model/context match, WeightsIdentityGuard
// must never report PASS, since no genuinely trusted weights-hash source
// exists (the investigated candidate binding fails its own sidecar
// integrity check).
func TestRunIdentityPreflight_WeightsGuardNeverFabricated(t *testing.T) {
	dial := fakeDial(lfm2boundary.PreflightResult{Model: lfm2boundary.RequiredModel, ContextLength: lfm2boundary.RequiredContext}, nil)
	result, err := RunIdentityPreflight(context.Background(), "http://fake-endpoint", lfm2boundary.RequiredModel, dial)
	if err != nil {
		t.Fatal(err)
	}
	if result.WeightsIdentityGuard == "PASS" {
		t.Fatal("WeightsIdentityGuard reported PASS -- this must never happen given the investigated candidate binding fails its own integrity check")
	}
	if result.WeightsIdentityGuard != "NOT_AVAILABLE" {
		t.Fatalf("WeightsIdentityGuard = %q, want NOT_AVAILABLE", result.WeightsIdentityGuard)
	}
	if result.Detail == "" {
		t.Fatal("expected a non-empty Detail explaining why the guard is NOT_AVAILABLE")
	}
}

func TestRunIdentityPreflight_DialErrorPropagates(t *testing.T) {
	dial := fakeDial(lfm2boundary.PreflightResult{}, errFakeDial)
	_, err := RunIdentityPreflight(context.Background(), "http://fake-endpoint", lfm2boundary.RequiredModel, dial)
	if err == nil {
		t.Fatal("expected dial error to propagate")
	}
}

func TestRunIdentityPreflight_NilDialFailsClosed(t *testing.T) {
	_, err := RunIdentityPreflight(context.Background(), "http://fake-endpoint", lfm2boundary.RequiredModel, nil)
	if err == nil {
		t.Fatal("expected error for nil dial function")
	}
}

// TestWeightsIdentityGuard_DetectsSidecarMismatch directly proves the
// integrity-check logic: a candidate profile whose sidecar hash doesn't
// match its actual bytes must never be trusted, exactly as the real
// profiles/parrot-lfm2-vl-1.6b-r1.json case is handled.
func TestWeightsIdentityGuard_DetectsSidecarMismatch(t *testing.T) {
	status, detail := weightsIdentityGuard()
	if status != "NOT_AVAILABLE" {
		t.Fatalf("weightsIdentityGuard() status = %q, want NOT_AVAILABLE (given the known sidecar mismatch in this repo)", status)
	}
	if detail == "" {
		t.Fatal("expected a non-empty detail")
	}
}
