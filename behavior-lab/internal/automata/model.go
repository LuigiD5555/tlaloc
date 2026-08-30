package automata

const TraceSchema = "tlaloc.tlaloque-action-trace.r0"
const AutomatonSchema = "origami.automaton.r0"

type Predicate struct {
	CellID string `json:"cell_id"`
	State  string `json:"state"`
}

type TraceStep struct {
	Step      int         `json:"step"`
	Tlaloque  string      `json:"tlaloque"`
	FromState string      `json:"from_state"`
	ToState   string      `json:"to_state"`
	Requires  []Predicate `json:"requires,omitempty"`
	EmitsTo   []string    `json:"emits_to,omitempty"`
}

type ActionTrace struct {
	Schema string      `json:"schema"`
	ID     string      `json:"id"`
	Steps  []TraceStep `json:"steps"`
}

type Cell struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind,omitempty"`
	InitialState string   `json:"initial_state"`
	Neighbors    []string `json:"neighbors,omitempty"`
}

type Rule struct {
	ID         string      `json:"id"`
	TargetCell string      `json:"target_cell"`
	FromState  string      `json:"from_state,omitempty"`
	ToState    string      `json:"to_state"`
	Requires   []Predicate `json:"requires,omitempty"`
	Priority   int         `json:"priority,omitempty"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

type AutomatonIR struct {
	Schema            string `json:"schema"`
	ID                string `json:"id"`
	Cells             []Cell `json:"cells"`
	Rules             []Rule `json:"rules"`
	Edges             []Edge `json:"edges,omitempty"`
	SourceTraceSHA256 string `json:"source_trace_sha256"`
}

type Metrics struct {
	TraceSteps                 int     `json:"trace_steps"`
	UniqueCells                int     `json:"unique_cells"`
	UniqueRules                int     `json:"unique_rules"`
	RepeatedTransitionsRemoved int     `json:"repeated_transitions_removed"`
	DistillationRatio          float64 `json:"distillation_ratio"`
}

type Result struct {
	Schema    string      `json:"schema"`
	Automaton AutomatonIR `json:"automaton"`
	Metrics   Metrics     `json:"metrics"`
}
