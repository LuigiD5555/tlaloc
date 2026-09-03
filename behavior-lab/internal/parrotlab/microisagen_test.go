package parrotlab

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The microisa_visual generator renders ~666 PNGs plus A4 crops from
// 1050x1500 pages; under -race that is far too slow to repeat per test, so
// it is generated exactly once per test binary and shared.
var (
	microISAGenOnce  sync.Once
	microISAGenCases []Case
	microISAGenDir   string
	microISAGenErr   error
)

func generateMicroISAForTest(t *testing.T) ([]Case, string) {
	t.Helper()
	microISAGenOnce.Do(func() {
		dir, err := os.MkdirTemp("", "microisa-gen-")
		if err != nil {
			microISAGenErr = err
			return
		}
		src := "../../experiments/parrot-microisa-r0.1/datasets"
		copyFileBestEffort(filepath.Join(src, "microisa-visual.crops.json"), filepath.Join(dir, "microisa-visual.crops.json"))
		pages, _ := filepath.Glob(filepath.Join(src, "microisa-visual", "pages", "*.png"))
		if len(pages) > 0 {
			_ = os.MkdirAll(filepath.Join(dir, "microisa-visual", "pages"), 0o755)
			for _, page := range pages {
				copyFileBestEffort(page, filepath.Join(dir, "microisa-visual", "pages", filepath.Base(page)))
			}
		}
		if _, err := GenerateMicroISAVisual(dir, 42, false); err != nil {
			microISAGenErr = err
			return
		}
		microISAGenCases, microISAGenErr = LoadCases(filepath.Join(dir, "microisa-visual.jsonl"))
		microISAGenDir = dir
	})
	if microISAGenErr != nil {
		t.Fatalf("generate microisa_visual: %v", microISAGenErr)
	}
	return microISAGenCases, microISAGenDir
}

func copyFileBestEffort(from, to string) {
	if data, err := os.ReadFile(from); err == nil {
		_ = os.WriteFile(to, data, 0o644)
	}
}

func TestMicroISAGeneratorIsDeterministicAndValid(t *testing.T) {
	casesA, dirA := generateMicroISAForTest(t)
	if problems := Validate(casesA); len(problems) > 0 {
		t.Fatalf("validate: %d problems; first: %v", len(problems), problems[0])
	}
	rawA, _ := os.ReadFile(filepath.Join(dirA, "microisa-visual.jsonl"))

	dirB := t.TempDir()
	src := "../../experiments/parrot-microisa-r0.1/datasets"
	copyFileBestEffort(filepath.Join(src, "microisa-visual.crops.json"), filepath.Join(dirB, "microisa-visual.crops.json"))
	pages, _ := filepath.Glob(filepath.Join(src, "microisa-visual", "pages", "*.png"))
	_ = os.MkdirAll(filepath.Join(dirB, "microisa-visual", "pages"), 0o755)
	for _, page := range pages {
		copyFileBestEffort(page, filepath.Join(dirB, "microisa-visual", "pages", filepath.Base(page)))
	}
	if _, err := GenerateMicroISAVisual(dirB, 42, false); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	rawB, _ := os.ReadFile(filepath.Join(dirB, "microisa-visual.jsonl"))
	if string(rawA) != string(rawB) {
		t.Fatal("generation is not deterministic across runs")
	}

	// A1 = one observation per base stimulus.
	a1 := filterCases(casesA, "A1")
	a1Bases := map[string]bool{}
	for _, item := range a1 {
		if item.Condition != "canonical" || item.VariedDim != "" {
			t.Fatalf("A1 case %s is not canonical", item.CaseID)
		}
		a1Bases[item.BaseID] = true
	}
	if len(a1) != 260 || len(a1Bases) != 260 {
		t.Fatalf("A1: want 260 cases / 260 bases, got %d / %d", len(a1), len(a1Bases))
	}

	// No answer text leaks into the instruction (choice families legitimately
	// list every option, so only entity/numeric are checked).
	for _, item := range casesA {
		if item.TaskFamily == "choice" {
			continue
		}
		answer := strings.ToLower(item.Expected.Value)
		if answer != "" && len(answer) >= 4 && strings.Contains(strings.ToLower(item.Instruction), answer) {
			t.Fatalf("case %s leaks its answer %q into the instruction", item.CaseID, answer)
		}
	}
}

func TestMicroISALaddersAreNestedByBaseID(t *testing.T) {
	cases, _ := generateMicroISAForTest(t)
	byBase := map[string][]Case{}
	for _, item := range filterCases(cases, "A2") {
		if item.VariedDim == "reference_type" {
			continue
		}
		byBase[item.BaseID] = append(byBase[item.BaseID], item)
	}
	if len(byBase) == 0 {
		t.Fatal("no A2 ladder bases")
	}
	for base, rungs := range byBase {
		dims := map[string]bool{}
		for _, item := range rungs {
			dims[item.VariedDim] = true
			if item.BaseID != base {
				t.Fatalf("rung %s has base %s, want %s", item.CaseID, item.BaseID, base)
			}
		}
		if len(dims) != 1 {
			t.Fatalf("base %s mixes ladder dimensions: %v", base, dims)
		}
	}
}

func TestMicroISAReferenceBlockIsCategorical(t *testing.T) {
	cases, _ := generateMicroISAForTest(t)
	seen := map[string]int{}
	for _, item := range filterCases(cases, "A2") {
		if item.VariedDim != "reference_type" {
			continue
		}
		seen[strings.TrimPrefix(item.Condition, "reftype=")]++
	}
	for _, kind := range microISAReferenceTypes {
		if seen[kind] != microISARefBases {
			t.Fatalf("reference type %q: want %d, got %d", kind, microISARefBases, seen[kind])
		}
	}
}

func TestSyntheticReadingStringsUseOnlyTheUnambiguousAlphabet(t *testing.T) {
	allowedLetters := map[rune]bool{}
	for _, symbol := range letterAlphabet {
		allowedLetters[symbol] = true
	}
	// The r0 abort: full A-Z let a Q render like O. These must never recur.
	for _, banned := range []rune{'B', 'G', 'I', 'J', 'L', 'O', 'Q', 'S', 'V', 'W', 'Z'} {
		if allowedLetters[banned] {
			t.Fatalf("ambiguous letter %q is back in letterAlphabet", banned)
		}
	}
	allowedDigits := map[rune]bool{}
	for _, symbol := range digitAlphabet {
		allowedDigits[symbol] = true
	}
	for _, banned := range []rune{'0', '1', '5', '8'} {
		if allowedDigits[banned] {
			t.Fatalf("ambiguous digit %q is back in digitAlphabet", banned)
		}
	}

	for _, word := range nounBank {
		for _, symbol := range word {
			if !allowedLetters[symbol] {
				t.Fatalf("nounBank word %q contains disallowed letter %q", word, symbol)
			}
		}
	}

	cases, _ := generateMicroISAForTest(t)
	for _, item := range cases {
		if len(item.Capabilities) != 1 {
			continue
		}
		switch item.Capabilities[0] {
		case "READ_SHORT_LABEL", "READ_SHORT_TEXT":
			if item.Source != "synthetic" {
				continue
			}
			for _, symbol := range item.Expected.Value {
				if !allowedLetters[symbol] {
					t.Fatalf("%s %s target %q contains disallowed glyph %q",
						item.Capabilities[0], item.CaseID, item.Expected.Value, symbol)
				}
			}
		}
	}
}

func filterCases(cases []Case, sub string) []Case {
	var out []Case
	for _, item := range cases {
		if item.SubExperiment == sub {
			out = append(out, item)
		}
	}
	return out
}
