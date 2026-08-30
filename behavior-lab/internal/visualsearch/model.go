package visualsearch

const SchemaR0 = "tlaloc.origami-visual-search.r0"

type MutationKind string

const (
	MutationPrompt                MutationKind = "PROMPT"
	MutationChannelRole           MutationKind = "CHANNEL_ROLE"
	MutationPrimitive             MutationKind = "PRIMITIVE"
	MutationLayout                MutationKind = "LAYOUT"
	MutationRedundancy            MutationKind = "REDUNDANCY"
	MutationColorUsage            MutationKind = "COLOR_USAGE"
	MutationNumericStructure      MutationKind = "NUMERIC_STRUCTURE"
	MutationInterferenceStructure MutationKind = "INTERFERENCE_STRUCTURE"
	MutationDepthStructure        MutationKind = "DEPTH_STRUCTURE"
	MutationTemporalStructure     MutationKind = "TEMPORAL_STRUCTURE"
	MutationEmergentStructure     MutationKind = "EMERGENT_STRUCTURE"
)

type Mutation struct {
	Kind         MutationKind `json:"kind"`
	Target       string       `json:"target"`
	Value        string       `json:"value"`
	Rationale    string       `json:"rationale,omitempty"`
	Experimental bool         `json:"experimental"`
}

type Candidate struct {
	Schema        string     `json:"schema"`
	ID            string     `json:"id"`
	BaseProfileID string     `json:"base_profile_id"`
	PromptBaseSHA string     `json:"prompt_base_sha256,omitempty"`
	Mutations     []Mutation `json:"mutations"`
}

type Metrics struct {
	SemanticRoundtripRate       float64 `json:"semantic_roundtrip_rate"`
	BootProbePassRate           float64 `json:"boot_probe_pass_rate"`
	NativeIndexRecoveryRate     float64 `json:"native_index_recovery_rate"`
	NativeSemanticAnswerRate    float64 `json:"native_semantic_answer_rate"`
	RoutingAccuracy             float64 `json:"routing_accuracy"`
	VerifiedEvidenceRate        float64 `json:"verified_evidence_rate"`
	TransportPassRate           float64 `json:"transport_pass_rate"`
	PerceptualRevealRate        float64 `json:"perceptual_reveal_rate,omitempty"`
	ContextEfficiency           float64 `json:"context_efficiency"`
	MeanContextTokens           float64 `json:"mean_context_tokens"`
	CarrierBytes                int     `json:"carrier_bytes"`
	RecoverableSemanticUnits    int     `json:"recoverable_semantic_units,omitempty"`
	MeanRecognitionMillis       float64 `json:"mean_recognition_millis,omitempty"`
	MeanBootstrapSteps          float64 `json:"mean_bootstrap_steps,omitempty"`
	MeanDecodeSteps             float64 `json:"mean_decode_steps,omitempty"`
	MechanicalDependencyViolations int `json:"mechanical_dependency_violations"`
	UnverifiedMechanicalClaims  int     `json:"unverified_mechanical_claims"`
	FalseExact                  int     `json:"false_exact"`
	BudgetViolations            int     `json:"budget_violations"`
	UnknownViolations           int     `json:"unknown_violations"`
	RealModels                  int     `json:"real_models"`
	Trials                      int     `json:"trials"`
}

type Evidence struct {
	CandidateID string  `json:"candidate_id"`
	Metrics     Metrics `json:"metrics"`
	CorpusID    string  `json:"corpus_id,omitempty"`
	EvidenceRef string  `json:"evidence_ref,omitempty"`
}

type Policy struct {
	MaxCarrierBytes               int     `json:"max_carrier_bytes"`
	MaxMeanContextTokens          float64 `json:"max_mean_context_tokens"`
	MinSemanticRoundtripRate      float64 `json:"min_semantic_roundtrip_rate"`
	MinNativeIndexRecoveryRate    float64 `json:"min_native_index_recovery_rate"`
	MinNativeSemanticAnswerRate   float64 `json:"min_native_semantic_answer_rate"`
	MinVerifiedEvidenceRate       float64 `json:"min_verified_evidence_rate"`
	MinRoutingAccuracy            float64 `json:"min_routing_accuracy"`
	MinPerceptualRevealRate       float64 `json:"min_perceptual_reveal_rate"`
	MinRealModelsForPerception    int     `json:"min_real_models_for_perception"`
	MinTrials                     int     `json:"min_trials"`
	MinImprovement                float64 `json:"min_improvement"`
}

func DefaultPolicy() Policy {
	return Policy{
		MaxCarrierBytes:             512000,
		MaxMeanContextTokens:        4000,
		MinSemanticRoundtripRate:    1,
		MinNativeIndexRecoveryRate:  .95,
		MinNativeSemanticAnswerRate: .90,
		MinVerifiedEvidenceRate:     .95,
		MinRoutingAccuracy:          .95,
		MinPerceptualRevealRate:     .95,
		MinRealModelsForPerception:  3,
		MinTrials:                   9,
		MinImprovement:              .01,
	}
}

type Gate struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"`
}

type Evaluation struct {
	Schema             string     `json:"schema"`
	CandidateID        string     `json:"candidate_id"`
	BaseProfileID      string     `json:"base_profile_id"`
	Score              float64    `json:"score"`
	BaselineScore      float64    `json:"baseline_score"`
	Improvement        float64    `json:"improvement"`
	Gates              []Gate     `json:"gates"`
	PromotionCandidate bool       `json:"promotion_candidate"`
	Recommendation     string     `json:"recommendation"`
	Metrics            Metrics    `json:"metrics"`
	Mutations          []Mutation `json:"mutations"`
}

type Tournament struct {
	Schema         string       `json:"schema"`
	BaseProfileID  string       `json:"base_profile_id"`
	Baseline       Metrics      `json:"baseline"`
	Evaluations    []Evaluation `json:"evaluations"`
	WinnerID       string       `json:"winner_id,omitempty"`
	Recommendation string       `json:"recommendation"`
	Authority      string       `json:"authority"`
}
