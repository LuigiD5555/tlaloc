package groundingscore

import (
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// The shipped calibration profile for groundingscore-distilled-r0 must NOT
// pass the ACTIVE admission gate: its out-of-distribution within-tolerance
// accuracy is ~0.46 and its confidence is uninformative, so the measured
// confidence floor is unreachable. This documents that and guards against a
// future profile silently claiming competence the model does not have. If
// the model is genuinely improved and re-measured (see tools/GROUNDING_RESULTS.md
// "Next (r1)"), update this test deliberately.
func TestShippedProfile_IsNotAdmissibleAsActive(t *testing.T) {
	path := filepath.Join("..", "..", "..", "models", "groundingscore-distilled-r0.calibration.json")
	profile, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile(%s): %v", path, err)
	}
	if profile.OutOfDistribution.N == 0 {
		t.Error("expected the profile to carry a measured out-of-distribution slice")
	}

	// The consolidator's gate: with this profile every prediction must be an
	// abstention, so the consolidator always falls back to answerscore.
	if verdict := profile.Verdict(calibration.Query{Confidence: 0.999}); verdict == calibration.Answered {
		t.Fatalf("shipped profile should never return ANSWERED (confidence floor %.2f), got %s",
			profile.ConfidenceFloor, verdict)
	}

	if admitted, reasons := profile.AdmitAsActive(); admitted {
		t.Fatalf("groundingscore-distilled-r0 must not be admissible as ACTIVE with its current OOD behavior (acc=%.3f); reasons=%v",
			profile.OutOfDistribution.Accuracy, reasons)
	}
}
