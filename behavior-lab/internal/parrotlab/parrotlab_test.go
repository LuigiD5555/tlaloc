package parrotlab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScoreAnswerFamilies(t *testing.T) {
	cases := []struct {
		name            string
		item            Case
		raw             string
		wantContract    bool
		wantSemantic    bool
		wantFormatValid bool
		wantUnsupported bool
	}{
		{"numeric within tolerance", Case{TaskFamily: "numeric", Expected: Expected{Number: ptr(42), Tolerance: 1}}, "42.4", true, true, true, false},
		{"numeric narrated still semantic", Case{TaskFamily: "numeric", Expected: Expected{Number: ptr(42), Tolerance: 1}}, "the answer is about 42.4 folds after counting", false, true, false, false},
		{"choice valid but wrong is not unsupported", Case{TaskFamily: "choice", Choices: []string{"circle", "square"}, Expected: Expected{Value: "circle"}}, "square", false, false, true, false},
		{"choice outside universe is unsupported", Case{TaskFamily: "choice", Choices: []string{"circle", "square"}, Expected: Expected{Value: "circle"}}, "triangle", false, false, false, true},
		{"choice alias", Case{TaskFamily: "choice", Choices: []string{"a", "b"}, Expected: Expected{Value: "b", Aliases: []string{"block b"}}}, "Block B", true, true, true, false},
		{"exact narration is contract failure", Case{TaskFamily: "exact", Expected: Expected{Value: "red"}}, "The larger shape is the square and its colour is red, I think.", false, true, false, false},
		{"abstain expected", Case{TaskFamily: "abstain", Expected: Expected{Abstain: true}}, "UNKNOWN", true, true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScoreAnswer(tc.item, tc.raw)
			if got.ContractSuccess != tc.wantContract || got.SemanticCorrect != tc.wantSemantic ||
				got.FormatValid != tc.wantFormatValid || got.UnsupportedAssertion != tc.wantUnsupported {
				t.Fatalf("ScoreAnswer(%q) = %+v; want contract=%v semantic=%v format=%v unsupported=%v",
					tc.raw, got, tc.wantContract, tc.wantSemantic, tc.wantFormatValid, tc.wantUnsupported)
			}
		})
	}
}

func TestValidateRejectsChoicesConflatedWithAnswer(t *testing.T) {
	// The P-1 failure mode: expected.value not among choices.
	cases := []Case{{
		CaseID: "x-1", Stage: StageInstructionCliff, BaseID: "x", Operations: 1,
		TaskFamily: "choice", Choices: []string{"warm", "cool"},
		Instruction: "classify", Expected: Expected{Value: "hot"},
	}}
	problems := Validate(cases)
	joined := ""
	for _, problem := range problems {
		joined += problem.Error() + "\n"
	}
	if !strings.Contains(joined, "not one of choices") {
		t.Fatalf("expected a choices/answer conflation error, got: %s", joined)
	}
}

func TestComputeGroupP95IsNearMax(t *testing.T) {
	records := make([]RunRecord, 0)
	for _, wall := range []int64{4443, 2498, 3238, 2655, 3566} {
		records = append(records, RunRecord{TaskFamily: "exact",
			Score:     Score{ContractSuccess: true, SemanticCorrect: true, FormatValid: true},
			Resources: Resources{WallMS: wall}})
	}
	stat := computeGroup(records)
	if stat.P95LatencyMS < 3566 {
		t.Fatalf("p95 = %d, expected >= 3566 (near max), not the minimum", stat.P95LatencyMS)
	}
}

