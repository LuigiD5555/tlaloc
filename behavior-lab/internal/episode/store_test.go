package episode

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_RoundTrip(t *testing.T) {
	root := t.TempDir()
	observedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	ep := Episode{
		Schema:           Schema,
		EpisodeID:        "t1-run-1-wf-001-B",
		SourceExperiment: SourceT1,
		TaskID:           "wf-001/B",
		Success:          true,
	}

	path, err := Store(root, ep, observedAt)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	wantPath := filepath.Join(root, "2026-09", "t1-run-1-wf-001-B.json")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.EpisodeID != ep.EpisodeID || got.Schema != ep.Schema ||
		got.SourceExperiment != ep.SourceExperiment || got.TaskID != ep.TaskID ||
		got.Success != ep.Success {
		t.Errorf("Load(Store(ep)) = %+v, want %+v", got, ep)
	}
}

func TestStore_RefusesToOverwrite(t *testing.T) {
	root := t.TempDir()
	observedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	ep := Episode{Schema: Schema, EpisodeID: "dup-episode"}

	if _, err := Store(root, ep, observedAt); err != nil {
		t.Fatalf("first Store: %v", err)
	}
	if _, err := Store(root, ep, observedAt); err == nil {
		t.Error("second Store with same EpisodeID: got nil error, want an error (must not overwrite)")
	}
}

func TestStoreAll_WritesOnePerEpisode(t *testing.T) {
	root := t.TempDir()
	observedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	episodes := []Episode{
		{Schema: Schema, EpisodeID: "e1"},
		{Schema: Schema, EpisodeID: "e2"},
	}

	paths, err := StoreAll(root, episodes, observedAt)
	if err != nil {
		t.Fatalf("StoreAll: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("len(paths) = %d, want 2", len(paths))
	}
}
