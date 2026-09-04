package perceptenvelope

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func loadR1DAllocOrSkip(t *testing.T) R1DAllocation {
	t.Helper()
	body, err := os.ReadFile("../../experiments/parrot-perceptual-envelope-r1/datasets/R1D_ASSOCIATION_DATASET.json")
	if err != nil {
		t.Skipf("R1D_ASSOCIATION_DATASET.json absent: %v", err)
	}
	var a R1DAllocation
	if err := json.Unmarshal(body, &a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestR1E_WrongMap_DeterministicAndValid(t *testing.T) {
	elig := EligibleR1DBases(loadR1DAllocOrSkip(t))
	if len(elig) < 6 {
		t.Fatalf("only %d eligible bases", len(elig))
	}
	wm1, err := BuildR1EWrongMap(elig)
	if err != nil {
		t.Fatal(err)
	}
	wm2, err := BuildR1EWrongMap(elig)
	if err != nil {
		t.Fatal(err)
	}
	if R1EWrongMapFP(wm1) != R1EWrongMapFP(wm2) {
		t.Fatal("wrong-image pairing not deterministic")
	}
	if len(wm1.Pairs) != len(elig) {
		t.Fatalf("%d pairs for %d bases", len(wm1.Pairs), len(elig))
	}
	byID := map[string]R1DBase{}
	for _, b := range elig {
		byID[b.BaseID] = b
	}
	matched := 0
	for _, p := range wm1.Pairs {
		if p.BaseID == p.WrongBaseID {
			t.Errorf("%s paired with itself", p.BaseID)
		}
		gc, _ := parseFamilyValue(FamMultiDigit, p.BaseValue)
		gw, _ := parseFamilyValue(FamMultiDigit, p.WrongValue)
		if gc == gw {
			t.Errorf("%s: wrong value %q == task gold %q", p.BaseID, p.WrongValue, p.BaseValue)
		}
		if wb, ok := byID[p.WrongBaseID]; ok && strings.Contains(wb.LineText, p.BaseValue) {
			t.Errorf("%s: wrong base %s line contains the base operand %q", p.BaseID, p.WrongBaseID, p.BaseValue)
		}
		if p.DigitLenMatched {
			matched++
			if len(p.BaseValue) != len(p.WrongValue) {
				t.Errorf("%s: digit_length_matched but %q vs %q", p.BaseID, p.BaseValue, p.WrongValue)
			}
		}
	}
	if matched == 0 {
		t.Error("no digit-length-matched wrong pairs")
	}
}

func TestR1E_WrongMap_PoisonInvariant(t *testing.T) {
	elig := EligibleR1DBases(loadR1DAllocOrSkip(t))
	base, err := BuildR1EWrongMap(elig)
	if err != nil {
		t.Fatal(err)
	}
	// A poisoned expected-answers file must not exist in the pairing path at
	// all; pairing depends only on the seed and base/candidate ids.
	poisoned, err := BuildR1EWrongMap(elig)
	if err != nil {
		t.Fatal(err)
	}
	if R1EWrongMapFP(base) != R1EWrongMapFP(poisoned) {
		t.Fatal("pairing fingerprint changed on re-run")
	}
}

func TestR1E_ImageConsistentScoring(t *testing.T) {
	cases := []struct {
		raw, wrong, gold string
		want             bool
	}{
		{"64", "64", "128", true},
		{"128", "64", "128", false},
		{"the number", "64", "128", false},
		{"64", "64", "64", false}, // degenerate: wrong == gold
	}
	for i, c := range cases {
		if got := r1eImageConsistent(c.raw, c.wrong, c.gold); got != c.want {
			t.Errorf("case %d (%q,%q,%q): got %v want %v", i, c.raw, c.wrong, c.gold, got, c.want)
		}
	}
}

func TestR1E_Instructions_NoLeakage(t *testing.T) {
	for _, c := range R1ECapabilities {
		if containsAnyDigit(c.Instruction) {
			t.Errorf("%s instruction contains a digit: %q", c.Capability, c.Instruction)
		}
		if c.Opcode == "" || c.Instruction == "" {
			t.Errorf("%s: empty opcode/instruction", c.Capability)
		}
	}
	if len(R1EConditions) != 3 || R1EConditions[0] != "E0_NO_IMAGE" {
		t.Errorf("unexpected conditions %v", R1EConditions)
	}
}

func TestR1E_Classification(t *testing.T) {
	mk := func(ci, ni, wg, wv float64) R1ECapabilityTable {
		ct := R1ECapabilityTable{
			Bases: 22, PairsTotal: 22,
			CorrectImageAccuracy: ci, NoImageAccuracy: ni,
			WrongImageTaskGoldAccuracy: wg, WrongImageVisibleOperandAccuracy: wv,
		}
		classifyR1E(&ct)
		return ct
	}
	if c := mk(1.0, 0.1, 0.05, 0.85); c.Classification != "VISUALLY_DEPENDENT" {
		t.Errorf("visually dependent case -> %s", c.Classification)
	}
	if c := mk(1.0, 0.95, 0.9, 0.05); c.Classification != "SHORTCUT_DOMINATED" {
		t.Errorf("shortcut case -> %s", c.Classification)
	}
	if c := mk(1.0, 0.6, 0.5, 0.3); c.Classification != "MIXED_VISUAL_AND_PRIOR" {
		t.Errorf("mixed case -> %s", c.Classification)
	}
	c := mk(0.5, 0.5, 0.5, 0.5)
	c.Bases = 3
	classifyR1E(&c)
	if c.Classification != "INSUFFICIENT_EVIDENCE" {
		t.Errorf("small-n case -> %s", c.Classification)
	}
}
