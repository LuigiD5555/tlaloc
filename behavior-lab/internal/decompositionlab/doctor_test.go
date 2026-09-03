package decompositionlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

func writeValidDataset(t *testing.T) (path string) {
	t.Helper()
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "deadbeef", Records: validRecords(30)}
	return writeDataset(t, dataset)
}

func writeFrozenMicroISA(t *testing.T) string {
	t.Helper()
	artifact := exocortex.MicroISAArtifact{
		Schema: exocortex.MicroISAArtifactSchemaR0, ExperimentID: "parrot-microisa-r0.1", Records: 666, Frozen: true,
		MaxSafeOpsSemantic: 1, MaxSafeOpsContract: 1,
		Opcodes: map[string]exocortex.MicroISAOpcodeFinding{
			"EXTRACT_NUMBER": {IntrinsicVerdict: exocortex.VerdictStrong, TightCropAccuracy: floatPtrT(0.88), FullPageAccuracy: floatPtrT(0.5), PDFTransferVerdict: exocortex.TransferPartial},
		},
	}
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "PARROT_MICRO_ISA_R0.json")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestFreeze_HashVerifiesBothArtifacts(t *testing.T) {
	datasetPath := writeValidDataset(t)
	artifactPath := writeFrozenMicroISA(t)
	spec := Spec{DatasetPath: datasetPath, MicroISAArtifactPath: artifactPath, ExecutorID: "parrot-lfm2-vl-1.6b", ModelID: "lfm2-vl-1.6b", ProfileVersion: "r0"}

	manifest, dataset, err := Freeze(spec)
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if manifest.DatasetRecordCount != 30 || len(dataset.Records) != 30 {
		t.Fatalf("unexpected record count: %+v", manifest)
	}
	if manifest.MicroISAExperimentID != "parrot-microisa-r0.1" {
		t.Fatalf("microisa_experiment_id = %q", manifest.MicroISAExperimentID)
	}
	if manifest.DatasetSHA256 == "" || manifest.MicroISAArtifactSHA256 == "" {
		t.Fatalf("expected non-empty hashes in manifest: %+v", manifest)
	}
}

func TestFreeze_RejectsMissingDatasetPath(t *testing.T) {
	if _, _, err := Freeze(Spec{MicroISAArtifactPath: "x"}); err == nil {
		t.Fatalf("expected an error when dataset_path is missing")
	}
}

func TestDoctor_ReportsNotReadyWithoutEndpoint(t *testing.T) {
	datasetPath := writeValidDataset(t)
	artifactPath := writeFrozenMicroISA(t)
	spec := Spec{DatasetPath: datasetPath, MicroISAArtifactPath: artifactPath, ExecutorID: "parrot-lfm2-vl-1.6b", ModelID: "lfm2-vl-1.6b", ProfileVersion: "r0"}

	result, err := Doctor(context.Background(), spec)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if result.Ready {
		t.Fatalf("expected Ready=false with no endpoint configured")
	}
	if len(result.Profile.Capabilities) == 0 {
		t.Fatalf("expected a compiled profile even without an endpoint")
	}
}

func TestDoctor_ReadyWhenEndpointServesConfiguredModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"lfm2-vl-1.6b"}]}`)
	}))
	defer server.Close()

	datasetPath := writeValidDataset(t)
	artifactPath := writeFrozenMicroISA(t)
	spec := Spec{DatasetPath: datasetPath, MicroISAArtifactPath: artifactPath, ExecutorID: "parrot-lfm2-vl-1.6b", ModelID: "lfm2-vl-1.6b", ProfileVersion: "r0", Endpoint: server.URL}

	result, err := Doctor(context.Background(), spec)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if !result.Ready {
		t.Fatalf("expected Ready=true, got %+v", result)
	}
}

func TestDoctor_NotReadyWhenModelMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"some-other-model"}]}`)
	}))
	defer server.Close()

	datasetPath := writeValidDataset(t)
	artifactPath := writeFrozenMicroISA(t)
	spec := Spec{DatasetPath: datasetPath, MicroISAArtifactPath: artifactPath, ExecutorID: "parrot-lfm2-vl-1.6b", ModelID: "lfm2-vl-1.6b", ProfileVersion: "r0", Endpoint: server.URL}

	result, err := Doctor(context.Background(), spec)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if result.Ready {
		t.Fatalf("expected Ready=false when the endpoint does not serve the configured model")
	}
	if len(result.Reasons) == 0 {
		t.Fatalf("expected a reason explaining why doctor is not ready")
	}
}
