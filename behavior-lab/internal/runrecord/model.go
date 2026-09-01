package runrecord

const Schema = "tlaloc.run-record.v1"

type Record struct {
	Schema       string            `json:"schema"`
	RunID        string            `json:"run_id"`
	EnvHash      string            `json:"env_hash"`
	VariableAxis string            `json:"variable_axis"`
	Component    Component         `json:"component"`
	Model        Model             `json:"model"`
	Sampling     Sampling          `json:"sampling"`
	Prompt       Prompt            `json:"prompt"`
	Fixture      Fixture           `json:"fixture"`
	Host         Host              `json:"host"`
	Outcome      Outcome           `json:"outcome"`
	Repetitions  Repetitions       `json:"repetitions"`
	Replay       string            `json:"replay"`
	Trace        []TransitionEvent `json:"trace"`
}

type Component struct {
	Tlaloc    string `json:"tlaloc"`
	Origami   string `json:"origami"`
	TonalLock string `json:"tonal_lock"`
}

type Model struct {
	Provider      string `json:"provider"`
	IDRequested   string `json:"id_requested"`
	IDReported    string `json:"id_reported"`
	Quantization  string `json:"quantization"`
	ContextWindow int    `json:"context_window"`
	Tokenizer     string `json:"tokenizer"`
	Endpoint      string `json:"endpoint"`
	ObservedAt    string `json:"observed_at"`
}

type Sampling struct {
	Temperature float64  `json:"temperature"`
	TopP        float64  `json:"top_p"`
	Seed        int64    `json:"seed"`
	MaxTokens   int      `json:"max_tokens"`
	Stop        []string `json:"stop"`
}

type Prompt struct {
	BehaviorSpecID     string `json:"behaviorspec_id"`
	PromptIRHash       string `json:"promptir_hash"`
	CompiledPromptHash string `json:"compiled_prompt_hash"`
}

type Fixture struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type Host struct {
	OS     string `json:"os"`
	CPU    string `json:"cpu"`
	GPU    string `json:"gpu"`
	RAMGB  int    `json:"ram_gb"`
	Go     string `json:"go"`
	Python string `json:"python"`
}

type Outcome struct {
	OutputHash string `json:"output_hash"`
	Parsed     bool   `json:"parsed"`
	Verdict    string `json:"verdict"`
	LatencyMS  int64  `json:"latency_ms"`
	TokensIn   int    `json:"tokens_in"`
	TokensOut  int    `json:"tokens_out"`
}

type Repetitions struct {
	N                   int            `json:"n"`
	VerdictDistribution map[string]int `json:"verdict_distribution"`
}

type TransitionEvent struct {
	Sequence  int    `json:"t"`
	From      string `json:"from"`
	To        string `json:"to"`
	At        string `json:"at"`
	LatencyMS int64  `json:"latency_ms"`
	Actor     string `json:"actor"`
}
