package tonalt1arms

import "context"

// ParrotAdapter is the narrow interface every Parrot-routed DAG node calls
// through. One method keeps a fake trivial to write and a real client (built
// in a later task, never invoked in this one) swappable behind it -- the
// same "small interface, injected client" pattern already established in
// internal/exocortex/parrot_tlaloque_r1.go's perceptionCompleter.
type ParrotAdapter interface {
	Call(ctx context.Context, req ParrotRequest) (ParrotResponse, error)
}

// ParrotRequest is one Parrot adapter call's input. Capability names which
// frozen adapter this is (EXTRACT_NUMBER | NORMALIZE | COMPARE_NUMBERS |
// ARITHMETIC, per T1_D5_ARM_B_POLICY.json's parrot_adapters); Prompt is that
// capability's frozen prompt_template (already filled in by the caller);
// Image is non-nil only for EXTRACT_NUMBER (a visual capability) or Arm A's
// single monolithic call.
type ParrotRequest struct {
	Capability  string
	Prompt      string
	Image       []byte
	Temperature float64
	MaxTokens   int
}

// ParrotResponse is one Parrot adapter call's outcome, broken into the
// separately-tracked statuses the task's record schema (§17) requires:
// TransportOK (did the HTTP round-trip succeed at all), SchemaOK (did the
// response parse as the expected shape), ContractOK (did the parsed value
// satisfy the capability's semantic contract, e.g. a finite number). A
// response can be TransportOK but not SchemaOK/ContractOK.
type ParrotResponse struct {
	RawOutput   string
	ParsedValue float64
	ParsedOK    bool
	TransportOK bool
	SchemaOK    bool
	ContractOK  bool
	FailureCode string
}
