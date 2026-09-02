package temporalbench

import (
	"strings"
	"testing"
)

func TestParseDebugFooter(t *testing.T) {
	text := `I can see BOOT and ROSETTA but cannot locate T2.
ORIGAMI_DEBUG_R0={"schema":"tlaloc.origami-debug-trace.r0","status":"FAIL","last_completed_stage":"ROSETTA","selected_codec":"ST2","last_instruction":"READ_ROSETTA","next_instruction":"LOCATE_T2","failure_code":"T2_NOT_FOUND","evidence_refs":["T0","T1"],"confidence":0.72}`
	answer, trace, violations := ParseDebugFromResponse(Response{QuestionID: "Q3", Text: text})
	if trace == nil {
		t.Fatal("expected debug trace")
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %#v", violations)
	}
	if strings.Contains(answer, "ORIGAMI_DEBUG_R0") {
		t.Fatal("debug footer leaked into scored answer")
	}
	if trace.LastCompletedStage != "ROSETTA" || trace.FailureCode != "T2_NOT_FOUND" {
		t.Fatalf("unexpected trace: %#v", trace)
	}
}

func TestDebugTraceRejectsPassWithFailure(t *testing.T) {
	trace := DebugTrace{Schema: DebugTraceSchema, Status: "PASS", LastCompletedStage: "ANSWER", FailureCode: "T2_NOT_FOUND", Confidence: 1}
	v := validateDebugTrace(trace)
	if !contains(v, "PASS_WITH_FAILURE_CODE") {
		t.Fatalf("expected PASS_WITH_FAILURE_CODE: %#v", v)
	}
}

func TestDebugSummaryLocatesFailureFrontier(t *testing.T) {
	reports := []DebugResult{
		{QuestionID: "Q0", Present: true, Valid: true, Status: "PASS", LastCompletedStage: "ROSETTA", FailureCode: "NONE"},
		{QuestionID: "Q3", Present: true, Valid: true, Status: "FAIL", LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"},
		{QuestionID: "Q4", Present: true, Valid: true, Status: "FAIL", LastCompletedStage: "T2_NAVIGATION", FailureCode: "SEMANTIC_EVIDENCE_INSUFFICIENT"},
		{QuestionID: "Q7", Present: true, Valid: true, Status: "FAIL", LastCompletedStage: "ROSETTA", FailureCode: "T2_NOT_FOUND"},
	}
	s := summarizeDebug(reports, true)
	if s.TraceCoverage != 1 {
		t.Fatalf("coverage=%v", s.TraceCoverage)
	}
	if s.DominantFailureFrontier != "ROSETTA" {
		t.Fatalf("frontier=%s", s.DominantFailureFrontier)
	}
	if s.EarliestFailureFrontier != "ROSETTA" {
		t.Fatalf("earliest=%s", s.EarliestFailureFrontier)
	}
	if s.FurthestCompletedStage != "T2_NAVIGATION" {
		t.Fatalf("furthest=%s", s.FurthestCompletedStage)
	}
	if s.MostCommonFailureCode != "T2_NOT_FOUND" {
		t.Fatalf("failure=%s", s.MostCommonFailureCode)
	}
}

func TestTargetedDiagnosticRetryScoresOnlyRequestedQuestion(t *testing.T) {
	trial := Trial{
		ID: "diag-q3", ModelID: "SYNTHETIC", Condition: "NATIVE_PNG_ONLY", DiagnosticMode: true,
		DiagnosticQuestionIDs: []string{"Q3"}, Specimen: Specimen{ID: "s"},
		Responses: []Response{{QuestionID: "Q3", Text: `Cannot locate T2.
ORIGAMI_DEBUG_R0={"schema":"tlaloc.origami-debug-trace.r0","status":"FAIL","last_completed_stage":"ROSETTA","selected_codec":"ST2","last_instruction":"READ_ROSETTA","next_instruction":"LOCATE_T2","failure_code":"T2_NOT_FOUND","evidence_refs":["T0","T1"],"confidence":0.7}`}},
	}
	got := EvaluateTrial(trial)
	if len(got.Questions) != 1 || got.Questions[0].QuestionID != "Q3" {
		t.Fatalf("unexpected questions: %#v", got.Questions)
	}
	if got.MissingQuestionCount != 0 {
		t.Fatalf("missing=%d", got.MissingQuestionCount)
	}
	if got.DebugSummary == nil || got.DebugSummary.TraceCoverage != 1 {
		t.Fatalf("summary=%#v", got.DebugSummary)
	}
	if got.DebugSummary.DominantFailureFrontier != "ROSETTA" {
		t.Fatalf("frontier=%s", got.DebugSummary.DominantFailureFrontier)
	}
}

func TestDiagnosticInstructionDoesNotRequestChainOfThought(t *testing.T) {
	s := strings.ToLower(DiagnosticInstruction())
	if !strings.Contains(s, "not private reasoning") {
		t.Fatal("must explicitly exclude private reasoning")
	}
	if !strings.Contains(s, "origami_debug_r0=") {
		t.Fatal("missing machine-readable marker")
	}
}

func contains(in []string, want string) bool {
	for _, v := range in {
		if v == want {
			return true
		}
	}
	return false
}
