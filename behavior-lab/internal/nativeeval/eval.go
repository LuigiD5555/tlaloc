package nativeeval

import (
	"strings"
)

const SchemaR0 = "tlaloc.native-semantic-regression.r0"

type QueryClass string

const (
	QueryIdentity QueryClass = "IDENTITY"
	QueryIndex    QueryClass = "INDEX"
	QueryOverview QueryClass = "OVERVIEW"
	QueryLocate   QueryClass = "LOCATE_TOPIC"
	QueryExact    QueryClass = "EXACT"
)

type Trial struct {
	Schema          string     `json:"schema"`
	ID              string     `json:"id"`
	QueryClass      QueryClass `json:"query_class"`
	Question        string     `json:"question"`
	ExpectedEntries []string   `json:"expected_entries,omitempty"`
	ModelOutput     string     `json:"model_output"`
	DeclaredExactCapability bool `json:"declared_exact_capability"`
}

type Result struct {
	Schema                        string   `json:"schema"`
	TrialID                       string   `json:"trial_id"`
	IndexRecoveryRate             float64  `json:"index_recovery_rate"`
	SemanticAnswerPresent         bool     `json:"semantic_answer_present"`
	MechanicalDependencyViolation bool     `json:"mechanical_dependency_violation"`
	UnverifiedMechanicalClaims    []string `json:"unverified_mechanical_claims,omitempty"`
	Pass                          bool     `json:"pass"`
}

func Evaluate(t Trial) Result {
	out := normalize(t.ModelOutput)
	result := Result{Schema: SchemaR0 + ".result", TrialID: t.ID}
	if len(t.ExpectedEntries) > 0 {
		matched := 0
		for _, entry := range t.ExpectedEntries {
			if strings.Contains(out, normalize(entry)) { matched++ }
		}
		result.IndexRecoveryRate = float64(matched) / float64(len(t.ExpectedEntries))
	}
	result.SemanticAnswerPresent = semanticAnswerPresent(t.QueryClass, result.IndexRecoveryRate, out)
	if isSemantic(t.QueryClass) {
		result.MechanicalDependencyViolation = containsAny(out, []string{
			"NEED THE ORIGINAL FILE", "NEED THE IMAGE FILE", "NEED ACCESS TO THE FILE", "NEED THE BINARY",
			"EXTRACT THE BITS", "EXTRACT BITS", "READ THE BITS", "DECODE THE BINARY", "DECOMPRESS THE",
			"CANNOT READ THE PAYLOAD", "NEED A DECODER",
		})
	}
	if !t.DeclaredExactCapability {
		for _, marker := range []struct{label string; phrases []string}{
			{"BYTE_LAYOUT", []string{" BYTES", "BYTE HEADER", "BYTE PAYLOAD"}},
			{"COMPRESSION", []string{"BZIP2", "GZIP", "ZSTD", "COMPRESSED RESIDUAL"}},
			{"HASH", []string{"SHA256:", "SHA-256:", "HASH SHA256"}},
			{"ARCHIVE", []string{"ARCHIVE CONTENTS", "DECOMPRESS THE RESIDUAL"}},
		} {
			if containsAny(out, marker.phrases) { result.UnverifiedMechanicalClaims = append(result.UnverifiedMechanicalClaims, marker.label) }
		}
	}
	result.Pass = result.SemanticAnswerPresent && !result.MechanicalDependencyViolation && len(result.UnverifiedMechanicalClaims) == 0
	return result
}

func semanticAnswerPresent(class QueryClass, indexRate float64, output string) bool {
	switch class {
	case QueryIndex:
		return indexRate >= .75
	case QueryIdentity, QueryOverview, QueryLocate:
		return strings.TrimSpace(output) != "" && !strings.Contains(output, "UNKNOWN")
	case QueryExact:
		return strings.TrimSpace(output) != ""
	default:
		return false
	}
}

func isSemantic(class QueryClass) bool {
	return class == QueryIdentity || class == QueryIndex || class == QueryOverview || class == QueryLocate
}

func containsAny(text string, phrases []string) bool {
	for _, phrase := range phrases { if strings.Contains(text, phrase) { return true } }
	return false
}

func normalize(value string) string {
	return strings.Join(strings.Fields(strings.ToUpper(value)), " ")
}
