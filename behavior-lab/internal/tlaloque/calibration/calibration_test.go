package calibration

import (
	"math"
	"testing"
)

func almost(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMetrics_HandComputed(t *testing.T) {
	almost(t, Accuracy([]Prediction{{0.9, true}, {0.8, true}, {0.2, false}}), 2.0/3.0)

	// (0.9-1)^2 = 0.01 ; (0.8-0)^2 = 0.64 ; mean = 0.325
	almost(t, BrierScore([]Prediction{{0.9, true}, {0.8, false}}), 0.325)

	// 2 bins: {0.4,true} -> bin0 conf 0.4 acc 1.0 gap 0.6 ; {0.6,false} -> bin1 conf 0.6 acc 0 gap 0.6
	almost(t, ExpectedCalibrationError([]Prediction{{0.4, true}, {0.6, false}}, 2), 0.6)

	curve := AbstentionCurve([]Prediction{{0.3, false}, {0.6, true}, {0.9, true}}, []float64{0.8, 0.5})
	if len(curve) != 2 || curve[0].Threshold != 0.5 {
		t.Fatalf("curve not sorted ascending: %+v", curve)
	}
	almost(t, curve[0].Coverage, 2.0/3.0)
	almost(t, curve[0].CoveredAccuracy, 1.0)
	almost(t, curve[1].Coverage, 1.0/3.0)
}

func TestBuildEvalSlice_CoverageCountsAbstentions(t *testing.T) {
	slice := BuildEvalSlice([]Prediction{{0.9, true}, {0.9, true}, {0.9, false}}, 1, 10)
	if slice.N != 4 {
		t.Errorf("N: got %d, want 4", slice.N)
	}
	almost(t, slice.Coverage, 0.75)
	almost(t, slice.Accuracy, 2.0/3.0)
}

func measuredProfile() CalibrationProfile {
	return CalibrationProfile{
		Schema:            Schema,
		ConfidenceFloor:   0.7,
		InDistribution:    EvalSlice{N: 500, Accuracy: 0.99, ECE: 0.03},
		OutOfDistribution: EvalSlice{N: 200, Accuracy: 0.86, ECE: 0.09},
		AbstentionCurve:   []AbstentionPoint{{0.5, 1, 0.8}, {0.7, 0.8, 0.9}, {0.9, 0.5, 0.97}},
	}
}

func TestVerdict(t *testing.T) {
	profile := measuredProfile()

	if v := profile.Verdict(Query{Confidence: 0.5}); v != LowEvidence {
		t.Errorf("below floor: got %s, want LOW_EVIDENCE", v)
	}
	if v := profile.Verdict(Query{Confidence: 0.95}); v != Answered {
		t.Errorf("confident & measured OOD: got %s, want ANSWERED", v)
	}

	noOOD := profile
	noOOD.OutOfDistribution = EvalSlice{}
	if v := noOOD.Verdict(Query{Confidence: 0.95}); v != Unknown {
		t.Errorf("no OOD evidence: got %s, want UNKNOWN", v)
	}

	denied := profile
	denied.UnsupportedDomains = []string{"legal"}
	if v := denied.Verdict(Query{Confidence: 0.95, Domain: "Legal"}); v != Unsupported {
		t.Errorf("deny-listed domain: got %s, want UNSUPPORTED", v)
	}

	allowed := profile
	allowed.SupportedDomains = []string{"swarm"}
	if v := allowed.Verdict(Query{Confidence: 0.95, Domain: "biology"}); v != Unsupported {
		t.Errorf("outside allow-list: got %s, want UNSUPPORTED", v)
	}
	if v := allowed.Verdict(Query{Confidence: 0.95, Domain: "swarm"}); v != Answered {
		t.Errorf("inside allow-list: got %s, want ANSWERED", v)
	}
}

func TestAdmitAsActive(t *testing.T) {
	if admitted, reasons := measuredProfile().AdmitAsActive(); !admitted {
		t.Errorf("expected a measured profile to be admitted, blocked by: %v", reasons)
	}

	// Perfect in-distribution but never measured out-of-distribution — the
	// exact shape of questionclass-charcnn-r0 today. Must be refused.
	untested := CalibrationProfile{
		Schema:          Schema,
		ConfidenceFloor: 0.7,
		InDistribution:  EvalSlice{N: 2000, Accuracy: 1.0, ECE: 0.0},
		AbstentionCurve: []AbstentionPoint{{0.5, 1, 1}, {0.7, 1, 1}, {0.9, 1, 1}},
	}
	admitted, reasons := untested.AdmitAsActive()
	if admitted {
		t.Fatal("a model with no OOD measurement must not be admitted as ACTIVE")
	}
	found := false
	for _, reason := range reasons {
		if reason == "out-of-distribution slice is not measured on enough samples" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the OOD-sample reason, got %v", reasons)
	}
}
