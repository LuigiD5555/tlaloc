package tlaloque

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type WorkerEmpiricalMetric struct {
	WorkerID string  `json:"worker_id"`
	Cases    int     `json:"cases"`
	Accuracy float64 `json:"accuracy"`
}

type WorkerPairEmpiricalMetric struct {
	WorkerA          string  `json:"worker_a"`
	WorkerB          string  `json:"worker_b"`
	SharedCases      int     `json:"shared_cases"`
	Complementarity float64 `json:"complementarity"`
}

// EmpiricalSelectionSource is deliberately independent from Learning Memory's
// storage model. An adapter can map model+condition profiles to worker IDs
// without making the Tlaloque registry import Behavior Lab persistence types.
type EmpiricalSelectionSource interface {
	WorkerMetric(workerID string) (WorkerEmpiricalMetric, bool)
	PairMetric(workerA, workerB string) (WorkerPairEmpiricalMetric, bool)
}

type StaticEmpiricalSelectionSource struct {
	workers map[string]WorkerEmpiricalMetric
	pairs   map[string]WorkerPairEmpiricalMetric
}

func NewStaticEmpiricalSelectionSource(workers []WorkerEmpiricalMetric, pairs []WorkerPairEmpiricalMetric) (*StaticEmpiricalSelectionSource, error) {
	source := &StaticEmpiricalSelectionSource{
		workers: map[string]WorkerEmpiricalMetric{},
		pairs:   map[string]WorkerPairEmpiricalMetric{},
	}
	for _, metric := range workers {
		metric.WorkerID = strings.TrimSpace(metric.WorkerID)
		if metric.WorkerID == "" {
			return nil, fmt.Errorf("worker empirical metric requires worker_id")
		}
		if metric.Cases < 0 || math.IsNaN(metric.Accuracy) || metric.Accuracy < 0 || metric.Accuracy > 1 {
			return nil, fmt.Errorf("worker %q has invalid empirical metric", metric.WorkerID)
		}
		if _, exists := source.workers[metric.WorkerID]; exists {
			return nil, fmt.Errorf("duplicate worker empirical metric %q", metric.WorkerID)
		}
		source.workers[metric.WorkerID] = metric
	}
	for _, metric := range pairs {
		metric.WorkerA = strings.TrimSpace(metric.WorkerA)
		metric.WorkerB = strings.TrimSpace(metric.WorkerB)
		if metric.WorkerA == "" || metric.WorkerB == "" || metric.WorkerA == metric.WorkerB {
			return nil, fmt.Errorf("pair empirical metric requires two distinct worker ids")
		}
		if metric.SharedCases < 0 || math.IsNaN(metric.Complementarity) || metric.Complementarity < 0 || metric.Complementarity > 1 {
			return nil, fmt.Errorf("pair %q/%q has invalid empirical metric", metric.WorkerA, metric.WorkerB)
		}
		key := empiricalPairKey(metric.WorkerA, metric.WorkerB)
		if _, exists := source.pairs[key]; exists {
			return nil, fmt.Errorf("duplicate pair empirical metric %q/%q", metric.WorkerA, metric.WorkerB)
		}
		source.pairs[key] = metric
	}
	return source, nil
}

func (s *StaticEmpiricalSelectionSource) WorkerMetric(workerID string) (WorkerEmpiricalMetric, bool) {
	if s == nil {
		return WorkerEmpiricalMetric{}, false
	}
	metric, ok := s.workers[strings.TrimSpace(workerID)]
	return metric, ok
}

func (s *StaticEmpiricalSelectionSource) PairMetric(workerA, workerB string) (WorkerPairEmpiricalMetric, bool) {
	if s == nil {
		return WorkerPairEmpiricalMetric{}, false
	}
	metric, ok := s.pairs[empiricalPairKey(workerA, workerB)]
	return metric, ok
}

func empiricalPairKey(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if b < a {
		a, b = b, a
	}
	return a + "\x00" + b
}

// EmpiricalEnsembleSelectionStrategy preserves ordinary single-worker routing
// and changes only multi-selection. The first member is the fallback strategy's
// best worker. Each later member maximizes empirical quality plus average
// complementarity with the already selected set; fallback rank is the stable
// tie-break and the prior when empirical quality is unavailable.
type EmpiricalEnsembleSelectionStrategy struct {
	Fallback              SelectionStrategy
	Source                EmpiricalSelectionSource
	MinCases              int
	MinSharedCases        int
	QualityWeight         float64
	ComplementarityWeight float64
}

func (s EmpiricalEnsembleSelectionStrategy) Select(candidates []SelectionCandidate, req SelectionRequest) Result[CapabilityWorker] {
	return s.fallback().Select(candidates, req)
}

