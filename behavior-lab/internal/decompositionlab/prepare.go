package decompositionlab

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "image/png"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

// PREPARE (T0 protocol sections 27-28). Deterministic import + preparation
// of the real frozen P0 / P2-A artifacts into experiment-ready T0 inputs:
// it verifies hashes, builds the corrected P0Records (Goal != AtomicStep,
// multi EvidenceRef), compiles the runtime CapabilityProfile without
// touching the source, runs the frozen T0-B eligibility audit, and records
// provenance for every imported artifact. It runs zero model inference.

const PrepareManifestSchemaR0 = "tlaloc.exocortex-decomposition-t0.r0.prepare-manifest"

// PrepareManifest binds a T0 run to the exact evidence it was prepared from.
type PrepareManifest struct {
	Schema             string               `json:"schema"`
	ExperimentID       string               `json:"experiment_id"`
	P0Baseline         P0BaselineProvenance `json:"p0_baseline"`
	P0ProvenanceSHA256 string               `json:"p0_provenance_sha256"`
	P2AArtifactPath    string               `json:"p2a_artifact_path"`
	P2AArtifactSHA256  string               `json:"p2a_artifact_sha256"`
	P2AExperimentID    string               `json:"p2a_experiment_id"`
	ProfileID          string               `json:"profile_id"`
	ModelID            string               `json:"model_id"`
	MaxSafeOps         int                  `json:"max_safe_ops"`
	T0BDatasetPath     string               `json:"t0b_dataset_path"`
	T0BDatasetSHA256   string               `json:"t0b_dataset_sha256"`
	EligibilityPath    string               `json:"eligibility_path"`
	C0BaselinePath     string               `json:"c0_baseline_path"`
	EligibleR0         int                  `json:"eligible_r0"`
	NotApplicableR0    int                  `json:"not_applicable_r0"`
	R0TaskCoverage     float64              `json:"r0_task_coverage"`
}

// PrepareInput configures a prepare run.
type PrepareInput struct {
	P0ExperimentDir string
	P2AArtifactPath string
	ExecutorID      string
	ModelID         string
	ProfileVersion  string
	OutDir          string
}

func pngDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// BuildT0BRecords turns the frozen P0 provenance into corrected P0Records:
// Goal is the full question (stays in Tlaloc), EvidenceRefs carry the
// page-level ground-truth address plus the required-fact text span (a
// layout locator, never the answer), and Recipe is filled by the
// eligibility audit for ELIGIBLE cases only.
func BuildT0BRecords(provRecords []P0ProvenanceRecord, p0ExperimentDir string) ([]P0Record, error) {
	imageDir := filepath.Join(p0ExperimentDir, "datasets", "end-to-end", "scaffold-images")
	records := make([]P0Record, 0, len(provRecords))
	for _, pr := range provRecords {
		if len(pr.PageRefs) == 0 {
			return nil, fmt.Errorf("P0 record %q has no page_refs", pr.BaseID)
		}
		page := pr.PageRefs[0]
		imagePath := filepath.Join(imageDir, fmt.Sprintf("p%d.png", page))
		w, h, err := pngDimensions(imagePath)
		if err != nil {
			return nil, fmt.Errorf("P0 record %q page image: %w", pr.BaseID, err)
		}
		address := ""
		if len(pr.GTAddresses) > 0 {
			address = pr.GTAddresses[0]
		}
		docID := "fold-bench"
		if address != "" {
			if trimmed := strings.TrimPrefix(address, "ohf://"); trimmed != address {
				if idx := strings.IndexByte(trimmed, '/'); idx > 0 {
					docID = trimmed[:idx]
				}
			}
		}
		var refs []EvidenceRef
		for i, fact := range pr.RequiredFacts {
			refs = append(refs, EvidenceRef{
				ID: fmt.Sprintf("ev%d", i+1), DocID: docID, Page: page,
				Address: address, TextSpan: fact,
			})
		}
		if len(refs) == 0 {
			refs = append(refs, EvidenceRef{ID: "ev1", DocID: docID, Page: page, Address: address})
		}
		records = append(records, P0Record{
			BaseID: pr.BaseID, Goal: pr.Question, Question: pr.Question, ExpectedAnswer: pr.ExpectedAnswer,
			Category: pr.Category, TaskFamily: pr.TaskFamily, DocID: docID, Page: page,
			PageImagePath: imagePath, PageWidth: float64(w), PageHeight: float64(h),
			EvidenceAddress: address, EvidenceRefs: refs,
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].BaseID < records[j].BaseID })
	return records, nil
}

