package calibration

import (
	"math"
	"sort"
)

// ProportionInterval is a two-sided confidence interval for a binomial
// success proportion: the point estimate plus its lower and upper bounds.
type ProportionInterval struct {
	Proportion float64 `json:"proportion"`
	Low        float64 `json:"ci95_low"`
	High       float64 `json:"ci95_high"`
}

// WilsonInterval returns the 95% Wilson score interval for correct
// successes out of total trials. The Wilson interval is preferred over the
// normal approximation for the small samples (n = 30..50) used in the
// Parrot Capability Lab: it stays inside [0, 1] and does not collapse to a
// zero-width interval at proportions of 0 or 1. total <= 0 yields a zeroed
// interval.
func WilsonInterval(correct, total int) ProportionInterval {
	if total <= 0 {
		return ProportionInterval{}
	}
	const z = 1.959963984540054 // 97.5th percentile of the standard normal
	n := float64(total)
	phat := float64(correct) / n
	z2 := z * z
	denominator := 1 + z2/n
	center := (phat + z2/(2*n)) / denominator
	margin := (z / denominator) * math.Sqrt(phat*(1-phat)/n+z2/(4*n*n))
	low := center - margin
	high := center + margin
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	return ProportionInterval{Proportion: phat, Low: low, High: high}
}

// Prediction is one scored, labeled outcome from an evaluation run:
// the model's stated confidence in its top choice, and whether that choice
// was correct. Abstentions are excluded before metric computation — they
// are accounted for separately via Coverage.
type Prediction struct {
	Confidence float64
	Correct    bool
}

// Accuracy is the fraction of predictions that were correct.
func Accuracy(predictions []Prediction) float64 {
	if len(predictions) == 0 {
		return 0
	}
	correct := 0
	for _, prediction := range predictions {
		if prediction.Correct {
			correct++
		}
	}
	return float64(correct) / float64(len(predictions))
}

// BrierScore is the mean squared error between stated confidence and
// correctness (1 for correct, 0 for wrong). Lower is better; 0 is perfect.
func BrierScore(predictions []Prediction) float64 {
	if len(predictions) == 0 {
		return 0
	}
	total := 0.0
	for _, prediction := range predictions {
		outcome := 0.0
		if prediction.Correct {
			outcome = 1.0
		}
		diff := prediction.Confidence - outcome
		total += diff * diff
	}
	return total / float64(len(predictions))
}

// ExpectedCalibrationError bins predictions by confidence and returns the
// weighted average gap between mean confidence and observed accuracy per
// bin. bins must be >= 1; a common choice is 10.
func ExpectedCalibrationError(predictions []Prediction, bins int) float64 {
	if len(predictions) == 0 || bins < 1 {
		return 0
	}
	type bucket struct {
		count      int
		confidence float64
		correct    int
	}
	buckets := make([]bucket, bins)
	for _, prediction := range predictions {
		index := int(prediction.Confidence * float64(bins))
		if index >= bins {
			index = bins - 1
		}
		if index < 0 {
			index = 0
		}
		buckets[index].count++
		buckets[index].confidence += prediction.Confidence
		if prediction.Correct {
			buckets[index].correct++
		}
	}
	total := float64(len(predictions))
	ece := 0.0
	for _, b := range buckets {
		if b.count == 0 {
			continue
		}
		meanConfidence := b.confidence / float64(b.count)
		accuracy := float64(b.correct) / float64(b.count)
		gap := meanConfidence - accuracy
		if gap < 0 {
			gap = -gap
		}
		ece += (float64(b.count) / total) * gap
	}
	return ece
}

// AbstentionCurve reports, for each threshold, how much of the input the
// model would still answer (coverage) and how accurate it would be on that
// covered subset. thresholds are sorted ascending in the output.
func AbstentionCurve(predictions []Prediction, thresholds []float64) []AbstentionPoint {
	sorted := append([]float64(nil), thresholds...)
	sort.Float64s(sorted)
	total := float64(len(predictions))
	curve := make([]AbstentionPoint, 0, len(sorted))
	for _, threshold := range sorted {
		covered := 0
		correct := 0
		for _, prediction := range predictions {
			if prediction.Confidence < threshold {
				continue
			}
			covered++
			if prediction.Correct {
				correct++
			}
		}
		point := AbstentionPoint{Threshold: threshold}
		if total > 0 {
			point.Coverage = float64(covered) / total
		}
		if covered > 0 {
			point.CoveredAccuracy = float64(correct) / float64(covered)
		}
		curve = append(curve, point)
	}
	return curve
}

// BuildEvalSlice computes an EvalSlice from a set of attempted predictions
// plus the count of abstentions on the same slice (so Coverage reflects
// them). eceBins is passed through to ExpectedCalibrationError.
func BuildEvalSlice(attempted []Prediction, abstentions, eceBins int) EvalSlice {
	sliceTotal := len(attempted) + abstentions
	slice := EvalSlice{
		N:        sliceTotal,
		Accuracy: Accuracy(attempted),
		ECE:      ExpectedCalibrationError(attempted, eceBins),
		Brier:    BrierScore(attempted),
	}
	if sliceTotal > 0 {
		slice.Coverage = float64(len(attempted)) / float64(sliceTotal)
	}
	return slice
}
