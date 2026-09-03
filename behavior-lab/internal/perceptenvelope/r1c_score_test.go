package perceptenvelope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const r1cPoolPath = "../../experiments/parrot-perceptual-envelope-r1/datasets/R1C_POOL.json"

func loadR1CPoolOrSkip(t *testing.T) MorphologyPool {
	t.Helper()
	body, err := os.ReadFile(r1cPoolPath)
	if err != nil {
		t.Skipf("R1C_POOL.json not present: %v", err)
	}
	var pool MorphologyPool
	if err := json.Unmarshal(body, &pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestR1C_Allocation_DeterministicAndLeakageProof(t *testing.T) {
	pool := loadR1CPoolOrSkip(t)
	a1 := AllocateR1C(pool, map[string]struct{}{})
	a2 := AllocateR1C(pool, map[string]struct{}{})
	if r1cAllocFingerprint(a1) != r1cAllocFingerprint(a2) {
		t.Fatal("allocation not deterministic")
	}
	// synthetic families present, real/synthetic never filed together
	sawSynth := false
	total := 0
	for _, fa := range a1.Families {
		for _, b := range fa.RealBases {
			if b.Provenance == ProvSynthetic || b.Candidate == nil {
				t.Errorf("%s: synthetic/invalid base under RealBases", b.BaseID)
			}
			total++
		}
		for _, b := range fa.SyntheticBases {
			if b.Provenance != ProvSynthetic || b.Candidate != nil {
				t.Errorf("%s: real base under SyntheticBases", b.BaseID)
			}
			sawSynth = true
			total++
		}
	}
	if !sawSynth {
		t.Error("no synthetic stratum allocated")
	}
	if total == 0 {
		t.Error("no bases allocated")
	}
}

func TestR1C_GlyphBank_And_SyntheticRender_Deterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("glyph bank build is slow; run without -short")
	}
	store := "../../experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0"
	if _, err := os.Stat(filepath.Join(store, "manifest.json")); err != nil {
		t.Skipf("store not present: %v", err)
	}
	b1, err := BuildGlyphBank(store, "")
	if err != nil {
		t.Fatal(err)
	}
	b2, err := BuildGlyphBank(store, "")
	if err != nil {
		t.Fatal(err)
	}
	if b1.SHA256 != b2.SHA256 {
		t.Fatalf("glyph bank not deterministic: %s vs %s", b1.SHA256, b2.SHA256)
	}
	i1, _, e1 := RenderSyntheticNumber(b1, "(512, 256)")
	i2, _, e2 := RenderSyntheticNumber(b2, "(512, 256)")
	if e1 != nil || e2 != nil {
		t.Fatalf("render errors: %v %v", e1, e2)
	}
	if pngHash(i1) != pngHash(i2) {
		t.Fatal("synthetic render not byte-stable")
	}
}

func TestR1C_ScorerSelfTest(t *testing.T) {
	for _, p := range R1CScorerSelfTest() {
		t.Error(p)
	}
}

func TestR1C_Score_DualEndpoints(t *testing.T) {
	cases := []struct {
		name                string
		family, raw, gold   string
		wantValue, wantSurf bool
		wantClass           string
	}{
		{"single digit ok", FamSingleDigit, "7", "7", true, true, ""},
		{"single digit wrong", FamSingleDigit, "1", "7", false, false, "DIGIT_SUBSTITUTION"},
		{"multi digit ok", FamMultiDigit, "1024", "1024", true, true, ""},
		{"multi digit suffix trunc", FamMultiDigit, "10", "1024", false, false, "SUFFIX_TRUNCATION"},
		{"multi digit prefix trunc", FamMultiDigit, "24", "1024", false, false, "PREFIX_TRUNCATION"},
		{"thousands faithful", FamThousands, "1,024", "1,024", true, true, ""},
		{"thousands sep dropped", FamThousands, "1024", "1,024", true, false, "THOUSANDS_SEPARATOR_DROPPED"},
		{"thousands value wrong", FamThousands, "1,025", "1,024", false, false, "DIGIT_SUBSTITUTION"},
		{"decimal faithful", FamDecimal, "0.001", "0.001", true, true, ""},
		{"decimal point dropped", FamDecimal, "125", "12.5", false, false, "DECIMAL_POINT_DROPPED"},
		{"decimal exact not float", FamDecimal, "0.1", "0.10", true, false, "SURFACE_FORM_ONLY"},
		{"percent faithful", FamPercentage, "12.5%", "12.5%", true, true, ""},
		{"percent sign dropped", FamPercentage, "12.5", "12.5%", false, false, "PERCENT_SIGN_DROPPED"},
		{"signed faithful", FamSigned, "-42", "-42", true, true, ""},
		{"sign dropped", FamSigned, "42", "-42", false, false, "SIGN_DROPPED"},
		{"sign flipped", FamSigned, "-42", "+42", false, false, "SIGN_FLIPPED"},
		{"range faithful hyphen", FamRange, "10-20", "10-20", true, true, ""},
		{"range dash folds", FamRange, "10-20", "10–20", true, true, ""},
		{"range endpoint error", FamRange, "10-25", "10–20", false, false, "RANGE_ENDPOINT_ERROR"},
		{"sci equal to plain", FamScientific, "1000000", "1e6", true, false, "EXPONENT_DROPPED"},
		{"sci faithful", FamScientific, "3.14e-4", "3.14e-4", true, true, ""},
		{"sci value error", FamScientific, "3.15e-4", "3.14e-4", false, false, "EXPONENT_VALUE_ERROR"},
		{"tuple faithful", FamCoordTuple, "(512, 256)", "(512, 256)", true, true, ""},
		{"tuple order error", FamCoordTuple, "(256, 512)", "(512, 256)", false, false, "TUPLE_ORDER_ERROR"},
		{"tuple arity error", FamCoordTuple, "(512, 256, 3)", "(512, 256)", false, false, "TUPLE_ARITY_ERROR"},
		{"equation operand", FamEquation, "128", "x = 128", true, false, ""},
		{"equation operand wrong", FamEquation, "129", "x = 128", false, false, "DIGIT_SUBSTITUTION"},
		{"abstain", FamMultiDigit, "", "128", false, false, "ABSTAIN"},
		{"commentary buries correct value", FamMultiDigit, "The number in the box is 128 exactly", "128", true, false, "COMMENTARY_CONTAMINATION"},
		{"commentary wrong value", FamMultiDigit, "The number in the box appears to be 12 here", "128", false, false, "COMMENTARY_CONTAMINATION"},
		{"table cell -> decimal", FamTableCell, "3.2", "3.2", true, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ScoreR1C(c.family, c.raw, c.gold)
			if got.ValueCorrect != c.wantValue {
				t.Errorf("ValueCorrect = %v, want %v (got canon %q gold canon %q)", got.ValueCorrect, c.wantValue, got.GotValueCanonical, got.GoldValueCanonical)
			}
			if got.SurfaceFormCorrect != c.wantSurf {
				t.Errorf("SurfaceFormCorrect = %v, want %v", got.SurfaceFormCorrect, c.wantSurf)
			}
			if c.wantClass != "" && got.FailureClass != c.wantClass {
				t.Errorf("FailureClass = %q, want %q", got.FailureClass, c.wantClass)
			}
		})
	}
}
