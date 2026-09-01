package swarmbench

import (
	"context"
	"time"
)

// This file wires a live, resident local model directly into the swarm as
// its intent/entity individuals — the honest counterpart to the calibrated
// lfm2vl_proxy.go stand-in. The proxy approximates what the probe measured;
// these functions make the real call every time, so a Phase 5 comparison
// against the single-shot baseline (lmstudio_baseline.go) uses the same live
// model on both sides of the comparison, not a live model on one side and an
// extrapolated approximation of it on the other.

const liveCallTimeout = 30 * time.Second

// IntentLogicLive returns an IntentLogicFunc that calls the resident model
// for every invocation. A transport or parse failure degrades to an empty
// label with zero confidence — the same "wrong answer, not a crash" rule
// intentWorker already documents — rather than aborting the swarm run.
func IntentLogicLive(client LMStudioClient) IntentLogicFunc {
	return func(text string) (string, float64) {
		ctx, cancel := context.WithTimeout(context.Background(), liveCallTimeout)
		defer cancel()
		intent, _, err := client.ClassifyIntent(ctx, text)
		if err != nil || intent == "" {
			return "", 0
		}
		return intent, 1
	}
}

// EntityLogicLive is IntentLogicLive's counterpart for organization
// extraction.
func EntityLogicLive(client LMStudioClient) EntityLogicFunc {
	return func(text string) (string, float64) {
		ctx, cancel := context.WithTimeout(context.Background(), liveCallTimeout)
		defer cancel()
		organization, _, err := client.ExtractOrganization(ctx, text)
		if err != nil || organization == "" {
			return "", 0
		}
		return organization, 1
	}
}
