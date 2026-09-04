package exocortex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParrotR1_OptionalPhaseHRegression is SUPPLEMENTARY evidence only. It
// runs only when TLALOC_PHASEH_DIR points at the frozen Phase-H experiment
// directory; it never mutates those artifacts and performs no inference.
// When the directory is absent it SKIPS (never fails): runtime and CI must
// not depend on local historical experiment files.
func TestParrotR1_OptionalPhaseHRegression(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv("TLALOC_PHASEH_DIR"))
	if dir == "" {
		t.Skip("set TLALOC_PHASEH_DIR to the frozen Phase-H experiment dir for the optional regression")
	}

	// 1. the frozen Phase-H artifacts must reference this exact profile hash
	referenced := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr == nil && strings.Contains(string(body), profileR1HashFull) {
			referenced = true
		}
		return nil
	})
	if !referenced {
		t.Fatalf("no Phase-H JSON under %s references profile hash %s", dir, profileR1HashFull)
	}

	// 2. AdapterR1 still reproduces the frozen H-D behaviour: a missing
	//    visual operand is rejected pre-inference with zero model calls.
	decision, err := (AdapterR1{Profile: loadFrozenProfile(t)}).Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: false})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !decision.Rejected || decision.ModelCallCount != 0 {
		t.Fatalf("regression: Phase-H H-D behaviour changed: %+v", decision)
	}

	// 3. and the frozen H-A low-scale recovery still upscales to preferred.
	decision, err = (AdapterR1{Profile: loadFrozenProfile(t)}).Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 8, VisualFieldName: "LINE"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !transformApplied(decision, "UPSCALE_TO_PREFERRED") || decision.ResultingWorkingSet["target_line_height_px"] != float64(32) {
		t.Fatalf("regression: Phase-H H-A low-scale recovery changed: %+v", decision)
	}
}
