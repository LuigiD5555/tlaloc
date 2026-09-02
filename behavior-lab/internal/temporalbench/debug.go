package temporalbench

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const debugMarker = "ORIGAMI_DEBUG_R0="

var orderedStages = []string{
	"NONE",
	"BOOT",
	"ROSETTA",
	"CODEC_DISCOVERY",
	"T2_NAVIGATION",
	"SEMANTIC_DECODE",
	"TEMPORAL_ROUTE",
	"TEMPORAL_STEP",
	"EXACT_BOUNDARY",
	"ANSWER",
}

var validStatuses = setOf("PASS", "FAIL", "UNKNOWN", "NOT_VERIFIED", "CAPABILITY_MISMATCH")
var validStages = setOf(orderedStages...)
var validFailureCodes = setOf(
	"NONE",
	"NO_VISUAL_SIGNAL",
	"BOOT_NOT_FOUND",
	"ROSETTA_NOT_FOUND",
	"CODEC_NOT_FOUND",
	"CAPABILITY_MISMATCH",
	"T2_NOT_FOUND",
	"SEMANTIC_EVIDENCE_INSUFFICIENT",
	"TEMPORAL_RULE_AMBIGUOUS",
	"CHECKPOINT_NOT_FOUND",
	"EXACT_DECODER_REQUIRED",
	"UNSUPPORTED_OPERATION",
	"OTHER",
)

func DiagnosticInstruction() string {
	return `DIAGNOSTIC MODE (test-only): Answer the question normally. Then append exactly one line beginning ORIGAMI_DEBUG_R0= followed by a JSON object with schema, status, last_completed_stage, selected_codec, last_instruction, next_instruction, failure_code, evidence_refs, confidence, and optional note. Report only observable protocol progress, not private reasoning or chain-of-thought. Use stages NONE|BOOT|ROSETTA|CODEC_DISCOVERY|T2_NAVIGATION|SEMANTIC_DECODE|TEMPORAL_ROUTE|TEMPORAL_STEP|EXACT_BOUNDARY|ANSWER. Use status PASS|FAIL|UNKNOWN|NOT_VERIFIED|CAPABILITY_MISMATCH. If no failure, failure_code must be NONE.`
}

func ParseDebugFromResponse(r Response) (string, *DebugTrace, []string) {
	if r.Debug != nil {
		trace := *r.Debug
		return r.Text, &trace, validateDebugTrace(trace)
	}
	idx := strings.LastIndex(r.Text, debugMarker)
	if idx < 0 {
		return r.Text, nil, nil
	}
	answer := strings.TrimSpace(r.Text[:idx])
	raw := strings.TrimSpace(r.Text[idx+len(debugMarker):])
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimPrefix(raw, "json")
		raw = strings.TrimSpace(raw)
		if end := strings.Index(raw, "```"); end >= 0 {
			raw = strings.TrimSpace(raw[:end])
		}
	}
	var trace DebugTrace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		return answer, nil, []string{"DEBUG_TRACE_INVALID_JSON"}
	}
	return answer, &trace, validateDebugTrace(trace)
}

func validateDebugTrace(t DebugTrace) []string {
	v := []string{}
	if t.Schema != DebugTraceSchema {
		v = append(v, "DEBUG_TRACE_SCHEMA")
	}
	status := strings.ToUpper(strings.TrimSpace(t.Status))
	stage := strings.ToUpper(strings.TrimSpace(t.LastCompletedStage))
	failure := strings.ToUpper(strings.TrimSpace(t.FailureCode))
	if !validStatuses[status] {
		v = append(v, "DEBUG_STATUS_INVALID")
	}
	if !validStages[stage] {
		v = append(v, "DEBUG_STAGE_INVALID")
	}
	if failure == "" {
		failure = "NONE"
	}
	if !validFailureCodes[failure] {
		v = append(v, "DEBUG_FAILURE_CODE_INVALID")
	}
	if status == "PASS" && failure != "NONE" {
		v = append(v, "PASS_WITH_FAILURE_CODE")
	}
	if status != "PASS" && failure == "NONE" {
		v = append(v, "FAILURE_WITHOUT_CODE")
	}
	if t.Confidence < 0 || t.Confidence > 1 {
		v = append(v, "DEBUG_CONFIDENCE_RANGE")
	}
	if len(t.Note) > 160 {
		v = append(v, "DEBUG_NOTE_TOO_LONG")
	}
	return v
}

