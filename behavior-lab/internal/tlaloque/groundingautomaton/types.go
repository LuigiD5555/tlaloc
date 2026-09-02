package groundingautomaton

const (
	Capability = "VERIFY_ANSWER_GROUNDING"
	WorkerID   = "grounding-automaton-r0"

	InputSchema  = "tlaloc.grounding.r0.input"
	OutputSchema = "tlaloc.grounding.r0.output"
)

type Verdict string

const (
	VerdictSupported    Verdict = "SUPPORTED"
	VerdictContradicted Verdict = "CONTRADICTED"
	VerdictInsufficient Verdict = "INSUFFICIENT"
	VerdictUnknown      Verdict = "UNKNOWN"
)

type ReasonCode string

const (
	ReasonAligned                 ReasonCode = "ALIGNED"
	ReasonLowAlignment            ReasonCode = "LOW_ALIGNMENT"
	ReasonPolarityContradiction   ReasonCode = "POLARITY_CONTRADICTION"
	ReasonNumericContradiction    ReasonCode = "NUMERIC_CONTRADICTION"
	ReasonQuantifierContradiction ReasonCode = "QUANTIFIER_CONTRADICTION"
	ReasonAntonymContradiction    ReasonCode = "ANTONYM_CONTRADICTION"
)

type VerifyInput struct {
	Question    string `json:"question,omitempty"`
	ModelAnswer string `json:"model_answer"`
	PageContent string `json:"page_content"`
}

type Reason struct {
	Code     ReasonCode `json:"code"`
	Claim    string     `json:"claim,omitempty"`
	Evidence string     `json:"evidence,omitempty"`
	Detail   string     `json:"detail,omitempty"`
}

type ClaimTrace struct {
	Claim     string   `json:"claim"`
	Evidence  string   `json:"evidence,omitempty"`
	Alignment float64  `json:"alignment"`
	Verdict   Verdict  `json:"verdict"`
	Reasons   []Reason `json:"reasons,omitempty"`
}

type VerifyOutput struct {
	Verdict    Verdict      `json:"verdict"`
	Coverage   float64      `json:"coverage"`
	Confidence float64      `json:"confidence"`
	Claims     []ClaimTrace `json:"claims"`
}
