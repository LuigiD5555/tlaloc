package tonalt1arms

import (
	"encoding/json"
	"regexp"
	"sort"
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
//
// JSON-object handling (frozen policy: T1_D5_ARM_A_POLICY.json's
// parser:"json_extract_numeric_field" names no specific answer key — its
// parrot_prompt block doesn't ask the model to use a particular field name
// either). Since no canonical key is frozen anywhere, this does NOT rank
// candidate keys by an invented priority (that would itself be an
// unauthorized invention). Instead it deterministically collects the set of
// DISTINCT numeric values across all top-level fields:
//   - zero numeric fields -> JSON_NO_NUMERIC_FIELD
//   - exactly one distinct value (whether from one field, or several fields
//     that happen to agree) -> accept it
//   - more than one distinct value -> fail closed, AMBIGUOUS_MODEL_OUTPUT
//
// Distinctness is computed via a value-keyed set, so Go's randomized map
// iteration order can never influence the accept/reject decision — order is
// only ever used to pick a deterministic (sorted-by-key) representative
// value to report, never to break a real tie between different values.
func ParseArmAResponse(raw string) (value float64, ok bool, failureCode string) {
	// Trim leading/trailing whitespace
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, "EMPTY_RESPONSE"
	}

	// Strategy 1: Try JSON parse
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &jsonObj); err == nil {
		return parseJSONNumericObject(jsonObj)
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

// parseJSONNumericObject implements the deterministic candidate-collection
// rule described above. It never depends on map iteration order for its
// accept/reject decision: keys are sorted first purely so that, in the
// single-distinct-value case, the reported value is picked the same way on
// every run regardless of Go's randomized map iteration.
func parseJSONNumericObject(jsonObj map[string]interface{}) (value float64, ok bool, failureCode string) {
	keys := make([]string, 0, len(jsonObj))
	for k := range jsonObj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	distinct := make(map[float64]struct{})
	var firstByKeyOrder float64
	found := false
	for _, k := range keys {
		f, isNum := jsonObj[k].(float64)
		if !isNum {
			continue
		}
		if !found {
			firstByKeyOrder = f
			found = true
		}
		distinct[f] = struct{}{}
	}

	if !found {
		return 0, false, "JSON_NO_NUMERIC_FIELD"
	}
	if len(distinct) > 1 {
		return 0, false, "AMBIGUOUS_MODEL_OUTPUT"
	}
	return firstByKeyOrder, true, ""
}
