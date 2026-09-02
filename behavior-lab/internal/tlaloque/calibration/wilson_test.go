package calibration

import (
	"math"
	"testing"
)

func TestWilsonIntervalKnownValues(t *testing.T) {
	// Reference values from the Wilson score formula (z = 1.96).
	cases := []struct {
		correct, total    int
		wantLow, wantHigh float64
	}{
		{39, 50, 0.6476, 0.8725},
		{0, 30, 0.0000, 0.1134},
		{30, 30, 0.8866, 1.0000},
		{15, 30, 0.3352, 0.6648},
	}
	for _, c := range cases {
		got := WilsonInterval(c.correct, c.total)
		if math.Abs(got.Low-c.wantLow) > 0.01 || math.Abs(got.High-c.wantHigh) > 0.01 {
			t.Errorf("WilsonInterval(%d,%d) = [%.4f, %.4f], want ~[%.4f, %.4f]",
				c.correct, c.total, got.Low, got.High, c.wantLow, c.wantHigh)
		}
		if got.Low < 0 || got.High > 1 {
			t.Errorf("WilsonInterval(%d,%d) escaped [0,1]: %+v", c.correct, c.total, got)
		}
	}
}

func TestWilsonIntervalEmpty(t *testing.T) {
	if got := WilsonInterval(0, 0); got != (ProportionInterval{}) {
		t.Errorf("WilsonInterval(0,0) = %+v, want zero value", got)
	}
}
