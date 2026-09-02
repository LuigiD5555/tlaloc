package groundingscore

import (
	"encoding/json"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// LoadProfile reads a CalibrationProfile JSON (produced by
// tools/grounding_calibrate.py) from disk. The profile is competence
// evidence measured on held-in and held-out data; the consolidator
// consults it (calibration.Verdict) before trusting the distilled score
// instead of falling back to the heavier answerscore judge.
func LoadProfile(path string) (calibration.CalibrationProfile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return calibration.CalibrationProfile{}, fmt.Errorf("groundingscore: reading calibration profile: %w", err)
	}
	var profile calibration.CalibrationProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return calibration.CalibrationProfile{}, fmt.Errorf("groundingscore: decoding calibration profile: %w", err)
	}
	if profile.Schema != calibration.Schema {
		return calibration.CalibrationProfile{}, fmt.Errorf("groundingscore: calibration profile schema %q, want %q", profile.Schema, calibration.Schema)
	}
	return profile, nil
}
