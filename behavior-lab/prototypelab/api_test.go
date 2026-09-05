package prototypelab_test

import (
	"os"
	"testing"
	"time"

	"tlaloc.local/behaviorlab/prototypelab"
)

// This external-package test is intentionally shaped like a downstream
// consumer (for example TONAL): it can use the public API without importing
// any tlaloc/internal package.
func TestPublicConsumerCanWriteBundle(t *testing.T) {
	manifest := prototypelab.RunManifest{
		Schema:           prototypelab.ManifestSchema,
		RunID:            "consumer-run-1",
		SourceExperiment: "CONSUMER_PROTO",
		Prototype: prototypelab.Prototype{
			ID:            "CONSUMER_PROTO",
			Version:       "0.2",
			ParentVersion: "0.1",
			Hypothesis:    "bounded change improves the failing capability",
		},
	}
	episodes := []prototypelab.Episode{{
		Schema:           prototypelab.EpisodeSchema,
		EpisodeID:        "consumer-episode-1",
		SourceExperiment: "CONSUMER_PROTO",
		RunID:            "consumer-run-1",
		TaskID:           "task-1",
		Success:          true,
		SemanticCorrect:  true,
		ExactCorrect:     true,
	}}

	paths, err := prototypelab.WriteBundle(
		t.TempDir(), manifest, episodes,
		time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	if _, err := os.Stat(paths.Summary); err != nil {
		t.Fatalf("summary not written: %v", err)
	}
}