func makeDebugResult(q QuestionResult, response Response, diagnostic bool) DebugResult {
	answer, trace, violations := ParseDebugFromResponse(response)
	_ = answer
	out := DebugResult{QuestionID: q.QuestionID, Present: trace != nil, Violations: append([]string(nil), violations...)}
	if trace == nil {
		if diagnostic && len(violations) == 0 {
			out.Violations = append(out.Violations, "DEBUG_TRACE_MISSING")
		}
		return out
	}
	out.Valid = len(violations) == 0
	out.Status = strings.ToUpper(strings.TrimSpace(trace.Status))
	out.LastCompletedStage = strings.ToUpper(strings.TrimSpace(trace.LastCompletedStage))
	out.SelectedCodec = strings.TrimSpace(trace.SelectedCodec)
	out.LastInstruction = strings.TrimSpace(trace.LastInstruction)
	out.NextInstruction = strings.TrimSpace(trace.NextInstruction)
	out.FailureCode = strings.ToUpper(strings.TrimSpace(trace.FailureCode))
	if out.FailureCode == "" {
		out.FailureCode = "NONE"
	}
	out.EvidenceRefs = append([]string(nil), trace.EvidenceRefs...)
	out.Confidence = trace.Confidence
	if q.Pass {
		allowed := out.Status == "PASS"
		if q.QuestionID == "Q8" && (out.Status == "UNKNOWN" || out.Status == "NOT_VERIFIED") {
			allowed = true
		}
		if !allowed {
			out.Violations = append(out.Violations, "ANSWER_TRACE_MISMATCH")
		}
	}
	if !q.Pass && out.Status == "PASS" {
		out.Violations = append(out.Violations, "ANSWER_TRACE_MISMATCH")
	}
	return out
}

func summarizeDebug(reports []DebugResult, diagnostic bool) *DebugSummary {
	if !diagnostic && len(reports) == 0 {
		return nil
	}
	s := &DebugSummary{DiagnosticMode: diagnostic}
	if len(reports) == 0 {
		return s
	}
	present, consistent := 0, 0
	frontierCount := map[string]int{}
	failureCount := map[string]int{}
	earliestRank := len(orderedStages) + 1
	furthestRank := -1
	for _, r := range reports {
		if !r.Present {
			s.MissingTraceCount++
			continue
		}
		present++
		if !r.Valid {
			s.InvalidTraceCount++
		}
		mismatch := false
		for _, v := range r.Violations {
			if v == "ANSWER_TRACE_MISMATCH" {
				mismatch = true
				s.AnswerTraceMismatchCount++
			}
		}
		if r.Valid && !mismatch {
			consistent++
		}
		rank := stageRank(r.LastCompletedStage)
		if rank > furthestRank {
			furthestRank = rank
			s.FurthestCompletedStage = r.LastCompletedStage
		}
		if r.Status != "PASS" {
			frontierCount[r.LastCompletedStage]++
			if rank >= 0 && rank < earliestRank {
				earliestRank = rank
				s.EarliestFailureFrontier = r.LastCompletedStage
			}
			if r.FailureCode != "" && r.FailureCode != "NONE" {
				failureCount[r.FailureCode]++
			}
		}
	}
	s.TraceCoverage = float64(present) / float64(len(reports))
	if present > 0 {
		s.TraceConsistencyScore = float64(consistent) / float64(present)
	}
	s.DominantFailureFrontier = modeKey(frontierCount)
	s.MostCommonFailureCode = modeKey(failureCount)
	return s
}

func stageRank(stage string) int {
	stage = strings.ToUpper(strings.TrimSpace(stage))
	for i, s := range orderedStages {
		if s == stage {
			return i
		}
	}
	return -1
}

func modeKey(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best := ""
	bestN := -1
	for _, k := range keys {
		if m[k] > bestN {
			best = k
			bestN = m[k]
		}
	}
	return best
}

func setOf(in ...string) map[string]bool {
	out := map[string]bool{}
	for _, s := range in {
		out[s] = true
	}
	return out
}

func FormatDebugExample() string {
	example := DebugTrace{Schema: DebugTraceSchema, Status: "FAIL", LastCompletedStage: "ROSETTA", SelectedCodec: "ST2", LastInstruction: "READ_ROSETTA", NextInstruction: "LOCATE_T2", FailureCode: "T2_NOT_FOUND", EvidenceRefs: []string{"T0", "T1"}, Confidence: 0.72, Note: "T2 could not be located from visible evidence."}
	b, _ := json.Marshal(example)
	return fmt.Sprintf("%s%s", debugMarker, string(b))
}
