package temporalbench

const (
	CampaignSchema   = "tlaloc.temporal-native-benchmark.r0.campaign"
	ResultSchema     = "tlaloc.temporal-native-benchmark.r0.result"
	DebugTraceSchema = "tlaloc.origami-debug-trace.r0"
)

type Specimen struct {
	ID       string `json:"id"`
	SHA256   string `json:"sha256"`
	Variant  string `json:"variant"`
	PNGBytes int    `json:"png_bytes,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type DebugTrace struct {
	Schema             string   `json:"schema"`
	Status             string   `json:"status"`
	LastCompletedStage string   `json:"last_completed_stage"`
	SelectedCodec      string   `json:"selected_codec,omitempty"`
	LastInstruction    string   `json:"last_instruction,omitempty"`
	NextInstruction    string   `json:"next_instruction,omitempty"`
	FailureCode        string   `json:"failure_code,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	Confidence         float64  `json:"confidence,omitempty"`
	Note               string   `json:"note,omitempty"`
}

type Response struct {
	QuestionID string      `json:"question_id"`
	Text       string      `json:"text"`
	LatencyMS  int64       `json:"latency_ms,omitempty"`
	Debug      *DebugTrace `json:"debug,omitempty"`
}

type Trial struct {
	ID                    string     `json:"id"`
	ModelID               string     `json:"model_id"`
	Provider              string     `json:"provider,omitempty"`
	Condition             string     `json:"condition"`
	DiagnosticMode        bool       `json:"diagnostic_mode,omitempty"`
	DiagnosticQuestionIDs []string   `json:"diagnostic_question_ids,omitempty"`
	Specimen              Specimen   `json:"specimen"`
	Responses             []Response `json:"responses"`
}

type Campaign struct {
	Schema      string  `json:"schema"`
	BenchmarkID string  `json:"benchmark_id"`
	Trials      []Trial `json:"trials"`
}

type QuestionResult struct {
	QuestionID string   `json:"question_id"`
	Layer      string   `json:"layer"`
	Pass       bool     `json:"pass"`
	Score      float64  `json:"score"`
	Missing    []string `json:"missing,omitempty"`
	Violations []string `json:"violations,omitempty"`
}

type DebugResult struct {
	QuestionID         string   `json:"question_id"`
	Present            bool     `json:"present"`
	Valid              bool     `json:"valid"`
	Status             string   `json:"status,omitempty"`
	LastCompletedStage string   `json:"last_completed_stage,omitempty"`
	SelectedCodec      string   `json:"selected_codec,omitempty"`
	LastInstruction    string   `json:"last_instruction,omitempty"`
	NextInstruction    string   `json:"next_instruction,omitempty"`
	FailureCode        string   `json:"failure_code,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	Confidence         float64  `json:"confidence,omitempty"`
	Violations         []string `json:"violations,omitempty"`
}

type DebugSummary struct {
	DiagnosticMode           bool    `json:"diagnostic_mode"`
	TraceCoverage            float64 `json:"trace_coverage"`
	TraceConsistencyScore    float64 `json:"trace_consistency_score"`
	DominantFailureFrontier  string  `json:"dominant_failure_frontier,omitempty"`
	EarliestFailureFrontier  string  `json:"earliest_failure_frontier,omitempty"`
	FurthestCompletedStage   string  `json:"furthest_completed_stage,omitempty"`
	MostCommonFailureCode    string  `json:"most_common_failure_code,omitempty"`
	MissingTraceCount        int     `json:"missing_trace_count"`
	InvalidTraceCount        int     `json:"invalid_trace_count"`
	AnswerTraceMismatchCount int     `json:"answer_trace_mismatch_count"`
}

type LayerScore struct {
	Layer  string  `json:"layer"`
	Passed int     `json:"passed"`
	Total  int     `json:"total"`
	Score  float64 `json:"score"`
}

type TrialResult struct {
	TrialID              string           `json:"trial_id"`
	ModelID              string           `json:"model_id"`
	Condition            string           `json:"condition"`
	DiagnosticMode       bool             `json:"diagnostic_mode,omitempty"`
	SpecimenID           string           `json:"specimen_id"`
	Questions            []QuestionResult `json:"questions"`
	DebugReports         []DebugResult    `json:"debug_reports,omitempty"`
	DebugSummary         *DebugSummary    `json:"debug_summary,omitempty"`
	Layers               []LayerScore     `json:"layers"`
	OverallScore         float64          `json:"overall_score"`
	SelfBootstrapScore   float64          `json:"self_bootstrap_score"`
	TemporalReasoning    float64          `json:"temporal_reasoning_score"`
	ExactHonesty         float64          `json:"exact_honesty_score"`
	InventedExactClaims  int              `json:"invented_exact_claims"`
	MissingQuestionCount int              `json:"missing_question_count"`
}

type Comparison struct {
	ModelID          string  `json:"model_id"`
	SpecimenID       string  `json:"specimen_id"`
	NativeScore      float64 `json:"native_score"`
	AssistedScore    float64 `json:"assisted_score"`
	AssistanceGain   float64 `json:"assistance_gain"`
	PristineScore    float64 `json:"pristine_score"`
	DegradedScore    float64 `json:"degraded_score"`
	DegradationDelta float64 `json:"degradation_delta"`
}

type Result struct {
	Schema       string        `json:"schema"`
	BenchmarkID  string        `json:"benchmark_id"`
	Trials       []TrialResult `json:"trials"`
	Comparisons  []Comparison  `json:"comparisons,omitempty"`
	RealEvidence bool          `json:"real_evidence"`
}
