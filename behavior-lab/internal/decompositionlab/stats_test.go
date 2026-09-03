package decompositionlab

import (
	"math"
	"testing"
)

func almostEqual(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestWilsonCI95_MatchesKnownReferenceValues(t *testing.T) {
	// Reference values cross-checked against the standard Wilson score
	// interval formula (e.g. statsmodels proportion_confint(method="wilson")).
	low, high := WilsonCI95(1, 1)
	if !almostEqual(low, 0.2065, 1e-3) || !almostEqual(high, 1.0, 1e-9) {
		t.Fatalf("Wilson(1,1) = [%v, %v], want ~[0.2065, 1.0]", low, high)
	}
	low, high = WilsonCI95(15, 30)
	if !almostEqual(low, 0.3309, 1e-3) || !almostEqual(high, 0.6691, 1e-3) {
		t.Fatalf("Wilson(15,30) = [%v, %v], want ~[0.3309, 0.6691]", low, high)
	}
	low, high = WilsonCI95(0, 30)
	if low != 0 || high >= 0.2 {
		t.Fatalf("Wilson(0,30) = [%v, %v], want low=0 and a small upper bound", low, high)
	}
}

func TestWilsonCI95_ZeroN(t *testing.T) {
	low, high := WilsonCI95(0, 0)
	if low != 0 || high != 0 {
		t.Fatalf("Wilson(0,0) = [%v, %v], want [0, 0]", low, high)
	}
}

func TestMcNemarExact_AllConcordantHasPValueOne(t *testing.T) {
	pairs := []PairedOutcome{
		{CorrectBefore: true, CorrectAfter: true},
		{CorrectBefore: false, CorrectAfter: false},
	}
	r := McNemarExact(pairs)
	if r.Discordant != 0 {
		t.Fatalf("discordant = %d, want 0", r.Discordant)
	}
	if r.PValue != 1.0 {
		t.Fatalf("p = %v, want 1.0 when there is no discordance", r.PValue)
	}
}

func TestMcNemarExact_KnownCase(t *testing.T) {
	// 10 discordant pairs split 9-1: exact two-sided binomial sign test
	// p-value for (9,1) out of 10 at p=0.5 is 2*P(X<=1) = 2*(1+10)/1024
	// = 22/1024 ≈ 0.02148.
	var pairs []PairedOutcome
	for i := 0; i < 9; i++ {
		pairs = append(pairs, PairedOutcome{CorrectBefore: false, CorrectAfter: true})
	}
	pairs = append(pairs, PairedOutcome{CorrectBefore: true, CorrectAfter: false})
	for i := 0; i < 20; i++ {
		pairs = append(pairs, PairedOutcome{CorrectBefore: true, CorrectAfter: true})
	}
	r := McNemarExact(pairs)
	if r.Discordant != 10 || r.WrongToCorrect != 9 || r.CorrectToWrong != 1 {
		t.Fatalf("unexpected cell counts: %+v", r)
	}
	want := 22.0 / 1024.0
	if !almostEqual(r.PValue, want, 1e-4) {
		t.Fatalf("p = %v, want %v", r.PValue, want)
	}
	if !almostEqual(r.AbsoluteDelta, 8.0/30.0, 1e-9) {
		t.Fatalf("absolute delta = %v, want %v", r.AbsoluteDelta, 8.0/30.0)
	}
}

func TestMcNemarExact_EmptyInput(t *testing.T) {
	r := McNemarExact(nil)
	if r.PValue != 1.0 || r.Discordant != 0 {
		t.Fatalf("expected a neutral result on empty input, got %+v", r)
	}
}

func TestNewVisualExposure_ComputesRatioAndReduction(t *testing.T) {
	exposure := NewVisualExposure(1000, 100000)
	if !almostEqual(exposure.Ratio, 0.01, 1e-9) {
		t.Fatalf("ratio = %v, want 0.01", exposure.Ratio)
	}
	if !almostEqual(exposure.ReductionFactor, 100, 1e-9) {
		t.Fatalf("reduction factor = %v, want 100", exposure.ReductionFactor)
	}
}

func TestNewVisualExposure_GuardsZero(t *testing.T) {
	if got := NewVisualExposure(0, 0); got.Ratio != 0 || got.ReductionFactor != 0 {
		t.Fatalf("expected zero-value exposure for zero inputs, got %+v", got)
	}
}

func TestPercentile_MedianAndP95(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	if got := Median(values); !almostEqual(got, 55, 1e-9) {
		t.Fatalf("median = %v, want 55", got)
	}
	if got := Percentile95(values); got < 90 || got > 100 {
		t.Fatalf("p95 = %v, want within [90, 100]", got)
	}
	// Percentile must not mutate the caller's slice.
	if values[0] != 10 {
		t.Fatalf("Percentile mutated caller's slice: %v", values)
	}
}

func TestAccuracy(t *testing.T) {
	if got := Accuracy(15, 30); got != 0.5 {
		t.Fatalf("Accuracy(15,30) = %v, want 0.5", got)
	}
	if got := Accuracy(0, 0); got != 0 {
		t.Fatalf("Accuracy(0,0) = %v, want 0", got)
	}
}
