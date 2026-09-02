package groundingautomaton

// EvalCase is a small, serializable ground-truth case used by local and CI
// evaluation commands. Expected is intentionally categorical: R0 evaluates
// authority boundaries, not only a scalar score.
type EvalCase struct {
	ID       string  `json:"id"`
	Question string  `json:"question,omitempty"`
	Answer   string  `json:"answer"`
	Evidence string  `json:"evidence"`
	Expected Verdict `json:"expected"`
	Family   string  `json:"family,omitempty"`
}

type EvalObservation struct {
	ID       string       `json:"id"`
	Expected Verdict      `json:"expected"`
	Actual   Verdict      `json:"actual"`
	Output   VerifyOutput `json:"output"`
	Correct  bool         `json:"correct"`
}

type EvalMetrics struct {
	Total                       int     `json:"total"`
	Correct                     int     `json:"correct"`
	Accuracy                    float64 `json:"accuracy"`
	Covered                     int     `json:"covered"`
	Coverage                    float64 `json:"coverage"`
	Unknown                     int     `json:"unknown"`
	Insufficient                int     `json:"insufficient"`
	FalseSupportedContradiction int     `json:"false_supported_contradiction"`
	ContradictionTruePositive   int     `json:"contradiction_true_positive"`
	ContradictionFalsePositive  int     `json:"contradiction_false_positive"`
	ContradictionFalseNegative  int     `json:"contradiction_false_negative"`
	ContradictionPrecision      float64 `json:"contradiction_precision"`
	ContradictionRecall         float64 `json:"contradiction_recall"`
}

func Evaluate(cases []EvalCase) ([]EvalObservation, EvalMetrics) {
	observations := make([]EvalObservation, 0, len(cases))
	metrics := EvalMetrics{Total: len(cases)}
	for _, c := range cases {
		out := Verify(VerifyInput{Question: c.Question, ModelAnswer: c.Answer, PageContent: c.Evidence})
		obs := EvalObservation{ID: c.ID, Expected: c.Expected, Actual: out.Verdict, Output: out, Correct: out.Verdict == c.Expected}
		observations = append(observations, obs)
		if obs.Correct {
			metrics.Correct++
		}
		if out.Verdict == VerdictSupported || out.Verdict == VerdictContradicted {
			metrics.Covered++
		}
		if out.Verdict == VerdictUnknown {
			metrics.Unknown++
		}
		if out.Verdict == VerdictInsufficient {
			metrics.Insufficient++
		}
		if c.Expected == VerdictContradicted && out.Verdict == VerdictSupported {
			metrics.FalseSupportedContradiction++
		}
		if c.Expected == VerdictContradicted && out.Verdict == VerdictContradicted {
			metrics.ContradictionTruePositive++
		}
		if c.Expected != VerdictContradicted && out.Verdict == VerdictContradicted {
			metrics.ContradictionFalsePositive++
		}
		if c.Expected == VerdictContradicted && out.Verdict != VerdictContradicted {
			metrics.ContradictionFalseNegative++
		}
	}
	if metrics.Total > 0 {
		metrics.Accuracy = float64(metrics.Correct) / float64(metrics.Total)
		metrics.Coverage = float64(metrics.Covered) / float64(metrics.Total)
	}
	precisionDenom := metrics.ContradictionTruePositive + metrics.ContradictionFalsePositive
	if precisionDenom > 0 {
		metrics.ContradictionPrecision = float64(metrics.ContradictionTruePositive) / float64(precisionDenom)
	}
	recallDenom := metrics.ContradictionTruePositive + metrics.ContradictionFalseNegative
	if recallDenom > 0 {
		metrics.ContradictionRecall = float64(metrics.ContradictionTruePositive) / float64(recallDenom)
	}
	return observations, metrics
}
