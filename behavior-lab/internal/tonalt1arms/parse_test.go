package tonalt1arms

import (
	"encoding/json"
	"math"
	"sync"
	"testing"
)

func TestParseArmAResponse_BareNumber(t *testing.T) {
	tests := []struct {
		input  string
		expect float64
		ok     bool
	}{
		{"95", 95, true},
		{"-360", -360, true},
		{"1.9098732840549102", 1.9098732840549102, true},
		{" 95 ", 95, true}, // Whitespace is trimmed
		{"95.0", 95.0, true},
		{"-493.5294117647059", -493.5294117647059, true},
		{"", 0, false},       // Empty
		{"abc", 0, false},    // Non-numeric
		{"95.5.5", 0, false}, // Malformed decimal
		{"95 96", 0, false},  // Multiple numbers
	}

	for _, tt := range tests {
		value, ok, _ := ParseArmAResponse(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseArmAResponse(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && math.Abs(value-tt.expect) > 1e-10 {
			t.Errorf("ParseArmAResponse(%q) = %v, want %v", tt.input, value, tt.expect)
		}
	}
}

func TestParseArmAResponse_JSON(t *testing.T) {
	tests := []struct {
		input  string
		expect float64
		ok     bool
	}{
		{`{"value": 95}`, 95, true},
		{`{"result": -360}`, -360, true},
		{`{"value": 95.5}`, 95.5, true},
		// Multiple fields that numerically AGREE are not ambiguous: there is
		// exactly one distinct value, so it's accepted.
		{`{"a": 1, "b": 1}`, 1, true},
		{`{"value": 42, "confidence": 42}`, 42, true},
	}

	for _, tt := range tests {
		value, ok, failCode := ParseArmAResponse(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseArmAResponse(%q) ok = %v (failCode=%q), want %v", tt.input, ok, failCode, tt.ok)
		}
		if ok && math.Abs(value-tt.expect) > 1e-10 {
			t.Errorf("ParseArmAResponse(%q) = %v, want %v", tt.input, value, tt.expect)
		}
	}
}

// TestParseArmAResponse_Ambiguous covers correction B: a JSON object with
// two or more DISTINCT numeric values must fail closed with
// AMBIGUOUS_MODEL_OUTPUT rather than picking one arbitrarily (the old
// "first field wins" behavior, which depended on Go's randomized map
// iteration order, is no longer acceptable).
func TestParseArmAResponse_Ambiguous(t *testing.T) {
	tests := []string{
		`{"a": 1, "b": 2}`,
		`{"value": 95, "result": 96}`,
		`{"x": 1, "y": 2, "z": 3}`,
	}
	for _, input := range tests {
		value, ok, failCode := ParseArmAResponse(input)
		if ok {
			t.Errorf("ParseArmAResponse(%q) = (%v, ok=true), want fail closed with AMBIGUOUS_MODEL_OUTPUT", input, value)
		}
		if failCode != "AMBIGUOUS_MODEL_OUTPUT" {
			t.Errorf("ParseArmAResponse(%q) failCode = %q, want AMBIGUOUS_MODEL_OUTPUT", input, failCode)
		}
	}
}

func TestParseArmAResponse_Failures(t *testing.T) {
	tests := []struct {
		input    string
		wantFail bool
	}{
		{"", true},
		{"abc", true},
		{`{"no_numeric": "here"}`, true},
		{"NaN", true},
		{"Infinity", true},
	}

	for _, tt := range tests {
		_, ok, failCode := ParseArmAResponse(tt.input)
		if ok != !tt.wantFail {
			t.Errorf("ParseArmAResponse(%q) ok = %v, want fail=%v", tt.input, ok, tt.wantFail)
		}
		if !ok && failCode == "" {
			t.Errorf("ParseArmAResponse(%q) returned empty failCode on failure", tt.input)
		}
	}
}

// TestParseArmAResponse_DeterminismStress proves that repeated parsing of
// the same multi-key JSON produces byte-identical results across 1000 runs,
// and that map iteration order cannot leak into the outcome even under
// concurrent goroutines (required by the task's parser-determinism gate).
func TestParseArmAResponse_DeterminismStress(t *testing.T) {
	// Multi-key JSON where all numeric fields agree -> must always resolve
	// to the same accepted value.
	agreeing := `{"value": 42, "confidence": 42, "result": 42, "answer": 42}`
	firstValue, firstOK, firstCode := ParseArmAResponse(agreeing)
	for i := 0; i < 1000; i++ {
		value, ok, code := ParseArmAResponse(agreeing)
		if ok != firstOK || code != firstCode || (ok && value != firstValue) {
			t.Fatalf("run %d: ParseArmAResponse(%q) = (%v,%v,%q), want (%v,%v,%q)",
				i, agreeing, value, ok, code, firstValue, firstOK, firstCode)
		}
	}

	// Multi-key JSON with genuinely distinct values -> must always fail
	// closed the same way.
	ambiguous := `{"value": 42, "confidence": 43, "result": 44}`
	for i := 0; i < 1000; i++ {
		_, ok, code := ParseArmAResponse(ambiguous)
		if ok || code != "AMBIGUOUS_MODEL_OUTPUT" {
			t.Fatalf("run %d: ParseArmAResponse(%q) = (ok=%v, code=%q), want (ok=false, code=AMBIGUOUS_MODEL_OUTPUT)", i, ambiguous, ok, code)
		}
	}

	// Concurrent stress: many goroutines parsing the same inputs in a tight
	// loop must never observe a divergent result, which would indicate map
	// iteration order (or any other nondeterminism) leaking through.
	const goroutines = 16
	const iterations = 200
	var wg sync.WaitGroup
	errs := make(chan string, goroutines*iterations)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				value, ok, code := ParseArmAResponse(agreeing)
				if ok != firstOK || code != firstCode || (ok && value != firstValue) {
					errs <- "agreeing case diverged"
				}
				_, ok2, code2 := ParseArmAResponse(ambiguous)
				if ok2 || code2 != "AMBIGUOUS_MODEL_OUTPUT" {
					errs <- "ambiguous case diverged"
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for msg := range errs {
		t.Fatal(msg)
	}
}

// TestParseArmAResponse_MalformedFallsThroughToBareNumberRegex confirms
// malformed/non-object JSON still falls through to the bare-number regex
// fallback per the existing frozen policy, unaffected by the candidate-set
// rewrite.
func TestParseArmAResponse_MalformedFallsThroughToBareNumberRegex(t *testing.T) {
	// Not a JSON object (a bare JSON number) — json.Unmarshal into
	// map[string]interface{} fails, so this must fall through to strategy 2.
	value, ok, failCode := ParseArmAResponse("95")
	if !ok || value != 95 {
		t.Fatalf("ParseArmAResponse(\"95\") = (%v, %v, %q), want (95, true, \"\")", value, ok, failCode)
	}

	// A malformed JSON object (unterminated) must fail Strategy 1's
	// json.Unmarshal and Strategy 2's bare-number regex, ending in a
	// documented failure code.
	_, ok, failCode = ParseArmAResponse(`{"value": 95`)
	if ok {
		t.Fatalf("ParseArmAResponse(malformed JSON) unexpectedly ok")
	}
	if failCode != "NO_NUMERIC_PATTERN" {
		t.Fatalf("ParseArmAResponse(malformed JSON) failCode = %q, want NO_NUMERIC_PATTERN", failCode)
	}
}

// TestParseArmAResponse_CanonicalKeyMatchesJSONNumberSemantics is a sanity
// check that our understanding of encoding/json's number decoding into
// map[string]interface{} (always float64) still holds, since the ambiguity
// logic depends on it.
func TestParseArmAResponse_CanonicalKeyMatchesJSONNumberSemantics(t *testing.T) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(`{"value": 42}`), &obj); err != nil {
		t.Fatal(err)
	}
	if _, ok := obj["value"].(float64); !ok {
		t.Fatalf("expected encoding/json to decode a JSON number as float64, got %T", obj["value"])
	}
}
