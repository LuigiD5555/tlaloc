package perceptenvelope

import (
	"os"
	"testing"
)

const r1fExpDir = "../../experiments/parrot-perceptual-envelope-r1"

func r1fSkipIfNoSources(t *testing.T) {
	t.Helper()
	for _, f := range []string{"R1B_RECORDS.json", "R1A1_RECORDS.json", "R1D_DISTRACTOR_RECORDS.json", "R1E_RECORDS.json"} {
		if _, err := os.Stat(r1fExpDir + "/results/" + f); err != nil {
			t.Skipf("source result %s absent: %v", f, err)
		}
	}
}

func TestR1F_SentinelSelection_DeterministicAndComplete(t *testing.T) {
	r1fSkipIfNoSources(t)
	a, err := SelectR1FSentinels(r1fExpDir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SelectR1FSentinels(r1fExpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(R1FStrata)*r1fSentinelsPerStratum {
		t.Fatalf("selected %d sentinels, want 20", len(a))
	}
	perStratum := map[string]int{}
	ids := map[string]bool{}
	for i := range a {
		if a[i].SentinelID != b[i].SentinelID || a[i].RankKey != b[i].RankKey || a[i].BaseID != b[i].BaseID {
			t.Fatalf("selection not deterministic at %d: %+v vs %+v", i, a[i], b[i])
		}
		perStratum[a[i].Stratum]++
		if ids[a[i].SentinelID] {
			t.Errorf("duplicate sentinel id %s", a[i].SentinelID)
		}
		ids[a[i].SentinelID] = true
		if a[i].Capability != string(FrozenOpcode) && a[i].Capability != R1DAssocOpcode {
			t.Errorf("%s: unexpected capability %q", a[i].SentinelID, a[i].Capability)
		}
		if containsAnyDigit(a[i].Instruction) {
			t.Errorf("%s: instruction has a digit", a[i].SentinelID)
		}
	}
	for _, st := range R1FStrata {
		if perStratum[st.Key] != r1fSentinelsPerStratum {
			t.Errorf("stratum %s: %d sentinels, want 4", st.Key, perStratum[st.Key])
		}
	}
}

func TestR1F_StratumInvariants(t *testing.T) {
	r1fSkipIfNoSources(t)
	sent, err := SelectR1FSentinels(r1fExpDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sent {
		switch s.Stratum {
		case "A":
			if !s.PrevSemanticCorrect {
				t.Errorf("%s (A): source not correct", s.SentinelID)
			}
			if s.SourceCondition != "B4" || !s.HasImage {
				t.Errorf("%s (A): bad source cond / no image", s.SentinelID)
			}
		case "B":
			if s.PrevSemanticCorrect || s.SourceCondition != "B0" {
				t.Errorf("%s (B): expected B0 wrong", s.SentinelID)
			}
		case "C":
			if s.PrevSemanticCorrect || s.SourceCondition != "A1C6_FULL_VIEWPORT" {
				t.Errorf("%s (C): expected A1C6 wrong", s.SentinelID)
			}
		case "D":
			if s.PrevSemanticCorrect || (s.SourceCondition != "D1K1" && s.SourceCondition != "D1K2") {
				t.Errorf("%s (D): expected D1K1/K2 wrong", s.SentinelID)
			}
			if s.Capability != R1DAssocOpcode {
				t.Errorf("%s (D): capability %q", s.SentinelID, s.Capability)
			}
		case "E":
			if s.HasImage || s.SourceCondition != "E0_NO_IMAGE" {
				t.Errorf("%s (E): expected no-image E0", s.SentinelID)
			}
			if s.PrevRawOutput != "12345" {
				t.Errorf("%s (E): prev raw %q, want 12345", s.SentinelID, s.PrevRawOutput)
			}
		}
	}
}

func TestR1F_StabilityHelpers(t *testing.T) {
	if empiricalEntropy([]string{"a", "a", "a", "a", "a"}) != 0 {
		t.Error("entropy of constant should be 0")
	}
	if h := empiricalEntropy([]string{"a", "b"}); h != 1 {
		t.Errorf("entropy of {a,b} = %v, want 1", h)
	}
	if flipCount([]bool{false, false, true, false, false}) != 2 {
		t.Error("flipCount")
	}
	if !allFalse([]bool{false, false}) || allFalse([]bool{false, true}) {
		t.Error("allFalse")
	}
	_, freq := modeFrequency([]string{"x", "x", "y"})
	if freq != 2 {
		t.Errorf("mode freq %d want 2", freq)
	}
}

func TestR1F_BuildDataset_DecisionRuleFrozen(t *testing.T) {
	ds := BuildR1FDataset(nil, 0, 32)
	if ds.RepeatsPerSentinel != 5 || len(ds.RepeatIDs) != 5 {
		t.Error("repeats not 5")
	}
	if !contains(ds.DecisionRule, "BLIND_RETRY_NOT_USEFUL") || !contains(ds.DecisionRule, "not redefinable") {
		t.Errorf("decision rule not frozen: %q", ds.DecisionRule)
	}
	if r1fWrongStayWrongThreshold != 0.90 || r1fSemanticInvariantThreshold != 0.90 {
		t.Error("decision thresholds drifted from the frozen 0.90")
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0) }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
