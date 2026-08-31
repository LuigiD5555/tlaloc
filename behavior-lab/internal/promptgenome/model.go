package promptgenome

const (
	GenomeSchemaR1   = "tlaloc.prompt-genome.r1"
	CompiledSchemaR1 = "tlaloc.compiled-master-prompt.r1"
)

type Module struct {
	ID           string   `json:"id"`
	Version      int      `json:"version"`
	Purpose      string   `json:"purpose"`
	Text         string   `json:"text"`
	Priority     int      `json:"priority"`
	MinText      string   `json:"min_text,omitempty"`
	Required     bool     `json:"required"`
	Protected    bool     `json:"protected"`
	Maturity     string   `json:"maturity,omitempty"`
	EvidenceIDs  []string `json:"evidence_ids,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	ModelScopes  []string `json:"model_scopes,omitempty"`
}

type Genome struct {
	Schema  string   `json:"schema"`
	ID      string   `json:"id"`
	Version int      `json:"version"`
	Modules []Module `json:"modules"`
}

type CompileRequest struct {
	Genome          Genome   `json:"genome"`
	Model           string   `json:"model,omitempty"`
	RelevantModules []string `json:"relevant_modules,omitempty"`
	MaxChars        int      `json:"max_chars,omitempty"`
}

type CompiledModule struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Text    string `json:"text"`
	Mode    string `json:"mode"`
}

type Compiled struct {
	Schema        string           `json:"schema"`
	GenomeID      string           `json:"genome_id"`
	GenomeVersion int              `json:"genome_version"`
	Model         string           `json:"model,omitempty"`
	Modules       []CompiledModule `json:"modules"`
	Prompt        string           `json:"prompt"`
	Chars         int              `json:"chars"`
	Omitted       []string         `json:"omitted,omitempty"`
	Authority     string           `json:"authority"`
}