func TestPairedTransitionMcNemar(t *testing.T) {
	// 20 stimuli: at OP2 all correct, at OP3 only 5 correct -> 15 regressions,
	// 0 gains -> strongly significant.
	byBaseDepth := map[string]map[int]bool{}
	for index := 0; index < 20; index++ {
		byBaseDepth[string(rune('a'+index))] = map[int]bool{2: true, 3: index < 5}
	}
	transition := pairedTransition(2, 3, byBaseDepth)
	if transition.Regressions != 15 || transition.Gains != 0 {
		t.Fatalf("regressions=%d gains=%d, want 15/0", transition.Regressions, transition.Gains)
	}
	if !transition.Significant || transition.PValue >= 0.05 {
		t.Fatalf("expected significant McNemar, got p=%.5f", transition.PValue)
	}
}

func TestMcNemarExactPBalanced(t *testing.T) {
	if p := mcnemarExactP(5, 5); p < 0.95 {
		t.Fatalf("balanced discordant pairs should be non-significant, got p=%.3f", p)
	}
	if p := mcnemarExactP(0, 0); p != 1 {
		t.Fatalf("no discordant pairs -> p=1, got %.3f", p)
	}
}

func TestBuildCliffDetectsDropViaPairedTest(t *testing.T) {
	records := synthCliff(map[int]float64{1: 0.95, 2: 0.90, 3: 0.35, 4: 0.20, 5: 0.10}, 20)
	cliff := buildCliff(records, 15)
	if !cliff.Detected || cliff.Level != 3 || cliff.MaxSafeOps != 2 {
		t.Fatalf("expected cliff OP3 / MaxSafeOps 2, got detected=%v level=%d safe=%d",
			cliff.Detected, cliff.Level, cliff.MaxSafeOps)
	}
	if len(cliff.Transitions) != 4 {
		t.Fatalf("expected 4 paired transitions, got %d", len(cliff.Transitions))
	}
}

func TestGenerateInstructionCliffIsDeterministicAndValid(t *testing.T) {
	dir := t.TempDir()
	written, err := GenerateInstructionCliff(dir, 42, 10, false)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if written != 50 {
		t.Fatalf("wrote %d cases, want 50 (10 scenes x 5)", written)
	}
	cases, err := LoadCases(filepath.Join(dir, "instruction-cliff.jsonl"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if problems := Validate(cases); len(problems) > 0 {
		t.Fatalf("generated dataset invalid: %v", problems[:1])
	}
	if problems := FreezeValidate(cases); len(problems) > 0 {
		t.Fatalf("generated dataset not freeze-clean: %v", problems[:1])
	}
	// Every added primitive appears at every depth across the four families.
	seen := map[string]map[int]bool{}
	for _, item := range cases {
		if seen[item.AddedPrimitive] == nil {
			seen[item.AddedPrimitive] = map[int]bool{}
		}
		seen[item.AddedPrimitive][item.Operations] = true
	}
	for name, depths := range seen {
		for depth := 1; depth <= 5; depth++ {
			if !depths[depth] {
				t.Errorf("primitive %q never lands on depth %d (Latin square broken)", name, depth)
			}
		}
	}

	second := t.TempDir()
	if _, err := GenerateInstructionCliff(second, 42, 10, false); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	a := mustRead(t, filepath.Join(dir, "instruction-cliff.jsonl"))
	b := mustRead(t, filepath.Join(second, "instruction-cliff.jsonl"))
	if a != b {
		t.Fatal("generator is not deterministic for a fixed seed")
	}
}

func synthCliff(accuracyByDepth map[int]float64, perLevel int) []RunRecord {
	var records []RunRecord
	for depth, accuracy := range accuracyByDepth {
		hits := int(accuracy*float64(perLevel) + 0.5)
		for index := 0; index < perLevel; index++ {
			correct := index < hits
			records = append(records, RunRecord{
				Stage:      StageInstructionCliff,
				Operations: depth,
				BaseID:     string(rune('a' + index)),
				TaskFamily: "exact",
				Score:      Score{ContractSuccess: correct, SemanticCorrect: correct, FormatValid: true, Parsed: "x"},
			})
		}
	}
	return records
}

func ptr(value float64) *float64 { return &value }

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
