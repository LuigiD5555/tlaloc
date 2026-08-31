package learningmemory

const EventSchema = "tlaloc.learning-memory.r0.event"

const (
	EventObservation  = "OBSERVATION"
	EventChange       = "CHANGE_ATTEMPT"
	EventOutcome      = "OUTCOME_LINK"
	EvidenceRealModel = "REAL_MODEL"
	EvidenceSynthetic = "SYNTHETIC"
	EvidenceCI        = "CI"
	EvidenceManual    = "MANUAL"
)

type Event struct {
	Schema             string   `json:"schema"`
	EventID            string   `json:"event_id"`
	EventType          string   `json:"event_type"`
	EvidenceClass      string   `json:"evidence_class"`
	RecordedAt         string   `json:"recorded_at,omitempty"`
	SourceCampaignSHA  string   `json:"source_campaign_sha256,omitempty"`
	SourceResultSHA    string   `json:"source_result_sha256,omitempty"`
	BenchmarkID        string   `json:"benchmark_id,omitempty"`
	TrialID            string   `json:"trial_id,omitempty"`
	ModelID            string   `json:"model_id,omitempty"`
	Provider           string   `json:"provider,omitempty"`
	Condition          string   `json:"condition,omitempty"`
	SpecimenID         string   `json:"specimen_id,omitempty"`
	SpecimenSHA256     string   `json:"specimen_sha256,omitempty"`
	QuestionID         string   `json:"question_id,omitempty"`
	ScoreLayer         string   `json:"score_layer,omitempty"`
	Pass               *bool    `json:"pass,omitempty"`
	Status             string   `json:"status,omitempty"`
	LastCompletedStage string   `json:"last_completed_stage,omitempty"`
	FailureCode        string   `json:"failure_code,omitempty"`
	SelectedCodec      string   `json:"selected_codec,omitempty"`
	OverallScore       *float64 `json:"overall_score,omitempty"`
	OrigamiVersion     string   `json:"origami_version,omitempty"`
	TlalocVersion      string   `json:"tlaloc_version,omitempty"`
	CandidateID        string   `json:"candidate_id,omitempty"`
	ParentEventIDs     []string `json:"parent_event_ids,omitempty"`
	ChangeSummary      string   `json:"change_summary,omitempty"`
	BeforeScore        *float64 `json:"before_score,omitempty"`
	AfterScore         *float64 `json:"after_score,omitempty"`
	Delta              *float64 `json:"delta,omitempty"`
	Tags               []string `json:"tags,omitempty"`
}

type FailurePattern struct {
	Key              string   `json:"key"`
	Stage            string   `json:"stage,omitempty"`
	FailureCode      string   `json:"failure_code,omitempty"`
	ScoreLayer       string   `json:"score_layer,omitempty"`
	Count            int      `json:"count"`
	Models           []string `json:"models,omitempty"`
	Specimens        []string `json:"specimens,omitempty"`
	Questions        []string `json:"questions,omitempty"`
	SuggestedTarget  string   `json:"suggested_target,omitempty"`
}

type CandidateOutcome struct {
	CandidateID string  `json:"candidate_id"`
	Outcomes    int     `json:"outcomes"`
	MeanDelta   float64 `json:"mean_delta"`
	BestDelta   float64 `json:"best_delta"`
	WorstDelta  float64 `json:"worst_delta"`
}

type Summary struct {
	Schema                 string             `json:"schema"`
	StoreRoot              string             `json:"store_root"`
	TotalEvents            int                `json:"total_events"`
	ObservationEvents      int                `json:"observation_events"`
	RealModelObservations  int                `json:"real_model_observations"`
	SyntheticObservations  int                `json:"synthetic_observations"`
	PassedObservations     int                `json:"passed_observations"`
	FailedObservations     int                `json:"failed_observations"`
	ChangeAttempts         int                `json:"change_attempts"`
	OutcomeLinks           int                `json:"outcome_links"`
	TopRealFailurePatterns []FailurePattern   `json:"top_real_failure_patterns,omitempty"`
	CandidateOutcomes      []CandidateOutcome `json:"candidate_outcomes,omitempty"`
	NextDebugTarget        string             `json:"next_debug_target,omitempty"`
}
