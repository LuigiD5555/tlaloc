package experimentalspine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/episode"
	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

// LoadFrozenT1Result reconstructs the in-memory subset of RunResult needed by
// the Experimental Spine from T1's three frozen raw JSON artifacts. It makes
// zero model/network calls and does not reinterpret the raw accounting.
func LoadFrozenT1Result(rawDir string) (tonalt1arms.RunResult, error) {
	if rawDir == "" {
		return tonalt1arms.RunResult{}, errors.New("experimentalspine: raw T1 directory is empty")
	}
	var result tonalt1arms.RunResult
	if err := readStrictJSON(filepath.Join(rawDir, "workflow_records.json"), &result.WorkflowRecords); err != nil {
		return tonalt1arms.RunResult{}, err
	}
	if err := readStrictJSON(filepath.Join(rawDir, "node_call_records.json"), &result.NodeRecords); err != nil {
		return tonalt1arms.RunResult{}, err
	}
	if err := readStrictJSON(filepath.Join(rawDir, "run_accounting.json"), &result.Accounting); err != nil {
		return tonalt1arms.RunResult{}, err
	}
	return result, nil
}

// MinimalT1Manifest derives only provenance that is actually present in the
// frozen T1 result. Repository SHAs, hypothesis, endpoint and timestamps stay
// empty unless the caller explicitly supplies an enriched manifest.
func MinimalT1Manifest(result tonalt1arms.RunResult) (RunManifest, error) {
	if len(result.WorkflowRecords) == 0 {
		return RunManifest{}, errors.New("experimentalspine: cannot derive T1 manifest from zero workflow records")
	}
	runID := result.WorkflowRecords[0].RunID
	if runID == "" {
		return RunManifest{}, errors.New("experimentalspine: first T1 workflow has empty run_id")
	}
	for _, wf := range result.WorkflowRecords[1:] {
		if wf.RunID != runID {
			return RunManifest{}, fmt.Errorf("experimentalspine: inconsistent T1 run ids: %q and %q", runID, wf.RunID)
		}
	}

	models := map[string]struct{}{}
	for _, rec := range result.NodeRecords {
		if rec.Model != "" {
			models[rec.Model] = struct{}{}
		}
	}
	reportedModel := ""
	if len(models) == 1 {
		for model := range models {
			reportedModel = model
		}
	}

	return RunManifest{
		Schema:           ManifestSchema,
		RunID:            runID,
		SourceExperiment: episode.SourceT1,
		Prototype: Prototype{
			ID:      "TONAL_T1",
			Version: "T1",
		},
		Model: Model{Reported: reportedModel},
	}, nil
}

func LoadManifest(path string) (RunManifest, error) {
	var manifest RunManifest
	if err := readStrictJSON(path, &manifest); err != nil {
		return RunManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return RunManifest{}, err
	}
	return manifest, nil
}

func readStrictJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("experimentalspine: read %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("experimentalspine: decode %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("experimentalspine: %s contains trailing JSON", path)
	}
	return nil
}
