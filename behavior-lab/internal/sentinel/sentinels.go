package sentinel

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// PermissionSentinel independently re-checks a proposed action against the
// policy — belt and suspenders for envelope.Compile. If the two ever
// disagree that is itself a serious bug.
type PermissionSentinel struct{}

func (PermissionSentinel) Name() string { return "permission-sentinel" }

func (PermissionSentinel) Inspect(_ context.Context, subject Subject) ([]Concern, error) {
	if subject.ProposedAction == nil || subject.Policy == nil {
		return nil, nil
	}
	var concerns []Concern
	if subject.ProposedAction.Risk > subject.Policy.MaxRisk {
		concerns = append(concerns, Concern{
			Kind: "risk_over_ceiling", Severity: Block,
			Detail: fmt.Sprintf("action is %s, policy ceiling is %s", subject.ProposedAction.Risk, subject.Policy.MaxRisk),
		})
	}
	if len(subject.Policy.AllowedCapabilities) > 0 {
		allowed := false
		for _, name := range subject.Policy.AllowedCapabilities {
			if name == subject.ProposedAction.Capability {
				allowed = true
			}
		}
		if !allowed {
			concerns = append(concerns, Concern{
				Kind: "capability_not_allowed", Severity: Block,
				Detail: fmt.Sprintf("%q is not on the policy allow-list", subject.ProposedAction.Capability),
			})
		}
	}
	return concerns, nil
}

// ScopeSentinel checks that every path-like argument of a proposed action
// stays inside the allowed paths — independent of the envelope's own
// StayInside check.
type ScopeSentinel struct{}

func (ScopeSentinel) Name() string { return "scope-sentinel" }

func (ScopeSentinel) Inspect(_ context.Context, subject Subject) ([]Concern, error) {
	if subject.ProposedAction == nil || len(subject.AllowedPaths) == 0 {
		return nil, nil
	}
	var concerns []Concern
	for name, value := range subject.ProposedAction.Arguments {
		if !looksLikePath(value) {
			continue
		}
		if !pathInside(value, subject.AllowedPaths) {
			concerns = append(concerns, Concern{
				Kind: "path_out_of_scope", Severity: Block,
				Detail: fmt.Sprintf("argument %s=%q is outside %v", name, value, subject.AllowedPaths),
			})
		}
	}
	return concerns, nil
}

// ConflictSentinel flags when two blackboard observations assign different
// values to the same key — an unresolved disagreement, not something to
// silently pick a winner for.
type ConflictSentinel struct{}

func (ConflictSentinel) Name() string { return "conflict-sentinel" }

func (ConflictSentinel) Inspect(_ context.Context, subject Subject) ([]Concern, error) {
	byKey := map[string]map[string][]string{}
	for _, observation := range subject.Observations {
		if byKey[observation.Key] == nil {
			byKey[observation.Key] = map[string][]string{}
		}
		byKey[observation.Key][observation.Value] = append(byKey[observation.Key][observation.Value], observation.Source)
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var concerns []Concern
	for _, key := range keys {
		if len(byKey[key]) < 2 {
			continue
		}
		values := make([]string, 0, len(byKey[key]))
		for value := range byKey[key] {
			values = append(values, value)
		}
		sort.Strings(values)
		concerns = append(concerns, Concern{
			Kind: "observation_conflict", Severity: Warn,
			Detail: fmt.Sprintf("key %q has conflicting values: %v", key, values),
		})
	}
	return concerns, nil
}

// OODSentinel raises a concern when the answer's confidence would not clear
// the specialist's measured calibration (see internal/tlaloque/calibration).
type OODSentinel struct{}

func (OODSentinel) Name() string { return "ood-sentinel" }

func (OODSentinel) Inspect(_ context.Context, subject Subject) ([]Concern, error) {
	if subject.Calibration == nil {
		return nil, nil
	}
	verdict := subject.Calibration.Verdict(calibration.Query{Confidence: subject.AnswerConfidence})
	if verdict == calibration.Answered {
		return nil, nil
	}
	return []Concern{{
		Kind: "out_of_distribution", Severity: Warn,
		Detail: fmt.Sprintf("calibration verdict for this input is %s, not ANSWERED", verdict),
	}}, nil
}

// NumericConsistencySentinel flags years/numbers stated in the answer that
// do not appear anywhere in the evidence — a cheap hallucination check.
type NumericConsistencySentinel struct{}

func (NumericConsistencySentinel) Name() string { return "numeric-consistency-sentinel" }

var numberToken = regexp.MustCompile(`\b\d[\d.,]*\b`)

func (NumericConsistencySentinel) Inspect(_ context.Context, subject Subject) ([]Concern, error) {
	if subject.Answer == "" || subject.Evidence == "" {
		return nil, nil
	}
	evidence := subject.Evidence
	seen := map[string]bool{}
	var unsupported []string
	for _, match := range numberToken.FindAllString(subject.Answer, -1) {
		normalized := strings.TrimRight(match, ".,")
		if len(normalized) < 2 || seen[normalized] {
			continue
		}
		seen[normalized] = true
		if !strings.Contains(evidence, normalized) {
			unsupported = append(unsupported, normalized)
		}
	}
	if len(unsupported) == 0 {
		return nil, nil
	}
	sort.Strings(unsupported)
	return []Concern{{
		Kind: "unsupported_number", Severity: Warn,
		Detail: fmt.Sprintf("answer states number(s) not found in the evidence: %v", unsupported),
	}}, nil
}

func looksLikePath(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") || strings.Contains(value, "/")
}

func pathInside(path string, prefixes []string) bool {
	clean := strings.TrimSpace(path)
	for _, prefix := range prefixes {
		prefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return true
		}
	}
	return false
}

// compile-time assertion that the concrete sentinels satisfy the interface.
var _ = []Sentinel{
	PermissionSentinel{}, ScopeSentinel{}, ConflictSentinel{}, OODSentinel{},
	NumericConsistencySentinel{},
}
