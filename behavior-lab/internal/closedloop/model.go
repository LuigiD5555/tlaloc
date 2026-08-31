package closedloop

import "tlaloc.local/behaviorlab/internal/visualsearch"

const (
	ConfigSchema   = "tlaloc.closed-experimental-loop.r0.config"
	ReportSchema   = "tlaloc.closed-experimental-loop.r0.report"
	OutcomeNative  = "NATIVE_SCORE"
	OutcomeOverall = "OVERALL_SCORE"
)

type ModelConfig struct {
	Name             string  `json:"name"`
	Provider         string  `json:"provider,omitempty"`
	BaseURL          string  `json:"base_url"`
	Model            string  `json:"model"`
	Compatibility    string  `json:"compatibility_strategy,omitempty"`
	TraceStream      bool    `json:"trace_stream,omitempty"`
	MaxOutputTokens  int     `json:"max_output_tokens,omitempty"`
	GenerationGuard  string  `json:"generation_guard,omitempty"`
	APIKeyEnv        string  `json:"api_key_env,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	TimeoutSeconds   int     `json:"timeout_seconds,omitempty"`
	TransportRetries int     `json:"transport_retries,omitempty"`
}

type SpecimenConfig struct {
	ID  string `json:"id"`
	PNG string `json:"png"`
}

type CandidateConfig struct {
	ID               string                  `json:"id"`
	PNG              string                  `json:"png"`
	BaseProfileID    string                  `json:"base_profile_id"`
	ParentSpecimenID string                  `json:"parent_specimen_id,omitempty"`
	Mutations        []visualsearch.Mutation `json:"mutations"`
	BuildCommand     []string                `json:"build_command,omitempty"`
}

func (c CandidateConfig) VisualCandidate() visualsearch.Candidate {
	return visualsearch.Candidate{
		Schema:        visualsearch.SchemaR0 + ".candidate",
		ID:            c.ID,
		BaseProfileID: c.BaseProfileID,
		Mutations:     append([]visualsearch.Mutation(nil), c.Mutations...),
	}
}

type Config struct {
	Schema                         string            `json:"schema"`
	RunID                          string            `json:"run_id"`
	BenchmarkID                    string            `json:"benchmark_id,omitempty"`
	OutputDir                      string            `json:"output_dir"`
	MemoryRoot                     string            `json:"memory_root,omitempty"`
	MasterPrompt                   string            `json:"master_prompt,omitempty"`
	OrigamiVersion                 string            `json:"origami_version,omitempty"`
	TlalocVersion                  string            `json:"tlaloc_version,omitempty"`
	TrialsPerModel                 int               `json:"trials_per_model,omitempty"`
	CandidatesPerGeneration       int               `json:"candidates_per_generation,omitempty"`
	MaxGenerations                int               `json:"max_generations,omitempty"`
	MinIncumbentImprovement       float64           `json:"min_incumbent_improvement,omitempty"`
	ContinueExplorationWhenStable bool              `json:"continue_exploration_when_stable,omitempty"`
	DiagnosticRetries             bool              `json:"diagnostic_retries"`
	Conditions                    []string          `json:"conditions,omitempty"`
	OutcomeMetric                 string            `json:"outcome_metric,omitempty"`
	Models                        []ModelConfig     `json:"models"`
	Baseline                      SpecimenConfig    `json:"baseline"`
	Candidates                    []CandidateConfig `json:"candidates,omitempty"`
	AutoCandidates                bool              `json:"auto_candidates,omitempty"`
	CandidateBuilder              []string          `json:"candidate_builder,omitempty"`
	AutoCandidateBaseProfileID    string            `json:"auto_candidate_base_profile_id,omitempty"`
	AutoCandidatesPerGeneration   int               `json:"auto_candidates_per_generation,omitempty"`
}

type ExecutionError struct {
	Generation  int    `json:"generation"`
	SpecimenID  string `json:"specimen_id"`
	CandidateID string `json:"candidate_id,omitempty"`
	ModelID     string `json:"model_id"`
	Condition   string `json:"condition"`
	QuestionID  string `json:"question_id"`
	Diagnostic  bool   `json:"diagnostic"`
	Error       string `json:"error"`
}

type ScoreSummary struct {
	CleanTrials  int     `json:"clean_trials"`
	MeanOverall  float64 `json:"mean_overall_score"`
	MeanNative   float64 `json:"mean_native_score"`
	MeanAssisted float64 `json:"mean_assisted_score"`
}

type SpecimenReport struct {
	SpecimenID      string       `json:"specimen_id"`
	CandidateID     string       `json:"candidate_id,omitempty"`
	PNG             string       `json:"png"`
	SHA256          string       `json:"sha256"`
	Scores          ScoreSummary `json:"scores"`
	CampaignPath    string       `json:"campaign_path"`
	ResultPath      string       `json:"result_path"`
	MemoryEvents    int          `json:"memory_events"`
	MemoryEventIDs  []string     `json:"memory_event_ids,omitempty"`
	ExecutionErrors int          `json:"execution_errors"`
}

type CandidateOutcome struct {
	CandidateID string  `json:"candidate_id"`
	Metric      string  `json:"metric"`
	Before      float64 `json:"before"`
	After       float64 `json:"after"`
	Delta       float64 `json:"delta"`
	NonRegress  bool    `json:"non_regression"`
	Advanceable bool    `json:"advanceable"`
	Reason      string  `json:"reason,omitempty"`
	EventID     string  `json:"event_id,omitempty"`
}

type GenerationReport struct {
	Generation         int                `json:"generation"`
	PlanBeforePath     string             `json:"plan_before_path"`
	QueuePath          string             `json:"queue_path,omitempty"`
	Baseline           SpecimenReport     `json:"baseline"`
	IncumbentBeforeID  string             `json:"incumbent_before_id"`
	IncumbentAfterID   string             `json:"incumbent_after_id"`
	IncumbentAdvanced  bool               `json:"incumbent_advanced"`
	IncumbentReason    string             `json:"incumbent_reason,omitempty"`
	ActiveFailureCount int                `json:"active_failure_count"`
	Candidates         []SpecimenReport   `json:"candidates,omitempty"`
	Outcomes           []CandidateOutcome `json:"outcomes,omitempty"`
	PlanAfterPath      string             `json:"plan_after_path"`
	SelectedIDs        []string           `json:"selected_candidate_ids,omitempty"`
	RemainingBank      int                `json:"remaining_candidate_bank"`
}

type Report struct {
	Schema            string             `json:"schema"`
	RunID             string             `json:"run_id"`
	OutputDir         string             `json:"output_dir"`
	MemoryRoot        string             `json:"memory_root"`
	InitialBaselineID string             `json:"initial_baseline_id"`
	FinalIncumbentID  string             `json:"final_incumbent_id"`
	Generations       []GenerationReport `json:"generations"`
	ExecutionErrors   []ExecutionError   `json:"execution_errors,omitempty"`
	FinalPlanPath     string             `json:"final_plan_path,omitempty"`
	StopReason        string             `json:"stop_reason"`
	Authority         string             `json:"authority"`
}
