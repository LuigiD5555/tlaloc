package questionclass

import (
	"encoding/json"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// LoadProfile reads a CalibrationProfile JSON (produced by
// tools/questionclass_calibrate.py) from disk. The profile is competence
// evidence measured on held-in and held-out data; a caller consults it
// before trusting this model's prediction and runs its AdmitAsActive gate
// before wiring the model as an active specialist.
func LoadProfile(path string) (calibration.CalibrationProfile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return calibration.CalibrationProfile{}, fmt.Errorf("questionclass: reading calibration profile: %w", err)
	}
	var profile calibration.CalibrationProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return calibration.CalibrationProfile{}, fmt.Errorf("questionclass: decoding calibration profile: %w", err)
	}
	if profile.Schema != calibration.Schema {
		return calibration.CalibrationProfile{}, fmt.Errorf("questionclass: calibration profile schema %q, want %q", profile.Schema, calibration.Schema)
	}
	return profile, nil
}
