package perceptenvelope

import (
	"os"
	"testing"
)

func r1gSkipIfNoInputs(t *testing.T) {
	t.Helper()
	for _, f := range []string{"SOURCE_POOL_R1.json", "R1A_BASES.json", "R1B_BASES.json", "R1D_ASSOCIATION_DATASET.json"} {
		if _, err := os.Stat(r1fExpDir + "/datasets/" + f); err != nil {
			t.Skipf("input %s absent: %v", f, err)
		}
	}
}

func TestR1G_SelectDataset_DeterministicAndComplete(t *testing.T) {
	r1gSkipIfNoInputs(t)
	a, err := SelectR1GDataset(r1fExpDir, 0, 32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SelectR1GDataset(r1fExpDir, 0, 32)
	if err != nil {
		t.Fatal(err)
	}
	if r1gDatasetFP(a) != r1gDatasetFP(b) {
		t.Fatal("R1-G selection not deterministic")
	}
	if len(a.RealAssoc) != 22 {
		t.Errorf("real assoc bases = %d, want 22", len(a.RealAssoc))
	}
	if len(a.SynAssoc) != 24 {
		t.Errorf("syn assoc bases = %d, want 24", len(a.SynAssoc))
	}
	if len(a.ScaleBases) < 12 || len(a.ContextBases) < 12 || len(a.CueBases) < 8 {
		t.Errorf("pool-derived base counts too small: scale=%d context=%d cue=%d", len(a.ScaleBases), len(a.ContextBases), len(a.CueBases))
	}
	// scale/cue disjoint
	scale := map[string]bool{}
	for _, x := range a.ScaleBases {
		scale[x.CandidateID] = true
	}
	for _, x := range a.CueBases {
		if scale[x.CandidateID] {
			t.Errorf("G-D base %s overlaps G-A", x.BaseID)
		}
	}
	// real assoc labelled + independent estimate false
	for _, x := range a.RealAssoc {
		if !x.InterventionReuse || x.SourceStage != "R1D_REAL_INTERVENTION_REUSE" {
			t.Errorf("%s not labelled intervention reuse", x.BaseID)
		}
		if x.RealCompetitor == "" || x.RealCompetitor == x.Gold {
			t.Errorf("%s: bad competitor %q vs gold %q", x.BaseID, x.RealCompetitor, x.Gold)
		}
	}
	// synthetic invariants
	for _, x := range a.SynAssoc {
		if x.SynValue == x.SynCompValue || x.SynLabel == x.SynCompLabel {
			t.Errorf("%s: synthetic target/competitor collide", x.BaseID)
		}
		if len(x.SynValue) < 2 || len(x.SynValue) > 4 {
			t.Errorf("%s: value %q not multi-digit", x.BaseID, x.SynValue)
		}
	}
}

func TestR1G_ThresholdsFrozen(t *testing.T) {
	if r1gEarnedDelta != 0.20 || r1gPromisingDelta != 0.10 || r1gMaxDegradation != 0.05 || r1gEarnedMcNemarSig != 0.05 {
		t.Error("R1-G verdict thresholds drifted from the frozen values")
	}
}

func TestR1G_VerdictClassifier(t *testing.T) {
	mk := func(delta float64, wc, cw int, degr float64, p float64) R1GRecoveryRow {
		return R1GRecoveryRow{
			N: 20, DegradationRate: degr,
			McNemar: AdjacentTransition{AbsoluteDelta: delta, WrongToCorrect: wc, CorrectToWrong: cw, PValue: p},
		}
	}
	if v := classifyR1GVerdict(mk(0.5, 12, 1, 0.0, 0.001)); v != "EARNED_RECOVERY" {
		t.Errorf("earned -> %s", v)
	}
	if v := classifyR1GVerdict(mk(0.15, 4, 2, 0.0, 0.3)); v != "PROMISING_RECOVERY" {
		t.Errorf("promising -> %s", v)
	}
	if v := classifyR1GVerdict(mk(0.02, 2, 2, 0.0, 1.0)); v != "NO_MEASURED_BENEFIT" {
		t.Errorf("no benefit -> %s", v)
	}
	if v := classifyR1GVerdict(mk(-0.3, 1, 8, 0.4, 0.01)); v != "HARMFUL" {
		t.Errorf("harmful -> %s", v)
	}
	small := mk(0.5, 3, 0, 0, 0.1)
	small.N = 4
	if v := classifyR1GVerdict(small); v != "INSUFFICIENT_EVIDENCE" {
		t.Errorf("insufficient -> %s", v)
	}
}

func TestR1G_SynSceneRenderer_MaskVsIsolateDiffer(t *testing.T) {
	bankPath := r1fExpDir + "/datasets/R1C_GLYPHBANK.json"
	if _, err := os.Stat(bankPath); err != nil {
		t.Skipf("glyph bank absent: %v", err)
	}
	bank, err := LoadGlyphBank(bankPath)
	if err != nil {
		t.Fatal(err)
	}
	base := r1gSynBases(1)[0]
	imgs, err := renderSynAssocConditions(bank, base)
	if err != nil {
		t.Fatal(err)
	}
	if pngHash(imgs[0]) == pngHash(imgs[1]) {
		t.Error("GC_SYN_0 (competitor visible) == GC_SYN_1 (masked)")
	}
	if pngHash(imgs[1]) == pngHash(imgs[2]) {
		t.Error("GC_SYN_1 (masked) == GC_SYN_2 (isolated) — should differ (vertical recentre)")
	}
	again, err := renderSynAssocConditions(bank, base)
	if err != nil {
		t.Fatal(err)
	}
	if pngHash(imgs[0]) != pngHash(again[0]) {
		t.Error("synthetic renderer not deterministic")
	}
}
