package groundingautomaton

import (
	"encoding/json"
	"testing"
)

func TestVerifyOutputRoundTripsCategoricalVerdict(t *testing.T) {
	original := Verify(VerifyInput{ModelAnswer: "The model has 28 million parameters.", PageContent: "The model has 27 million parameters."})
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded VerifyOutput
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Verdict != VerdictContradicted || len(decoded.Claims) != len(original.Claims) {
		t.Fatalf("contract roundtrip drift: original=%+v decoded=%+v", original, decoded)
	}
}
