// Package tlaloquekit is Tlaloc's public publication surface: the stable,
// dependency-free contract through which a consuming system (Tonal's
// runtime) obtains the small set of qualified Tlaloques Tlaloc has built,
// packaged and qualified, resolves a capability goal into a DAG, and
// executes one bounded capability at a time.
//
// This package deliberately exposes only plain Go types and JSON. It never
// requires a caller to import tlaloc.local/behaviorlab/internal/*. The
// internal Registry, CapabilityRouter, exocortex Tlaloques and Blackboard
// remain implementation details behind BuildQualifiedRegistry.
//
// What this package is NOT: it is not a workflow engine. It holds no
// Blackboard, runs no scheduler, and keeps no execution state. Goal intake,
// the DAG walk, Blackboard ownership, routing decisions across a whole
// workflow, verification coordination and accounting all belong to the
// consuming runtime.
package tlaloquekit

import (
	"context"
	"encoding/json"
)

// SchemaVersion identifies this contract revision.
const SchemaVersion = "tlaloc.tlaloquekit.r1"

// EngineKind is the public implementation-kind of a Tlaloque, ordered from
// the executor a router should prefer to the one it should avoid.
type EngineKind string

const (
	EngineDeterministic EngineKind = "DETERMINISTIC" // exact Go computation
	EngineAlgorithmic   EngineKind = "ALGORITHMIC"   // explicit parser / FSM / search
	EngineSpecialist    EngineKind = "SPECIALIST"    // classic / tiny trained model
	EngineGenerative    EngineKind = "GENERATIVE"    // SLM / LLM (e.g. Parrot)
)

