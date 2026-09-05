package tonalt1arms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/lfm2boundary"
)

// IdentityPreflightResult is the frozen model-identity gate's outcome.
// WeightsIdentityGuard is deliberately a string status, never a bool that
// could quietly default to "pass" -- its only legitimate values are "PASS"
// and "NOT_AVAILABLE", and NOT_AVAILABLE is a real, honest gate outcome
// (task correction F), not a placeholder.
type IdentityPreflightResult struct {
	ModelNameMatch       bool
	ContextMatch         bool
	WeightsIdentityGuard string // "PASS" | "NOT_AVAILABLE" -- never fabricated
	Detail               string
}

// FrozenWeightsSHA256Prefix is the task-stated frozen weights identity
// prefix for lfm2-vl-1.6b.
const FrozenWeightsSHA256Prefix = "0a82498e"

// dialFunc matches lfm2boundary.Preflight's signature so tests can inject a
// fake without importing any real transport.
type dialFunc func(ctx context.Context, endpoint, model string) (lfm2boundary.PreflightResult, error)

// RunIdentityPreflight validates a PreflightResult (obtained via the
// injected dial function -- lfm2boundary.Preflight itself in a live caller,
// a fake in every test in this task) against the frozen model
// name/context, and separately attempts the weights-identity investigation
// documented in weightsIdentityGuard below.
//
// INVESTIGATION FINDING (task correction F/J, recorded here rather than
// silently assumed): lfm2boundary.PreflightResult carries no field for a
// reproducible weights hash, and LM Studio's own /api/v1/models response
// (which Preflight parses) exposes no such field either -- confirmed by
// reading internal/lfm2boundary/preflight.go's ListedModel/ModelInstance
// structs directly. A second avenue was investigated per correction J:
// profiles/parrot-lfm2-vl-1.6b-r1.json's executor.weights_gguf_sha256 field
// DOES record a value matching the frozen 0a82498e... prefix exactly -- but
// that profile file's own SHA-256 does NOT match its sidecar
// profiles/parrot-lfm2-vl-1.6b-r1.json.sha256 (verified directly: sha256sum
// of the committed file is c5bf197f..., the sidecar records 8acc959b...).
// Since the one candidate binding fails its own integrity check, it is not
// used as a trusted identity source -- WeightsIdentityGuard resolves to
// NOT_AVAILABLE, and this function does not compare a live PreflightResult
// against that profile's value under any circumstance.
func RunIdentityPreflight(ctx context.Context, endpoint, model string, dial dialFunc) (IdentityPreflightResult, error) {
	if dial == nil {
		return IdentityPreflightResult{}, fmt.Errorf("tonalt1arms: RunIdentityPreflight: nil dial function")
	}
	result, err := dial(ctx, endpoint, model)
	if err != nil {
		return IdentityPreflightResult{}, fmt.Errorf("tonalt1arms: RunIdentityPreflight: preflight dial failed: %w", err)
	}

	out := IdentityPreflightResult{
		ModelNameMatch: result.Model == lfm2boundary.RequiredModel,
		ContextMatch:   result.ContextLength == lfm2boundary.RequiredContext,
	}
	out.WeightsIdentityGuard, out.Detail = weightsIdentityGuard()
	return out, nil
}

// weightsIdentityGuard performs the offline investigation described above
// and returns the honest gate outcome. It never fabricates a weights-hash
// comparison against data that does not exist, and it never trusts a
// candidate binding whose own integrity check fails.
func weightsIdentityGuard() (status string, detail string) {
	profilePath := "profiles/parrot-lfm2-vl-1.6b-r1.json"
	sidecarPath := profilePath + ".sha256"

	profilePathAbs := RepoPathHelper(profilePath)
	sidecarPathAbs := RepoPathHelper(sidecarPath)

	data, err := os.ReadFile(profilePathAbs)
	if err != nil {
		return "NOT_AVAILABLE", fmt.Sprintf("candidate profile %s not readable: %v", profilePath, err)
	}
	sidecarRaw, err := os.ReadFile(sidecarPathAbs)
	if err != nil {
		return "NOT_AVAILABLE", fmt.Sprintf("candidate profile sidecar %s not readable: %v", sidecarPath, err)
	}

	sum := sha256.Sum256(data)
	actualHash := hex.EncodeToString(sum[:])
	expectedHash := parseSidecarHash(string(sidecarRaw))

	if actualHash != expectedHash {
		return "NOT_AVAILABLE", fmt.Sprintf(
			"candidate weights-identity binding %s exists but FAILS its own sidecar integrity check (file sha256=%s, sidecar records=%s) -- not trusted as an identity source",
			profilePath, actualHash, expectedHash,
		)
	}

	// The profile file itself is internally consistent -- but this still
	// only proves the FROZEN REFERENCE PROFILE'S recorded value matches, not
	// that the currently-loaded LM Studio instance's actual weights hash to
	// that value (no such live check is possible without inference or a
	// filesystem-level LM Studio model-store inspection this task did not
	// find evidence of). Reported as PASS is deliberately not claimed here.
	return "NOT_AVAILABLE", "candidate profile passes its own integrity check, but no live (non-inference) verification path from a running LM Studio instance to a weights hash exists -- reported honestly as unavailable rather than PASS"
}

// parseSidecarHash extracts the leading hex hash from a `sha256sum`-style
// sidecar line ("<hash>  <filename>\n").
func parseSidecarHash(sidecar string) string {
	for i, r := range sidecar {
		if r == ' ' || r == '\t' || r == '\n' {
			return sidecar[:i]
		}
	}
	return sidecar
}
