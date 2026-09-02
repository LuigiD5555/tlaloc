package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// JSONValid checks that raw is well-formed JSON.
func JSONValid(raw []byte) Check {
	check := Check{Level: Structural, Kind: "json_valid"}
	check.Passed = json.Valid(raw)
	if !check.Passed {
		check.Detail = "output is not valid JSON"
	}
	return check
}

// HashMatches checks that data's SHA-256 equals expectedHex (case-
// insensitive, any length prefix of the full digest is accepted so callers
// can store a short id).
func HashMatches(data []byte, expectedHex string) Check {
	check := Check{Level: Structural, Kind: "hash_match"}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	want := normalizeHex(expectedHex)
	check.Passed = want != "" && len(want) <= len(actual) && actual[:len(want)] == want
	if !check.Passed {
		check.Detail = fmt.Sprintf("hash %s does not match expected %s", actual[:min(16, len(actual))], want)
	}
	return check
}

// RequiredFields checks that raw is a JSON object with every named key
// present and non-null.
func RequiredFields(raw []byte, fields []string) Check {
	check := Check{Level: Structural, Kind: "required_fields"}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		check.Detail = "output is not a JSON object"
		return check
	}
	missing := []string{}
	for _, field := range fields {
		value, present := obj[field]
		if !present || string(value) == "null" {
			missing = append(missing, field)
		}
	}
	sort.Strings(missing)
	check.Passed = len(missing) == 0
	if !check.Passed {
		check.Detail = "missing/null fields: " + fmt.Sprint(missing)
	}
	return check
}

// InRange checks min <= value <= max.
func InRange(name string, value, minValue, maxValue float64) Check {
	check := Check{Level: Structural, Kind: "in_range"}
	check.Passed = value >= minValue && value <= maxValue
	if !check.Passed {
		check.Detail = fmt.Sprintf("%s=%g outside [%g, %g]", name, value, minValue, maxValue)
	}
	return check
}

// OneOf checks that value is in allowed.
func OneOf(name, value string, allowed []string) Check {
	check := Check{Level: Structural, Kind: "one_of"}
	for _, candidate := range allowed {
		if candidate == value {
			check.Passed = true
			return check
		}
	}
	check.Detail = fmt.Sprintf("%s=%q not one of %v", name, value, allowed)
	return check
}

// DependenciesSatisfied checks that every id in needed is in satisfied.
func DependenciesSatisfied(needed, satisfied []string) Check {
	check := Check{Level: Structural, Kind: "dependencies_satisfied"}
	have := map[string]bool{}
	for _, id := range satisfied {
		have[id] = true
	}
	missing := []string{}
	for _, id := range needed {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	check.Passed = len(missing) == 0
	if !check.Passed {
		check.Detail = "unsatisfied dependencies: " + fmt.Sprint(missing)
	}
	return check
}

func normalizeHex(value string) string {
	out := make([]byte, 0, len(value))
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9', char >= 'a' && char <= 'f':
			out = append(out, byte(char))
		case char >= 'A' && char <= 'F':
			out = append(out, byte(char-'A'+'a'))
		}
	}
	return string(out)
}
