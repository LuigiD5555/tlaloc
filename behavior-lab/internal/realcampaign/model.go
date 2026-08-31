package realcampaign

const (
	SpecSchema     = "tlaloc.real-vlm-campaign.r0.spec"
	ManifestSchema = "tlaloc.real-vlm-campaign.r0.manifest"
	PhaseSmoke     = "SMOKE"
	PhaseEvidence  = "EVIDENCE"
)

type Spec struct {
	Schema             string   `json:"schema"`
	CampaignID         string   `json:"campaign_id"`
	Phase              string   `json:"phase"`
	Endpoint           string   `json:"endpoint"`
	Model              string   `json:"model,omitempty"`
	Compatibility      string   `json:"compatibility_strategy,omitempty"`
	TransportCondition string   `json:"transport_condition,omitempty"`
	InteropMemoryRoot  string   `json:"interop_memory_root,omitempty"`
	APIKeyEnv          string   `json:"api_key_env,omitempty"`
	Program            string   `json:"program"`
	TemporalCarrier    string   `json:"temporal_carrier"`
	CandidateBuilder   string   `json:"candidate_builder"`
	OutputDir          string   `json:"output_dir"`
	MasterPrompt       string   `json:"master_prompt,omitempty"`
	Temperature        float64  `json:"temperature,omitempty"`
	TimeoutSeconds     int      `json:"timeout_seconds,omitempty"`
	TransportRetries   int      `json:"transport_retries,omitempty"`
	TrialsPerModel     int      `json:"trials_per_model,omitempty"`
	CandidatesPerGen   int      `json:"candidates_per_generation,omitempty"`
	MaxGenerations     int      `json:"max_generations,omitempty"`
	Conditions         []string `json:"conditions,omitempty"`
}

type BuilderCapabilities struct {
	Schema             string   `json:"schema"`
	ParentProfiles     []string `json:"parent_profiles"`
	SupportedKinds     []string `json:"supported_kinds"`
	UnsupportedKinds   []string `json:"unsupported_kinds"`
	ExactPlaneMutation bool     `json:"exact_plane_mutation"`
	MaxMutations       int      `json:"max_mutations"`
}

type DoctorResult struct {
	Schema                    string              `json:"schema"`
	Endpoint                  string              `json:"endpoint"`
	CompatibilityStrategy     string              `json:"compatibility_strategy"`
	ModelInterop              ModelInteropProfile `json:"model_interop"`
	WorkingConfigurationPath  string              `json:"working_configuration_path,omitempty"`
	DiscoveredModels          []string            `json:"discovered_models"`
	SelectedModel             string              `json:"selected_model"`
	VisionTransport           bool                `json:"vision_transport"`
	ProbeResponse             string              `json:"probe_response,omitempty"`
	TemporalCarrier           string              `json:"temporal_carrier"`
	TemporalCarrierSHA256     string              `json:"temporal_carrier_sha256"`
	CandidateBuilder          string              `json:"candidate_builder"`
	CandidateBuilderSHA256    string              `json:"candidate_builder_sha256"`
	BuilderCapabilities       BuilderCapabilities `json:"builder_capabilities"`
	ProgramSHA256             string              `json:"program_sha256"`
	ParentProfile             string              `json:"parent_profile"`
	Ready                     bool                `json:"ready"`
}

type Manifest struct {
	Schema                    string              `json:"schema"`
	CampaignID                string              `json:"campaign_id"`
	Phase                     string              `json:"phase"`
	Status                    string              `json:"status"`
	Endpoint                  string              `json:"endpoint"`
	CompatibilityStrategy     string              `json:"compatibility_strategy"`
	ModelID                   string              `json:"model_id"`
	ModelInterop              ModelInteropProfile `json:"model_interop"`
	WorkingConfigurationPath  string              `json:"working_configuration_path,omitempty"`
	TlalocVersion             string              `json:"tlaloc_version"`
	OrigamiExpectedVersion    string              `json:"origami_expected_version"`
	ProgramPath               string              `json:"program_path"`
	ProgramSHA256             string              `json:"program_sha256"`
	BaselinePNG               string              `json:"baseline_png"`
	BaselineSHA256            string              `json:"baseline_sha256"`
	BaselineBytes             int                 `json:"baseline_bytes"`
	TemporalCarrier           string              `json:"temporal_carrier"`
	TemporalCarrierSHA256     string              `json:"temporal_carrier_sha256"`
	CandidateBuilder          string              `json:"candidate_builder"`
	CandidateBuilderSHA256    string              `json:"candidate_builder_sha256"`
	BuilderCapabilities       BuilderCapabilities `json:"builder_capabilities"`
	ClosedLoopConfig          string              `json:"closed_loop_config"`
	ClosedLoopConfigSHA256    string              `json:"closed_loop_config_sha256"`
	MemoryRoot                string              `json:"memory_root"`
	EvidencePolicy            string              `json:"evidence_policy"`
	PromotionEligible         bool                `json:"promotion_eligible"`
	CrossModelEvidence        bool                `json:"cross_model_evidence"`
}

type Prepared struct {
	Spec         Spec         `json:"spec"`
	Doctor       DoctorResult `json:"doctor"`
	Manifest     Manifest     `json:"manifest"`
	ManifestPath string       `json:"manifest_path"`
	ConfigPath   string       `json:"config_path"`
}
