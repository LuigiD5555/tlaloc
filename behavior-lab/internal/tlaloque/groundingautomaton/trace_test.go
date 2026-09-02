package groundingautomaton

import "testing"

func TestReasonCodesPreserveTraceOrder(t *testing.T) {
	trace := ClaimTrace{Reasons: []Reason{{Code: ReasonAligned}, {Code: ReasonNumericContradiction}}}
	codes := ReasonCodes(trace)
	if len(codes) != 2 || codes[0] != ReasonAligned || codes[1] != ReasonNumericContradiction {
		t.Fatalf("unexpected reason codes: %+v", codes)
	}
}
