package canonicalstate

import (
	"fmt"
	"sort"
)

type FusionOutcome struct {
	Claim      CanonicalClaim
	Conflict   *Conflict
	Decisions  []CandidateDecision
	Accepted   int
	Unresolved int
}

type FusionStrategy interface {
	Fuse(groupKey string, rows []verifiedCandidate) (FusionOutcome, error)
}

type fusionStrategyFunc func(string, []verifiedCandidate) (FusionOutcome, error)

func (f fusionStrategyFunc) Fuse(groupKey string, rows []verifiedCandidate) (FusionOutcome, error) {
	return f(groupKey, rows)
}

var defaultFusionStrategies = map[FusionMode]FusionStrategy{
	FusionExact:       fusionStrategyFunc(fuseExactObject),
	FusionSingleValue: fusionStrategyFunc(fuseSingleValue),
}

func fuseExactObject(groupKey string, rows []verifiedCandidate) (FusionOutcome, error) {
	if len(rows) == 0 {
		return FusionOutcome{}, fmt.Errorf("cannot fuse empty candidate group")
	}
	var pos, neg []verifiedCandidate
	for _, row := range rows {
		if row.C.Claim.Polarity {
			pos = append(pos, row)
		} else {
			neg = append(neg, row)
		}
	}
	if len(pos) > 0 && len(neg) > 0 {
		conflict := Conflict{ID: "conflict:" + shortHash(groupKey), ClaimKey: groupKey, Status: "UNRESOLVED"}
		for _, row := range pos {
			conflict.PositiveIDs = append(conflict.PositiveIDs, row.C.ID)
		}
		for _, row := range neg {
			conflict.NegativeIDs = append(conflict.NegativeIDs, row.C.ID)
		}
		sort.Strings(conflict.PositiveIDs)
		sort.Strings(conflict.NegativeIDs)
		claim := canonicalFromConflictRows(conflict, rows)
		decisions := make([]CandidateDecision, 0, len(rows))
		for _, row := range rows {
			decisions = append(decisions, CandidateDecision{CandidateID: row.C.ID, Status: "CONFLICT", Reasons: []string{conflict.ID}})
		}
		return FusionOutcome{Claim: claim, Conflict: &conflict, Decisions: decisions, Unresolved: 1}, nil
	}
	active := pos
	if len(active) == 0 {
		active = neg
	}
	return mergedFusionOutcome(active), nil
}

func fuseSingleValue(groupKey string, rows []verifiedCandidate) (FusionOutcome, error) {
	if len(rows) == 0 {
		return FusionOutcome{}, fmt.Errorf("cannot fuse empty candidate group")
	}
	values := map[string][]verifiedCandidate{}
	labels := map[string]string{}
	for _, row := range rows {
		key := fmt.Sprintf("%t\x00%s", row.C.Claim.Polarity, norm(row.C.Claim.Object))
		values[key] = append(values[key], row)
		prefix := "-"
		if row.C.Claim.Polarity {
			prefix = "+"
		}
		labels[key] = prefix + norm(row.C.Claim.Object)
	}
	if len(values) == 1 {
		return mergedFusionOutcome(rows), nil
	}

	conflict := Conflict{ID: "conflict:" + shortHash(groupKey), ClaimKey: groupKey, Status: "UNRESOLVED"}
	allIDs := make([]string, 0, len(rows))
	valueLabels := make([]string, 0, len(values))
	for _, label := range labels {
		valueLabels = append(valueLabels, label)
	}
	sort.Strings(valueLabels)
	for _, row := range rows {
		allIDs = append(allIDs, row.C.ID)
		if row.C.Claim.Polarity {
			conflict.PositiveIDs = append(conflict.PositiveIDs, row.C.ID)
		} else {
			conflict.NegativeIDs = append(conflict.NegativeIDs, row.C.ID)
		}
	}
	sort.Strings(allIDs)
	sort.Strings(conflict.PositiveIDs)
	sort.Strings(conflict.NegativeIDs)
	conflict.CandidateIDs = allIDs
	conflict.Values = valueLabels
	claim := canonicalFromConflictRows(conflict, rows)
	decisions := make([]CandidateDecision, 0, len(rows))
	for _, row := range rows {
		decisions = append(decisions, CandidateDecision{CandidateID: row.C.ID, Status: "CONFLICT", Reasons: []string{conflict.ID}})
	}
	return FusionOutcome{Claim: claim, Conflict: &conflict, Decisions: decisions, Unresolved: 1}, nil
}

func mergedFusionOutcome(rows []verifiedCandidate) FusionOutcome {
	claim := canonicalFromGroup(rows)
	decisions := make([]CandidateDecision, 0, len(rows))
	for i, row := range rows {
		status := "MERGED"
		if i == 0 {
			status = "ACCEPTED"
		}
		decisions = append(decisions, CandidateDecision{CandidateID: row.C.ID, Status: status})
	}
	return FusionOutcome{Claim: claim, Decisions: decisions, Accepted: len(rows)}
}
