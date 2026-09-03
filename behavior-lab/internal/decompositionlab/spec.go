package decompositionlab

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

const (
	// SpecSchemaR0 identifies the T0 experiment spec (section 14:
	// exocortex-decomposition-r0, ONE_OP_DECOMPOSITION_R0).
	SpecSchemaR0     = "tlaloc.exocortex-decomposition-t0.r0.spec"
	ManifestSchemaR0 = "tlaloc.exocortex-decomposition-t0.r0.manifest"
	ExperimentID     = "exocortex-decomposition-r0"
)

// Spec is T0's own MANIFEST/SPEC input (section 14): every path is a
// reference to an existing, frozen artifact. Spec never embeds P0/P1/P2-A
// content — only paths and, once loaded, their hashes.
type Spec struct {
	Schema string `json:"schema"`

	DatasetPath          string `json:"dataset_path"`           // the 30 frozen P0 image records (section 16)
	MicroISAArtifactPath string `json:"microisa_artifact_path"` // frozen P2-A results/PARROT_MICRO_ISA_R0.json
	ExecutorID           string `json:"executor_id"`
	ModelID              string `json:"model_id"`
	ProfileVersion       string `json:"profile_version"`

	Endpoint        string  `json:"endpoint"`
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"max_output_tokens"`
	MarginRatio     float64 `json:"margin_ratio"`

	PDFMemoryStoreDir string `json:"pdfmemory_store_dir,omitempty"` // required only for T0-B REAL conditions

	OutputDir string `json:"output_dir"`
}

// Manifest binds Spec to the concrete, hash-verified evidence it read
// (section 14, mirroring the realcampaign.Manifest provenance pattern —
// not duplicating it, T0 has its own distinct artifact set).
type Manifest struct {
	Schema                 string `json:"schema"`
	ExperimentID           string `json:"experiment_id"`
	DatasetPath            string `json:"dataset_path"`
	DatasetSHA256          string `json:"dataset_sha256"`
	DatasetRecordCount     int    `json:"dataset_record_count"`
	MicroISAArtifactPath   string `json:"microisa_artifact_path"`
	MicroISAArtifactSHA256 string `json:"microisa_artifact_sha256"`
	MicroISAExperimentID   string `json:"microisa_experiment_id"`
	ProfileID              string `json:"profile_id"`
	Endpoint               string `json:"endpoint"`
	ModelID                string `json:"model_id"`
}

// Freeze loads and hash-verifies every referenced artifact and produces the
// Manifest that binds this T0 run to them. It never modifies P0 or P2-A;
// it only reads and hashes them.
func Freeze(spec Spec) (Manifest, Dataset, error) {
	if strings.TrimSpace(spec.DatasetPath) == "" || strings.TrimSpace(spec.MicroISAArtifactPath) == "" {
		return Manifest{}, Dataset{}, fmt.Errorf("dataset_path and microisa_artifact_path are required")
	}
	dataset, datasetHash, err := LoadDataset(spec.DatasetPath)
	if err != nil {
		return Manifest{}, Dataset{}, err
	}
	artifactHash, artifactExperiment, err := loadMicroISAHash(spec.MicroISAArtifactPath)
	if err != nil {
		return Manifest{}, Dataset{}, err
	}
	manifest := Manifest{
		Schema: ManifestSchemaR0, ExperimentID: ExperimentID,
		DatasetPath: spec.DatasetPath, DatasetSHA256: datasetHash, DatasetRecordCount: len(dataset.Records),
		MicroISAArtifactPath: spec.MicroISAArtifactPath, MicroISAArtifactSHA256: artifactHash, MicroISAExperimentID: artifactExperiment,
		ProfileID: spec.ExecutorID + "@" + spec.ProfileVersion,
		Endpoint:  spec.Endpoint, ModelID: spec.ModelID,
	}
	return manifest, dataset, nil
}

// loadMicroISAHash hash-verifies the frozen P2-A artifact and returns its
// hash + experiment id. It accepts either the real P2-A schema
// (parrot-microisa-r0.1) or the SYNTHETIC_TEST_FIXTURE schema used by unit
// tests, so `Freeze`/`doctor`/`run` work against both without a second
// codepath at every call site (E0.15: reuse the exocortex loaders).
func loadMicroISAHash(path string) (hash, experimentID string, err error) {
	if real, h, rerr := exocortex.LoadMicroISAArtifactReal(path); rerr == nil {
		return h, real.ExperimentID, nil
	}
	art, h, serr := exocortex.LoadMicroISAArtifact(path)
	if serr != nil {
		return "", "", fmt.Errorf("micro-ISA artifact %s is neither the real P2-A schema nor the synthetic fixture schema: %w", path, serr)
	}
	return h, art.ExperimentID, nil
}

// CompileProfileFlexible compiles a runtime CapabilityProfile from either
// the real P2-A artifact or the synthetic fixture artifact.
func CompileProfileFlexible(artifactPath, executorID, modelID, profileVersion string) (exocortex.CapabilityProfile, error) {
	if profile, err := exocortex.CompileParrotProfileReal(artifactPath, executorID, modelID, profileVersion); err == nil {
		return profile, nil
	}
	return exocortex.CompileParrotProfile(artifactPath, executorID, modelID, profileVersion)
}

// WriteManifest persists the freeze manifest as canonical indented JSON.
func WriteManifest(path string, manifest Manifest) error {
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}
