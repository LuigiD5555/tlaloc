package foldtest

import (
	"encoding/json"
	"time"
)

// MetricsSnapshot captures test execution metrics
type MetricsSnapshot struct {
	Timestamp              time.Time        `json:"timestamp"`
	TotalDuration          time.Duration    `json:"total_duration"`
	SessionResults         []SessionResult  `json:"session_results"`
	AverageTurns           float64          `json:"average_turns"`
	AverageTokensPerTurn   float64          `json:"average_tokens_per_turn"`
	CommandLearningRate    float64          `json:"command_learning_rate"`
	SuccessfulUnfolds      int              `json:"successful_unfolds"`
	TotalUnfolds           int              `json:"total_unfolds"`
}

// ComputeMetrics aggregates results from multiple sessions
func ComputeMetrics(results []SessionResult, duration time.Duration) MetricsSnapshot {
	snapshot := MetricsSnapshot{
		Timestamp:      time.Now(),
		TotalDuration:  duration,
		SessionResults: results,
	}

	if len(results) == 0 {
		return snapshot
	}

	// Compute aggregates
	var totalTurns, totalTokens int
	learnedCount := 0
	for _, r := range results {
		totalTurns += r.Turns
		totalTokens += r.TotalTokensPrompt + r.TotalTokensCompletion
		if r.LearnedCommandUsed {
			learnedCount++
		}
		snapshot.SuccessfulUnfolds += len(r.Unfolds)
		snapshot.TotalUnfolds += len(r.Unfolds)
	}

	snapshot.AverageTurns = float64(totalTurns) / float64(len(results))
	if totalTurns > 0 {
		snapshot.AverageTokensPerTurn = float64(totalTokens) / float64(totalTurns)
	}
	snapshot.CommandLearningRate = float64(learnedCount) / float64(len(results))

	return snapshot
}

// MarshalJSON produces structured JSON output for the metrics
func (m MetricsSnapshot) MarshalJSON() ([]byte, error) {
	type Alias MetricsSnapshot
	return json.MarshalIndent((*Alias)(&m), "", "  ")
}