func writeJSON(path string, v any) (string, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return sha256Bytes(body), nil
}

// Prepare executes the full deterministic preparation and writes every T0
// input artifact under OutDir. It returns the manifest it wrote.
func Prepare(in PrepareInput) (PrepareManifest, error) {
	if in.ExecutorID == "" {
		in.ExecutorID = "parrot-lfm2-vl-1.6b"
	}
	if in.ModelID == "" {
		in.ModelID = "lfm2-vl-1.6b"
	}
	if in.ProfileVersion == "" {
		in.ProfileVersion = "r0"
	}

	outcomes, baselineProv, err := ImportP0Baseline(in.P0ExperimentDir)
	if err != nil {
		return PrepareManifest{}, fmt.Errorf("import P0 baseline: %w", err)
	}
	provRecords, provHash, err := LoadP0Provenance(in.P0ExperimentDir)
	if err != nil {
		return PrepareManifest{}, fmt.Errorf("load P0 provenance: %w", err)
	}

	profile, err := exocortex.CompileParrotProfileReal(in.P2AArtifactPath, in.ExecutorID, in.ModelID, in.ProfileVersion)
	if err != nil {
		return PrepareManifest{}, fmt.Errorf("compile P2-A profile: %w", err)
	}
	if err := exocortex.VerifySourceArtifact(profile); err != nil {
		return PrepareManifest{}, err
	}

	records, err := BuildT0BRecords(provRecords, in.P0ExperimentDir)
	if err != nil {
		return PrepareManifest{}, err
	}

	// Fill recipes from the eligibility audit, then re-validate.
	audit := RunEligibilityAudit(records, profile, baselineProv.DatasetSHA256)
	recipeByBase := map[string][]AtomicStep{}
	for _, c := range audit.Cases {
		if c.Eligibility == EligibleR0 {
			recipeByBase[c.BaseID] = c.Recipe
		}
	}
	for i := range records {
		records[i].Recipe = recipeByBase[records[i].BaseID]
		if err := records[i].ValidateRecipe(profile.MaxSafeOps); err != nil {
			return PrepareManifest{}, err
		}
	}

	dataset := Dataset{
		Schema: DatasetSchemaR0, SourceBenchmark: baselineProv.ExperimentID,
		SourceArtifactSHA256: baselineProv.DatasetSHA256, Records: records,
	}

	datasetPath := filepath.Join(in.OutDir, "datasets", "T0_P0_IMAGE_DATASET.json")
	datasetHash, err := writeJSON(datasetPath, dataset)
	if err != nil {
		return PrepareManifest{}, err
	}
	eligPath := filepath.Join(in.OutDir, "results", "T0B_ELIGIBILITY_R0.json")
	if _, err := writeJSON(eligPath, audit); err != nil {
		return PrepareManifest{}, err
	}
	c0Path := filepath.Join(in.OutDir, "results", "C0_P0_BASELINE.json")
	c0Sorted := make([]P0Outcome, 0, len(outcomes))
	for _, id := range SortedP0BaseIDs(outcomes) {
		c0Sorted = append(c0Sorted, outcomes[id])
	}
	if _, err := writeJSON(c0Path, map[string]any{
		"schema": "tlaloc.exocortex-t0b.c0-baseline.r0", "provenance": baselineProv, "outcomes": c0Sorted,
	}); err != nil {
		return PrepareManifest{}, err
	}

	_, p2aHash, err := exocortex.LoadMicroISAArtifactReal(in.P2AArtifactPath)
	if err != nil {
		return PrepareManifest{}, err
	}

	manifest := PrepareManifest{
		Schema: PrepareManifestSchemaR0, ExperimentID: ExperimentID,
		P0Baseline: baselineProv, P0ProvenanceSHA256: provHash,
		P2AArtifactPath: in.P2AArtifactPath, P2AArtifactSHA256: p2aHash, P2AExperimentID: profile.SourceExperiment,
		ProfileID: profile.ProfileID, ModelID: profile.ModelID, MaxSafeOps: profile.MaxSafeOps,
		T0BDatasetPath: datasetPath, T0BDatasetSHA256: datasetHash,
		EligibilityPath: eligPath, C0BaselinePath: c0Path,
		EligibleR0: audit.EligibleR0, NotApplicableR0: audit.NotApplicableR0, R0TaskCoverage: audit.R0TaskCoverage,
	}
	manifestPath := filepath.Join(in.OutDir, "manifest.json")
	if _, err := writeJSON(manifestPath, manifest); err != nil {
		return PrepareManifest{}, err
	}
	return manifest, nil
}
