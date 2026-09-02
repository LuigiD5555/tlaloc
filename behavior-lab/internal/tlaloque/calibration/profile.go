// Package calibration is the universal competence-evidence layer for
// learned Tlaloque (small trained classifiers / decoders). A softmax score
// is PREDICTION_CONFIDENCE; it is not EVIDENCE_OF_COMPETENCE. This package
// carries the second thing: a versioned CalibrationProfile measured on
// held-in and, crucially, held-*out* data, plus an abstention policy that
// lets a specialist answer UNKNOWN / UNSUPPORTED / LOW_EVIDENCE even when
// its own softmax says 0.98, and an admission gate that refuses to let a
// model become ACTIVE on in-distribution accuracy alone.
//
// The package is pure and deterministic: metric computation, a policy
// function, and a gate. It does not train, serve, or call models.
package calibration

const Schema = "tlaloc.calibration-profile.r0"

// AbstentionVerdict is what a calibrated specialist is allowed to return
// in place of a prediction.
type AbstentionVerdict string

const (
	// Answered: the prediction may be used as-is.
	Answered AbstentionVerdict = "ANSWERED"
	// Unknown: no out-of-distribution evidence exists for this specialist,
	// so its confidence on unfamiliar input cannot be trusted.
	Unknown AbstentionVerdict = "UNKNOWN"
	// Unsupported: the query's declared domain is outside what the
	// specialist was measured on.
	Unsupported AbstentionVerdict = "UNSUPPORTED"
	// LowEvidence: confidence is below the profile's measured useful floor.
	LowEvidence AbstentionVerdict = "LOW_EVIDENCE"
)

// EvalSlice is the measured behavior of a specialist on one data slice
// (in-distribution or out-of-distribution).
type EvalSlice struct {
	// N is the number of labeled examples in the slice. N == 0 means "this
	// slice was never measured" — a load-bearing distinction.
	N int `json:"n"`
	// Accuracy is the fraction of examples the specialist got right,
	// counting only examples where it did not abstain.
	Accuracy float64 `json:"accuracy"`
	// Coverage is the fraction of examples the specialist attempted (did
	// not abstain on). 1.0 when the model always answers.
	Coverage float64 `json:"coverage"`
	// ECE is the expected calibration error (lower is better; 0 is perfect
	// alignment between stated confidence and observed accuracy).
	ECE float64 `json:"ece"`
	// Brier is the mean squared error between confidence and correctness.
	Brier float64 `json:"brier"`
}

// AbstentionPoint is one row of the abstention curve: if the specialist
// only answers when confidence >= Threshold, it covers Coverage of inputs
// and is CoveredAccuracy accurate on the ones it does answer.
type AbstentionPoint struct {
	Threshold       float64 `json:"threshold"`
	Coverage        float64 `json:"coverage"`
	CoveredAccuracy float64 `json:"covered_accuracy"`
}

// CalibrationProfile is the competence evidence that must travel with a
// learned Tlaloque. It is produced by measuring the model, never asserted.
type CalibrationProfile struct {
	Schema       string `json:"schema"`
	WorkerID     string `json:"worker_id"`
	ModelVersion string `json:"model_version"`

	// TrainingDistributionID and CalibrationSetID identify exactly which
	// data produced the model and which data produced this profile, so a
	// profile can be invalidated when either changes.
	TrainingDistributionID string `json:"training_distribution_id"`
	CalibrationSetID       string `json:"calibration_set_id"`

	InDistribution    EvalSlice `json:"in_distribution"`
	OutOfDistribution EvalSlice `json:"out_of_distribution"`

	AbstentionCurve []AbstentionPoint `json:"abstention_curve"`

	// SupportedDomains, when non-empty, is an allow-list: a query whose
	// domain is not listed is UNSUPPORTED. UnsupportedDomains is an
	// explicit deny-list checked first.
	SupportedDomains   []string `json:"supported_domains,omitempty"`
	UnsupportedDomains []string `json:"unsupported_domains,omitempty"`

	// ConfidenceFloor is the confidence below which a prediction is
	// LOW_EVIDENCE. Chosen from the abstention curve (the lowest threshold
	// whose covered accuracy still clears a useful bar).
	ConfidenceFloor float64 `json:"confidence_floor"`

	GeneratedAt string `json:"generated_at"`
}