// Descriptor is the stable public Tlaloque contract.
type Descriptor struct {
	ID             string     `json:"id"`
	Capability     string     `json:"capability"`
	Engine         EngineKind `json:"engine"`
	Deterministic  bool       `json:"deterministic"`
	ParameterCount int64      `json:"parameter_count,omitempty"`
	InputSchema    string     `json:"input_schema"`
	OutputSchema   string     `json:"output_schema"`
	Dependencies   []string   `json:"dependencies,omitempty"`
	// ProfileRef and EvidenceRef point at the qualification evidence Tlaloc
	// produced for this Tlaloque. Empty for a Tlaloque that needs no
	// empirical envelope (a pure deterministic computation).
	ProfileRef  string `json:"profile_ref,omitempty"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

// BBox is a rectangle in a page's own coordinate space.
type BBox struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

// LocatedRegion is the documented output shape of the LOCATE_REGION
// capability: located document evidence, described as evidence — not as
// any executor's preprocessing recipe. A downstream Tlaloque (e.g. the
// R1-aware Parrot Tlaloque) uses it to prepare its own working set; TONAL
// only passes it along.
type LocatedRegion struct {
	DocumentID    string            `json:"document_id,omitempty"`
	Page          int               `json:"page"`
	PageImagePath string            `json:"page_image_path,omitempty"`
	SourceBBox    *BBox             `json:"source_bbox,omitempty"`
	LineBBox      *BBox             `json:"line_bbox,omitempty"`
	PageWidth     float64           `json:"page_width,omitempty"`
	PageHeight    float64           `json:"page_height,omitempty"`
	LayoutPath    string            `json:"layout_path,omitempty"`
	Provenance    map[string]string `json:"provenance,omitempty"`
}

// ParrotEndpoint is the OpenAI-compatible transport for a generative
// Tlaloque. It is supplied by the consuming runtime's wiring, never
// invented at execution time.
type ParrotEndpoint struct {
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// Goal asks for a capability, never for a particular executor.
type Goal struct {
	Capability          string   `json:"capability"`
	PreferDeterministic bool     `json:"prefer_deterministic,omitempty"`
	MaxParameters       int64    `json:"max_parameters,omitempty"`
	AvailableProducts   []string `json:"available_products,omitempty"`
}

// Candidate is one executor that could serve a capability, plus why it was
// or was not selected. It feeds the consuming runtime's routing trace.
type Candidate struct {
	Descriptor Descriptor `json:"descriptor"`
	Selected   bool       `json:"selected"`
	// Reason explains the outcome: the selection rationale when Selected,
	// or the rejection rationale otherwise (ranked below the winner, or
	// vetoed by its CapabilityProfile).
	Reason string `json:"reason"`
}

// PlanNode is one node of the resolved DAG. The worker is pinned so the
// consuming runtime's execution is reproducible.
type PlanNode struct {
	ID         string   `json:"id"`
	Capability string   `json:"capability"`
	WorkerID   string   `json:"worker_id"`
	DependsOn  []string `json:"depends_on,omitempty"`
}

// Resolution is the output of Resolve: a pinned DAG plus the candidate
// analysis for every capability it touched.
type Resolution struct {
	Goal       Goal                   `json:"goal"`
	PlanID     string                 `json:"plan_id"`
	Nodes      []PlanNode             `json:"nodes"`
	Selected   []Descriptor           `json:"selected"`
	Candidates map[string][]Candidate `json:"candidates"`
}

// Observation is a typed result written by a Tlaloque. A GENERATIVE
// Tlaloque's Observation is never automatically a fact — Status carries
// what the producer could honestly claim, and the consuming runtime's
// verification decides promotion.
type Observation struct {
	Producer       string            `json:"producer"`
	Capability     string            `json:"capability"`
	Key            string            `json:"key"`
	Value          json.RawMessage   `json:"value"`
	Kind           string            `json:"kind"`             // OBSERVATION | FACT
	Status         string            `json:"status,omitempty"` // "" | VERIFIED | UNSUPPORTED | UNKNOWN
	Confidence     float64           `json:"confidence,omitempty"`
	References     []string          `json:"references,omitempty"`
	Provenance     map[string]string `json:"provenance,omitempty"`
	ProfileVersion string            `json:"profile_version,omitempty"`
	RecordedAt     string            `json:"recorded_at,omitempty"`
}

// ExecutionRequest asks the kit to run exactly one capability on one node.
// The consuming runtime supplies the pinned WorkerID (from a Resolution),
// the node input it built, and the prior Blackboard observations any
// stateful Tlaloque (e.g. Verify) needs to see.
type ExecutionRequest struct {
	TaskID            string          `json:"task_id"`
	NodeID            string          `json:"node_id"`
	Capability        string          `json:"capability"`
	WorkerID          string          `json:"worker_id"`
	Input             json.RawMessage `json:"input"`
	PriorObservations []Observation   `json:"prior_observations,omitempty"`
}

// Usage is a Tlaloque's optional cost self-report. A deterministic
// Tlaloque leaves it nil.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	ModelCalls       int `json:"model_calls,omitempty"`
}

// ExecutionResult is one node's typed output.
type ExecutionResult struct {
	WorkerID     string          `json:"worker_id"`
	Output       json.RawMessage `json:"output"`
	Confidence   float64         `json:"confidence,omitempty"`
	Notes        string          `json:"notes,omitempty"`
	Observations []Observation   `json:"observations,omitempty"`
	Usage        *Usage          `json:"usage,omitempty"`
}

// QualifiedRegistry is the consuming runtime's whole view of Tlaloc. It
// answers "what can you do", "who would you pick", "resolve this goal to a
// DAG" and "run this one node" — nothing about workflows.
type QualifiedRegistry interface {
	// Capabilities lists every qualified Tlaloque, sorted by ID.
	Capabilities() []Descriptor

	// Candidates ranks the executors that could serve a capability under a
	// goal, marking the one Resolve would pin and why the rest would not
	// be. It performs no execution.
	Candidates(capability string, goal Goal) []Candidate

	// Resolve turns a capability goal into a pinned DAG, following the
	// Tlaloques' declared dependencies, with candidate analysis for the
	// routing trace.
	Resolve(goal Goal, planID string, maxParallel int) (Resolution, error)

	// Execute runs one pinned capability on one node and returns its typed
	// output and observations. It maintains no state between calls.
	Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error)

	// ParrotProfileID / ParrotProfileHash identify the frozen Parrot
	// CapabilityProfile in force, or "" when no generative Tlaloque is
	// registered.
	ParrotProfileID() string
	ParrotProfileHash() string
}
