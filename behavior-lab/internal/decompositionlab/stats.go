// Package decompositionlab implements the T0 experiment
// (ONE_OP_DECOMPOSITION_R0 / exocortex-decomposition-r0): it consumes the
// frozen P0 dataset and P2-A CapabilityProfile as read-only evidence, and
// drives the internal/exocortex vertical slice through the four T0-A
// (oracle) and three T0-B (real) conditions. It never fabricates a record:
// every RawRecord in results/EXOCORTEX_DECOMPOSITION_R0.json is either
// produced by an actual run against a real model endpoint, or the file
// does not exist yet.
package decompositionlab

import (
	"math"
	"sort"
)

// WilsonInterval computes the two-sided Wilson score confidence interval
// for a binomial proportion successes/n at the given z (1.96 for 95%).
// This is the same conservative interval family P1/P2-A used to derive
// max_safe rungs; T0 reuses the formula rather than reimplementing ad hoc
// confidence bounds.
func WilsonInterval(successes, n int, z float64) (low, high float64) {
	if n <= 0 {
		return 0, 0
	}
	p := float64(successes) / float64(n)
	nf := float64(n)
	denom := 1 + z*z/nf
	center := p + z*z/(2*nf)
	margin := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf))
	low = (center - margin) / denom
	high = (center + margin) / denom
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	return low, high
}

// WilsonCI95 is WilsonInterval at the standard z=1.96 (95%) used
// throughout T0's reporting.
func WilsonCI95(successes, n int) (low, high float64) {
	return WilsonInterval(successes, n, 1.96)
}

// PairedOutcome is one paired case's correctness under two conditions
// (e.g. C0 -> C1), used by McNemarExact.
type PairedOutcome struct {
	CorrectBefore bool
	CorrectAfter  bool
}

// McNemarResult reports the four paired cells and the exact two-sided
// McNemar p-value (binomial test on the discordant pairs), the standard
// test for a within-subject before/after accuracy comparison.
type McNemarResult struct {
	CorrectToCorrect int
	CorrectToWrong   int
	WrongToCorrect   int
	WrongToWrong     int
	Discordant       int
	PValue           float64
	AbsoluteDelta    float64
}

// McNemarExact computes the exact (binomial, not chi-squared)
// two-sided McNemar test over paired before/after outcomes. It is exact
// because T0's n is small (30 paired cases): a chi-squared approximation
// would be unreliable at that sample size.
func McNemarExact(pairs []PairedOutcome) McNemarResult {
	var r McNemarResult
	for _, p := range pairs {
		switch {
		case p.CorrectBefore && p.CorrectAfter:
			r.CorrectToCorrect++
		case p.CorrectBefore && !p.CorrectAfter:
			r.CorrectToWrong++
		case !p.CorrectBefore && p.CorrectAfter:
			r.WrongToCorrect++
		default:
			r.WrongToWrong++
		}
	}
	r.Discordant = r.CorrectToWrong + r.WrongToCorrect
	n := len(pairs)
	if n > 0 {
		before := r.CorrectToCorrect + r.CorrectToWrong
		after := r.CorrectToCorrect + r.WrongToCorrect
		r.AbsoluteDelta = float64(after-before) / float64(n)
	}
	r.PValue = exactBinomialTwoSided(r.WrongToCorrect, r.Discordant)
	return r
}

// exactBinomialTwoSided is the exact two-sided binomial sign test at p=0.5
// over `discordant` trials with `successes` in one direction: the standard
// formulation of an exact McNemar test.
func exactBinomialTwoSided(successes, discordant int) float64 {
	if discordant == 0 {
		return 1.0
	}
	pointProb := func(k int) float64 {
		return binomialCoefficient(discordant, k) * math.Pow(0.5, float64(discordant))
	}
	observed := pointProb(successes)
	total := 0.0
	for k := 0; k <= discordant; k++ {
		p := pointProb(k)
		if p <= observed*(1+1e-9) {
			total += p
		}
	}
	if total > 1 {
		total = 1
	}
	return total
}

func binomialCoefficient(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	result := 1.0
	for i := 0; i < k; i++ {
		result *= float64(n-i) / float64(i+1)
	}
	return result
}

// Accuracy is a simple successes/n ratio guarded against n==0.
func Accuracy(successes, n int) float64 {
	if n <= 0 {
		return 0
	}
	return float64(successes) / float64(n)
}

// VisualExposure reports the crop/full-page pixel ratio and its inverse,
// the reduction factor (section 19).
type VisualExposure struct {
	Ratio           float64
	ReductionFactor float64
}

func NewVisualExposure(cropPixels, fullPagePixels float64) VisualExposure {
	if fullPagePixels <= 0 || cropPixels <= 0 {
		return VisualExposure{}
	}
	ratio := cropPixels / fullPagePixels
	return VisualExposure{Ratio: ratio, ReductionFactor: 1 / ratio}
}

// Median and Percentile95 are the two descriptive statistics section 19
// asks for beyond the mean; both operate on a copy so the caller's slice
// is never reordered.
func Median(values []float64) float64 {
	return Percentile(values, 0.5)
}

func Percentile95(values []float64) float64 {
	return Percentile(values, 0.95)
}

// Percentile uses nearest-rank interpolation over a sorted copy.
func Percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
