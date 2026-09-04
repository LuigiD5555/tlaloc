package perceptenvelope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestR1D_ScorerSelfTest(t *testing.T) {
	for _, p := range R1DScorerSelfTest() {
		t.Error(p)
	}
}

func loadLVPoolOrSkip(t *testing.T) LabelValuePool {
	t.Helper()
	body, err := os.ReadFile("../../experiments/parrot-perceptual-envelope-r1/datasets/R1D_POOL.json")
	if err != nil {
		t.Skipf("R1D_POOL.json absent: %v", err)
	}
	var p LabelValuePool
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestR1D_Allocation_DeterministicEligibilityAndDistractors(t *testing.T) {
	pool := loadLVPoolOrSkip(t)
	a1 := AllocateR1D(pool)
	a2 := AllocateR1D(pool)
	if r1dAllocFP(a1) != r1dAllocFP(a2) {
		t.Fatal("allocation not deterministic")
	}
	for _, b := range a1.Bases {
		if !b.Eligible {
			continue
		}
		if !isPlainInt(b.Value) || len(b.Value) < 2 || len(b.Value) > 5 {
			t.Errorf("%s: value %q not 2-5 digit int", b.BaseID, b.Value)
		}
		if strings.Count(b.LineText, b.Value) != 1 {
			t.Errorf("%s: value not unique in line", b.BaseID)
		}
		if len(b.DistractorValues) != 8 {
			t.Errorf("%s: %d distractors, want 8", b.BaseID, len(b.DistractorValues))
		}
		seen := map[string]bool{}
		for _, d := range b.DistractorValues {
			if d == b.Value {
				t.Errorf("%s: distractor equals answer", b.BaseID)
			}
			if !isPlainInt(d) || len(d) < 2 || len(d) > 4 {
				t.Errorf("%s: distractor %q not 2-4 digit int", b.BaseID, d)
			}
			if seen[d] {
				t.Errorf("%s: duplicate distractor %q", b.BaseID, d)
			}
			seen[d] = true
		}
	}
	if a1.EligibleCount < 18 && a1.Proceed {
		t.Error("proceed true with <18 eligible")
	}
}

func TestR1D_GeometryAndPixelInvariants(t *testing.T) {
	store := "../../experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0"
	if _, err := os.Stat(filepath.Join(store, "manifest.json")); err != nil {
		t.Skipf("store absent: %v", err)
	}
	if testing.Short() {
		t.Skip("renders pages; run without -short")
	}
	pool := loadLVPoolOrSkip(t)
	alloc := AllocateR1D(pool)
	bank, err := LoadOrBuildGlyphBank("../../experiments/parrot-perceptual-envelope-r1/datasets/R1C_GLYPHBANK.json", store, "")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, base := range alloc.Bases {
		if !base.Eligible || n >= 3 {
			continue
		}
		n++
		imgs, err := RenderR1DSanity(store, "", base, bank)
		if err != nil {
			t.Fatalf("%s: %v", base.BaseID, err)
		}
		if pngHash(imgs["d0v_value_cued"]) == pngHash(imgs["d0l_label_cued"]) {
			t.Errorf("%s: D0V and D0L identical (cue did not move)", base.BaseID)
		}
		geo, _ := DeriveR1DGeometry(base)
		h0 := lineRectHash(imgs["d1_k0"], geo.LineRectCanvas)
		for _, k := range []string{"d1_k2", "d1_k8"} {
			if lineRectHash(imgs[k], geo.LineRectCanvas) != h0 {
				t.Errorf("%s: line-rect pixels changed at %s", base.BaseID, k)
			}
		}
		// determinism
		imgs2, _ := RenderR1DSanity(store, "", base, bank)
		if pngHash(imgs["d1_k8"]) != pngHash(imgs2["d1_k8"]) {
			t.Errorf("%s: d1_k8 render not byte-stable", base.BaseID)
		}
	}
}

func TestR1D_AssocInstruction_NoLeakage(t *testing.T) {
	if containsAnyDigit(R1DAssocInstruction) {
		t.Error("assoc instruction contains a digit")
	}
	if R1DAssocOpcode != "READ_ASSOCIATED_NUMBER" {
		t.Errorf("opcode = %q", R1DAssocOpcode)
	}
}
