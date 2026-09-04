package tonalt1arms

import (
	"math"
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
		{`{"a": 1, "b": 2}`, 1, true}, // First field wins
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
