package selectionprofiles

import (
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const CatalogSchema = "tlaloc.empirical-selection-catalog.r1"

type WorkerBinding struct {
	WorkerID   string  `json:"worker_id"`
	ModelID    string  `json:"model_id"`
	Condition  string  `json:"condition,omitempty"`
	Observed   bool    `json:"observed"`
	Cases      int     `json:"cases,omitempty"`
	Accuracy   float64 `json:"accuracy,omitempty"`
}

type PairBinding struct {
	WorkerA          string  `json:"worker_a"`
	WorkerB          string  `json:"worker_b"`
	SharedCases      int     `json:"shared_cases"`
	Complementarity float64 `json:"complementarity"`
	FailureOverlap  float64 `json:"failure_overlap"`
	OracleSuccess   float64 `json:"oracle_success"`
}

type Catalog struct {
	Schema        string          `json:"schema"`
	SummarySchema string          `json:"summary_schema,omitempty"`
	Workers       []WorkerBinding `json:"workers,omitempty"`
	Pairs         []PairBinding   `json:"pairs,omitempty"`
}

// Build binds stable Tlaloque worker IDs to the exact model+condition profiles
// declared on their descriptors. Missing empirical observations are retained in
// the catalog as Observed=false but are omitted from the strategy source, so
// routing falls back to its normal prior until Behavior Lab has evidence.
func Build(summary learningmemory.Summary, descriptors []tlaloque.CapabilityDescriptor) (Catalog, *tlaloque.StaticEmpiricalSelectionSource, error) {
	catalog := Catalog{Schema: CatalogSchema, SummarySchema: summary.Schema}
	profileMetrics := map[string]learningmemory.ModelPerformanceProfile{}
	for _, profile := range summary.ModelProfiles {
		key := profileKey(profile.ModelID, profile.Condition)
		if key == "" {
			continue
		}
		if _, exists := profileMetrics[key]; exists {
			return Catalog{}, nil, fmt.Errorf("duplicate model profile %q/%q", profile.ModelID, profile.Condition)
		}
		profileMetrics[key] = profile
	}

	profileWorkers := map[string][]string{}
	workerMetrics := []tlaloque.WorkerEmpiricalMetric{}
	seenWorkers := map[string]struct{}{}
	for _, raw := range descriptors {
		desc, err := raw.Normalize()
		if err != nil {
			return Catalog{}, nil, err
		}
		if _, exists := seenWorkers[desc.ID]; exists {
			return Catalog{}, nil, fmt.Errorf("duplicate worker descriptor %q", desc.ID)
		}
		seenWorkers[desc.ID] = struct{}{}
		if desc.EmpiricalProfile == nil {
			continue
		}
		profile := *desc.EmpiricalProfile
		key := profile.Key()
		binding := WorkerBinding{
			WorkerID:  desc.ID,
			ModelID:   profile.ModelID,
			Condition: profile.Condition,
		}
		if metric, ok := profileMetrics[key]; ok {
			binding.Observed = true
			binding.Cases = metric.Cases
			binding.Accuracy = metric.Accuracy
			workerMetrics = append(workerMetrics, tlaloque.WorkerEmpiricalMetric{
				WorkerID: desc.ID,
				Cases:    metric.Cases,
				Accuracy: metric.Accuracy,
			})
		}
		catalog.Workers = append(catalog.Workers, binding)
		profileWorkers[key] = append(profileWorkers[key], desc.ID)
	}
	for key := range profileWorkers {
		sort.Strings(profileWorkers[key])
	}
	sort.Slice(catalog.Workers, func(i, j int) bool { return catalog.Workers[i].WorkerID < catalog.Workers[j].WorkerID })

	pairMetrics := []tlaloque.WorkerPairEmpiricalMetric{}
	seenPairs := map[string]struct{}{}
	for _, pair := range summary.PairwiseProfiles {
		aWorkers := profileWorkers[profileKey(pair.ModelA, pair.ConditionA)]
		bWorkers := profileWorkers[profileKey(pair.ModelB, pair.ConditionB)]
		for _, workerA := range aWorkers {
			for _, workerB := range bWorkers {
				if workerA == workerB {
					continue
				}
				a, b := canonicalWorkerPair(workerA, workerB)
				key := a + "\x00" + b
				if _, exists := seenPairs[key]; exists {
					return Catalog{}, nil, fmt.Errorf("duplicate worker pair %q/%q", a, b)
				}
				seenPairs[key] = struct{}{}
				catalog.Pairs = append(catalog.Pairs, PairBinding{
					WorkerA:          a,
					WorkerB:          b,
					SharedCases:      pair.SharedCases,
					Complementarity: pair.Complementarity,
					FailureOverlap:  pair.FailureOverlap,
					OracleSuccess:   pair.OracleSuccess,
				})
				pairMetrics = append(pairMetrics, tlaloque.WorkerPairEmpiricalMetric{
					WorkerA:          a,
					WorkerB:          b,
					SharedCases:      pair.SharedCases,
					Complementarity: pair.Complementarity,
				})
			}
		}
	}
	sort.Slice(catalog.Pairs, func(i, j int) bool {
		if catalog.Pairs[i].WorkerA != catalog.Pairs[j].WorkerA {
			return catalog.Pairs[i].WorkerA < catalog.Pairs[j].WorkerA
		}
		return catalog.Pairs[i].WorkerB < catalog.Pairs[j].WorkerB
	})

	source, err := tlaloque.NewStaticEmpiricalSelectionSource(workerMetrics, pairMetrics)
	if err != nil {
		return Catalog{}, nil, err
	}
	return catalog, source, nil
}

func profileKey(modelID, condition string) string {
	modelID = strings.TrimSpace(modelID)
	condition = strings.TrimSpace(condition)
	if modelID == "" {
		return ""
	}
	return modelID + "\x00" + condition
}

func canonicalWorkerPair(a, b string) (string, string) {
	if b < a {
		return b, a
	}
	return a, b
}
