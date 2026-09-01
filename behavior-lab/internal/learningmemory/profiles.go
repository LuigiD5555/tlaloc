package learningmemory

import (
	"sort"
	"strings"
)

type empiricalProfileKey struct {
	ModelID   string
	Condition string
}

type empiricalCaseOutcome struct {
	Pass bool
}

// BuildEmpiricalProfiles derives deterministic individual and pairwise behavior
// from real-model observations. A profile is model+condition, so the same model
// under different prompts/protocols/configurations remains experimentally
// distinct. Pairwise comparisons only use cases observed by both profiles.
func BuildEmpiricalProfiles(events []Event) ([]ModelPerformanceProfile, []PairwiseModelProfile) {
	outcomes := map[empiricalProfileKey]map[string]empiricalCaseOutcome{}

	for _, event := range events {
		if event.EventType != EventObservation || event.EvidenceClass != EvidenceRealModel || event.Pass == nil {
			continue
		}
		modelID := strings.TrimSpace(event.ModelID)
		if modelID == "" {
			continue
		}
		caseKey := empiricalCaseKey(event)
		if caseKey == "" {
			continue
		}
		profile := empiricalProfileKey{ModelID: modelID, Condition: strings.TrimSpace(event.Condition)}
		cases := outcomes[profile]
		if cases == nil {
			cases = map[string]empiricalCaseOutcome{}
			outcomes[profile] = cases
		}
		pass := *event.Pass
		if existing, ok := cases[caseKey]; ok {
			// Duplicate observations for one configuration/case are collapsed
			// conservatively and order-independently: any observed failure wins.
			pass = existing.Pass && pass
		}
		cases[caseKey] = empiricalCaseOutcome{Pass: pass}
	}

	keys := make([]empiricalProfileKey, 0, len(outcomes))
	for key := range outcomes {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ModelID != keys[j].ModelID {
			return keys[i].ModelID < keys[j].ModelID
		}
		return keys[i].Condition < keys[j].Condition
	})

	individual := make([]ModelPerformanceProfile, 0, len(keys))
	for _, key := range keys {
		cases := outcomes[key]
		profile := ModelPerformanceProfile{ModelID: key.ModelID, Condition: key.Condition, Cases: len(cases)}
		for _, outcome := range cases {
			if outcome.Pass {
				profile.Passes++
			} else {
				profile.Failures++
			}
		}
		if profile.Cases > 0 {
			profile.Accuracy = float64(profile.Passes) / float64(profile.Cases)
		}
		individual = append(individual, profile)
	}

	pairwise := make([]PairwiseModelProfile, 0)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			a, b := keys[i], keys[j]
			profile := buildPairwiseProfile(a, outcomes[a], b, outcomes[b])
			if profile.SharedCases == 0 {
				continue
			}
			pairwise = append(pairwise, profile)
		}
	}
	return individual, pairwise
}

func empiricalCaseKey(event Event) string {
	benchmark := strings.TrimSpace(event.BenchmarkID)
	specimen := strings.TrimSpace(event.SpecimenSHA256)
	if specimen == "" {
		specimen = strings.TrimSpace(event.SpecimenID)
	}
	question := strings.TrimSpace(event.QuestionID)
	layer := strings.TrimSpace(event.ScoreLayer)
	// At least one stable discriminator is required. Model, provider, condition,
	// event ID and trial ID are intentionally excluded from case identity.
	if benchmark == "" && specimen == "" && question == "" {
		return ""
	}
	return benchmark + "\x00" + specimen + "\x00" + question + "\x00" + layer
}

func buildPairwiseProfile(a empiricalProfileKey, aCases map[string]empiricalCaseOutcome, b empiricalProfileKey, bCases map[string]empiricalCaseOutcome) PairwiseModelProfile {
	profile := PairwiseModelProfile{
		ModelA:     a.ModelID,
		ConditionA: a.Condition,
		ModelB:     b.ModelID,
		ConditionB: b.Condition,
	}
	for caseKey, aOutcome := range aCases {
		bOutcome, ok := bCases[caseKey]
		if !ok {
			continue
		}
		profile.SharedCases++
		switch {
		case aOutcome.Pass && bOutcome.Pass:
			profile.BothPass++
		case !aOutcome.Pass && !bOutcome.Pass:
			profile.BothFail++
		case aOutcome.Pass && !bOutcome.Pass:
			profile.ARecoversB++
		case !aOutcome.Pass && bOutcome.Pass:
			profile.BRecoversA++
		}
	}
	profile.FailureUnion = profile.BothFail + profile.ARecoversB + profile.BRecoversA
	if profile.FailureUnion > 0 {
		profile.FailureOverlap = float64(profile.BothFail) / float64(profile.FailureUnion)
		profile.Complementarity = 1 - profile.FailureOverlap
	}
	if profile.SharedCases > 0 {
		profile.OracleSuccess = float64(profile.SharedCases-profile.BothFail) / float64(profile.SharedCases)
	}
	return profile
}