func (s EmpiricalEnsembleSelectionStrategy) SelectMany(candidates []SelectionCandidate, req SelectionRequest, limit int) Result[[]CapabilityWorker] {
	if limit <= 0 {
		return DomainResult[[]CapabilityWorker](ResultInvalidRequest, nil, Diagnostic{Code: "INVALID_SELECTION_LIMIT", Message: "selection limit must be greater than zero"})
	}
	if len(candidates) == 0 {
		return DomainResult[[]CapabilityWorker](ResultNoCandidate, nil, noEligibleWorkerDiagnostic(req))
	}
	if limit > len(candidates) {
		limit = len(candidates)
	}

	baselineResult := selectManyByRepeatedStrategy(s.fallback(), candidates, req, len(candidates))
	if baselineResult.Err != nil {
		return Failure[[]CapabilityWorker](baselineResult.Err)
	}
	if baselineResult.Code != ResultSuccess || len(baselineResult.Value) == 0 {
		return DomainResult[[]CapabilityWorker](baselineResult.Code, baselineResult.Value, baselineResult.Diagnostics...)
	}
	baseline := baselineResult.Value
	if limit == 1 || len(baseline) == 1 {
		return Success(append([]CapabilityWorker(nil), baseline[:1]...))
	}

	rank := make(map[string]int, len(baseline))
	remaining := make(map[string]CapabilityWorker, len(baseline))
	for i, worker := range baseline {
		id := worker.Descriptor().ID
		rank[id] = i
		remaining[id] = worker
	}

	selected := []CapabilityWorker{baseline[0]}
	delete(remaining, baseline[0].Descriptor().ID)
	for len(selected) < limit && len(remaining) > 0 {
		var best CapabilityWorker
		bestUtility := math.Inf(-1)
		bestRank := math.MaxInt
		for _, candidate := range baseline {
			id := candidate.Descriptor().ID
			if _, ok := remaining[id]; !ok {
				continue
			}
			utility := s.utility(candidate, selected, rank[id], len(baseline))
			if best == nil || utility > bestUtility || (nearlyEqual(utility, bestUtility) && rank[id] < bestRank) {
				best = candidate
				bestUtility = utility
				bestRank = rank[id]
			}
		}
		if best == nil {
			break
		}
		selected = append(selected, best)
		delete(remaining, best.Descriptor().ID)
	}
	return Success(selected)
}

func (s EmpiricalEnsembleSelectionStrategy) fallback() SelectionStrategy {
	if s.Fallback != nil {
		return s.Fallback
	}
	return RankedSelectionStrategy{Scoring: DefaultScoringStrategy{}}
}

func (s EmpiricalEnsembleSelectionStrategy) utility(candidate CapabilityWorker, selected []CapabilityWorker, fallbackRank, candidateCount int) float64 {
	qualityWeight, complementarityWeight := s.weights()
	quality := fallbackRankPrior(fallbackRank, candidateCount)
	minCases := s.MinCases
	if minCases <= 0 {
		minCases = 1
	}
	candidateID := candidate.Descriptor().ID
	if s.Source != nil {
		if metric, ok := s.Source.WorkerMetric(candidateID); ok && metric.Cases >= minCases {
			quality = clampUnit(metric.Accuracy)
		}
	}

	minSharedCases := s.MinSharedCases
	if minSharedCases <= 0 {
		minSharedCases = 1
	}
	complementaritySum := 0.0
	complementarityPairs := 0
	if s.Source != nil {
		for _, peer := range selected {
			metric, ok := s.Source.PairMetric(candidateID, peer.Descriptor().ID)
			if !ok || metric.SharedCases < minSharedCases {
				continue
			}
			complementaritySum += clampUnit(metric.Complementarity)
			complementarityPairs++
		}
	}
	complementarity := 0.0
	if complementarityPairs > 0 {
		complementarity = complementaritySum / float64(complementarityPairs)
	}
	return qualityWeight*quality + complementarityWeight*complementarity
}

func (s EmpiricalEnsembleSelectionStrategy) weights() (float64, float64) {
	quality := s.QualityWeight
	complementarity := s.ComplementarityWeight
	if quality < 0 {
		quality = 0
	}
	if complementarity < 0 {
		complementarity = 0
	}
	if quality == 0 && complementarity == 0 {
		return 1, 1
	}
	return quality, complementarity
}

func fallbackRankPrior(rank, count int) float64 {
	if count <= 1 {
		return 1
	}
	if rank < 0 {
		rank = count - 1
	}
	if rank >= count {
		rank = count - 1
	}
	return 1 - float64(rank)/float64(count-1)
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12
}

// SortedWorkerMetrics is useful for diagnostics and reproducible artifacts.
func (s *StaticEmpiricalSelectionSource) SortedWorkerMetrics() []WorkerEmpiricalMetric {
	if s == nil {
		return nil
	}
	out := make([]WorkerEmpiricalMetric, 0, len(s.workers))
	for _, metric := range s.workers {
		out = append(out, metric)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WorkerID < out[j].WorkerID })
	return out
}
