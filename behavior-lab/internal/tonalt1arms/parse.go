package tonalt1arms

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// ParseArmAResponse parses Arm A's model response into a numeric value.
// This contract is frozen — no adjustments after the first model call.
//
// Strategy:
// 1. Try strict JSON parse first (object with numeric fields).
// 2. Fall back to bare-number regex match (^\s*-?\d+(\.\d+)?\s*$).
// 3. On any failure, return (NaN, false, "PARSE_FAILED").
//
// The model (1.6B VLM at temp=0, max_tokens=32) is unlikely to respect
// "respond in JSON" for a complex composite image and task. Both fallbacks
// must be nailed down and unit-tested before inference starts.
func ParseArmAResponse(raw string) (value float64, ok bool, failureCode string) {
	// Trim leading/trailing whitespace
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, "EMPTY_RESPONSE"
	}

	// Strategy 1: Try JSON parse
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &jsonObj); err == nil {
		// Extract the first numeric field found
		for _, v := range jsonObj {
			if f, ok := v.(float64); ok {
				return f, true, ""
			}
		}
		// JSON parse succeeded but no numeric field found
		return 0, false, "JSON_NO_NUMERIC_FIELD"
	}

	// Strategy 2: Bare-number regex
	// Matches: optional sign, one or more digits, optional decimal part
	numRegex := regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
	if numRegex.MatchString(raw) {
		f, err := strconv.ParseFloat(raw, 64)
		if err == nil {
			return f, true, ""
		}
		return 0, false, "PARSE_FLOAT_FAILED"
	}

	// No parse strategy worked
	return 0, false, "NO_NUMERIC_PATTERN"
}
