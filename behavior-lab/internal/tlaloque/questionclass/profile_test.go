package questionclass

import (
	"path/filepath"
	"testing"
)

// The shipped calibration profile for questionclass-charcnn-r0 must NOT
// pass the ACTIVE admission gate: its out-of-distribution accuracy is
// ~0.51 and its measured confidence floor is unreachable. This test
// documents that and guards against a future profile silently claiming
// competence the model does not have. If the model is genuinely improved
// and re-measured, update this test deliberately.
func TestShippedProfile_IsNotAdmissibleAsActive(t *testing.T) {
	path := filepath.Join("..", "..", "..", "models", "questionclass-charcnn-r0.calibration.json")
	profile, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile(%s): %v", path, err)
	}

	if profile.OutOfDistribution.N == 0 {
		t.Error("expected the profile to carry a measured out-of-distribution slice")
	}

	admitted, reasons := profile.AdmitAsActive()
	if admitted {
		t.Fatalf("questionclass-charcnn-r0 must not be admissible as ACTIVE with its current OOD behavior (acc=%.3f)", profile.OutOfDistribution.Accuracy)
	}
	t.Logf("correctly held back from ACTIVE; reasons: %v", reasons)
}
