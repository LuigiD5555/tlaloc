package experimentalspine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tlaloc.local/behaviorlab/internal/episode"
	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

// BundlePaths identifies the immutable common projection written beside an
// experiment's native/raw evidence.
type BundlePaths struct {
	Root         string `json:"root"`
	Manifest     string `json:"manifest"`
	EpisodesRoot string `json:"episodes_root"`
	Summary      string `json:"summary"`
	Episodes     int    `json:"episodes"`
}

// WriteBundle writes the generic Experimental Spine bundle for any prototype
// that can produce Episodes. The target is <outDir>/experience. Existing
// bundles are never overwritten.
func WriteBundle(outDir string, manifest RunManifest, episodes []episode.Episode, observedAt time.Time) (BundlePaths, error) {
	if err := validateBundleInput(outDir, manifest, episodes, observedAt); err != nil {
		return BundlePaths{}, err
	}
	return writePreparedBundle(outDir, manifest, episodes, Summarize(manifest, episodes), observedAt)
}

// FreezePrimaryT1Run is the intended T1 primary-run persistence boundary:
// first freeze the experiment-native scientific records, then immediately
// write the common experience projection beside them. It never changes T1's
// frozen raw formats. If projection fails, the raw freeze remains available
// for repair/backfill without repeating model calls.
func FreezePrimaryT1Run(outDir string, manifest RunManifest, result tonalt1arms.RunResult, observedAt time.Time) (BundlePaths, error) {
	if err := result.Freeze(outDir); err != nil {
		return BundlePaths{}, err
	}
	return WriteT1Bundle(outDir, manifest, result, observedAt)
}

// WriteT1Bundle adapts a T1 CrossArmRunner result into the same common bundle
// while preserving T1's frozen raw records as the source of truth. It also
// proves that the Episode projection reproduces T1's dynamic accounting before
// writing anything. This function can be used to backfill experience from an
// already-frozen in-memory T1 result without refreezing raw evidence.
func WriteT1Bundle(outDir string, manifest RunManifest, result tonalt1arms.RunResult, observedAt time.Time) (BundlePaths, error) {
	if manifest.SourceExperiment != episode.SourceT1 {
		return BundlePaths{}, fmt.Errorf("experimentalspine: T1 source_experiment = %q, want %q", manifest.SourceExperiment, episode.SourceT1)
	}
	if len(result.WorkflowRecords) == 0 {
		return BundlePaths{}, errors.New("experimentalspine: T1 result has no workflow records")
	}
	for _, wf := range result.WorkflowRecords {
		if wf.RunID != manifest.RunID {
			return BundlePaths{}, fmt.Errorf("experimentalspine: T1 workflow run_id = %q, manifest run_id = %q", wf.RunID, manifest.RunID)
		}
	}

	episodes := episode.FromT1RunResult(result)
	if err := validateBundleInput(outDir, manifest, episodes, observedAt); err != nil {
		return BundlePaths{}, err
	}

	summary := Summarize(manifest, episodes)
	summary.Cost.PlannedModelCallSlots = result.Accounting.PlannedModelCallSlots
	if err := validateT1Projection(summary, result.Accounting); err != nil {
		return BundlePaths{}, err
	}
	return writePreparedBundle(outDir, manifest, episodes, summary, observedAt)
}

func validateBundleInput(outDir string, manifest RunManifest, episodes []episode.Episode, observedAt time.Time) error {
	if outDir == "" {
		return errors.New("experimentalspine: output directory is empty")
	}
	if observedAt.IsZero() {
		return errors.New("experimentalspine: observedAt is zero; caller must supply the run observation time")
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	if len(episodes) == 0 {
		return errors.New("experimentalspine: refusing to publish an experience bundle with zero episodes")
	}
	for index, ep := range episodes {
		if ep.Schema != episode.Schema {
			return fmt.Errorf("experimentalspine: episode[%d] schema = %q, want %q", index, ep.Schema, episode.Schema)
		}
		if ep.SourceExperiment != "" && ep.SourceExperiment != manifest.SourceExperiment {
			return fmt.Errorf("experimentalspine: episode[%d] source_experiment = %q, manifest = %q", index, ep.SourceExperiment, manifest.SourceExperiment)
		}
		if ep.RunID != "" && ep.RunID != manifest.RunID {
			return fmt.Errorf("experimentalspine: episode[%d] run_id = %q, manifest = %q", index, ep.RunID, manifest.RunID)
		}
	}
	return nil
}

func validateT1Projection(summary Summary, accounting tonalt1arms.RunAccounting) error {
	type check struct {
		name string
		got  int
		want int
	}
	checks := []check{
		{"http_request_attempts", summary.Cost.HTTPRequestAttempts, accounting.HTTPRequestAttempts},
		{"valid_completions", summary.Cost.ValidCompletions, accounting.ValidCompletions},
		{"transport_failures", summary.Cost.TransportFailures, accounting.TransportFailures},
		{"schema_failures", summary.Cost.SchemaFailures, accounting.SchemaFailures},
		{"model_contract_failures", summary.Cost.ModelContractFailures, accounting.ModelContractFailures},
		{"blocked_by_dependency", summary.Cost.BlockedByDependency, accounting.BlockedByDependency},
	}
	for _, item := range checks {
		if item.got != item.want {
			return fmt.Errorf("experimentalspine: T1 projection accounting mismatch for %s: episodes=%d raw=%d", item.name, item.got, item.want)
		}
	}
	return nil
}

func writePreparedBundle(outDir string, manifest RunManifest, episodes []episode.Episode, summary Summary, observedAt time.Time) (BundlePaths, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return BundlePaths{}, err
	}
	finalRoot := filepath.Join(outDir, "experience")
	if _, err := os.Stat(finalRoot); err == nil {
		return BundlePaths{}, fmt.Errorf("experimentalspine: immutable experience bundle already exists: %s", finalRoot)
	} else if !errors.Is(err, os.ErrNotExist) {
		return BundlePaths{}, err
	}

	tempRoot, err := os.MkdirTemp(outDir, ".experience-tmp-*")
	if err != nil {
		return BundlePaths{}, err
	}
	defer os.RemoveAll(tempRoot)

	manifestPath := filepath.Join(tempRoot, "manifest.json")
	if err := writeJSONExclusive(manifestPath, manifest); err != nil {
		return BundlePaths{}, err
	}
	episodesRoot := filepath.Join(tempRoot, "episodes")
	if _, err := episode.StoreAll(episodesRoot, episodes, observedAt); err != nil {
		return BundlePaths{}, err
	}
	summaryPath := filepath.Join(tempRoot, "summary.json")
	if err := writeJSONExclusive(summaryPath, summary); err != nil {
		return BundlePaths{}, err
	}

	if err := os.Rename(tempRoot, finalRoot); err != nil {
		return BundlePaths{}, fmt.Errorf("experimentalspine: publish bundle: %w", err)
	}

	return BundlePaths{
		Root:         finalRoot,
		Manifest:     filepath.Join(finalRoot, "manifest.json"),
		EpisodesRoot: filepath.Join(finalRoot, "episodes"),
		Summary:      filepath.Join(finalRoot, "summary.json"),
		Episodes:     len(episodes),
	}, nil
}

func writeJSONExclusive(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
