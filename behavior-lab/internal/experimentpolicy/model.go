package experimentpolicy

const (
	IntentSchemaR1      = "tlaloc.experiment-intent.r1"
	CandidateSchemaR1   = "tlaloc.candidate-manifest.r1"
	SemanticSchemaR1    = "origami.semantic-manifest.r1"
	BuildSchemaR1       = "origami.candidate-build-manifest.r1"
	ParitySchemaR1      = "tlaloc.semantic-parity-report.r1"
	RegressionSchemaR1  = "tlaloc.regression-report.r1"
)

type ExperimentIntent struct {
	Schema              string   `json:"schema"`
	ID                  string   `json:"id"`
	Objective           string   `json:"objective"`
	BaselineCandidateID string   `json:"baseline_candidate_id"`
	FailureFrontier     string   `json:"failure_frontier"`
	MutableModule       string   `json:"mutable_module"`
	Preserve            []string `json:"preserve"`
	Avoid               []string `json:"avoid,omitempty"`
	Require             []string `json:"require"`
	CandidateBudget     int      `json:"candidate_budget"`
	TrialsPerModel      int      `json:"trials_per_model"`
	Models              []string `json:"models,omitempty"`
}

type Mutation struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Value  string `json:"value"`
}

type CandidateManifest struct {
	Schema              string     `json:"schema"`
	ID                  string     `json:"id"`
	ParentID            string     `json:"parent_id"`
	ProgramSHA256       string     `json:"program_sha256"`
	PayloadSHA256       string     `json:"payload_sha256,omitempty"`
	GenomeID            string     `json:"genome_id,omitempty"`
	GenomeVersion       int        `json:"genome_version,omitempty"`
	Mutations           []Mutation `json:"mutations"`
	ChangedModules      []string   `json:"changed_modules"`
	PreservedModules    []string   `json:"preserved_modules"`
	ForbiddenChanges    []string   `json:"forbidden_changes"`
	ExpectedEffect      string     `json:"expected_effect"`
	ParentEvidenceIDs   []string   `json:"parent_evidence_ids,omitempty"`
}

type SemanticFact struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SemanticManifest struct {
	Schema        string         `json:"schema"`
	ProgramSHA256 string         `json:"program_sha256"`
	PayloadSHA256 string         `json:"payload_sha256,omitempty"`
	Facts         []SemanticFact `json:"facts"`
}

type BuildManifest struct {
	Schema           string           `json:"schema"`
	CandidateID      string           `json:"candidate_id"`
	RendererVersion  string           `json:"renderer_version"`
	ArtifactSHA256   string           `json:"artifact_sha256"`
	ArtifactBytes    int              `json:"artifact_bytes"`
	ProgramSHA256    string           `json:"program_sha256"`
	PayloadSHA256    string           `json:"payload_sha256,omitempty"`
	AppliedMutations []Mutation       `json:"applied_mutations"`
	VisibleSemantics SemanticManifest `json:"visible_semantics"`
}

type Difference struct {
	Key      string `json:"key"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Allowed  bool   `json:"allowed"`
}

type ParityReport struct {
	Schema      string       `json:"schema"`
	CandidateID string       `json:"candidate_id"`
	Pass        bool         `json:"pass"`
	Differences []Difference `json:"differences,omitempty"`
	FailureCode string       `json:"failure_code,omitempty"`
}

type RegressionCheck struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"`
}

type RegressionReport struct {
	Schema      string            `json:"schema"`
	CandidateID string            `json:"candidate_id"`
	Pass        bool              `json:"pass"`
	Checks      []RegressionCheck `json:"checks"`
}
