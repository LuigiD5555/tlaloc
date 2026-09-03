package decompositionlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func validRecords(n int) []P0Record {
	cats := []string{CategoryLocate, CategoryEntity, CategoryFactual, CategoryNumeric, CategorySynthesis}
	records := make([]P0Record, 0, n)
	for i := 0; i < n; i++ {
		records = append(records, P0Record{
			BaseID: fmt.Sprintf("p0-img-%03d", i), Question: "what is shown", ExpectedAnswer: "42",
			Category: cats[i%len(cats)], DocID: "doc1", Page: i + 1, PageImagePath: fmt.Sprintf("pages/%03d.png", i),
			EvidenceAddress: fmt.Sprintf("ohf://carrier/docs/doc1/pages/%06d/regions/0001", i+1),
			Opcode:          "EXTRACT_NUMBER",
		})
	}
	return records
}

func writeDataset(t *testing.T, dataset Dataset) string {
	t.Helper()
	body, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		t.Fatalf("marshal dataset: %v", err)
	}
	path := filepath.Join(t.TempDir(), "t0_dataset.json")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write dataset: %v", err)
	}
	return path
}

func TestLoadDataset_AcceptsExactly30ValidRecords(t *testing.T) {
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "abc123", Records: validRecords(30)}
	path := writeDataset(t, dataset)
	body, _ := os.ReadFile(path)
	sum := sha256.Sum256(body)
	wantHash := hex.EncodeToString(sum[:])

	loaded, hash, err := LoadDataset(path)
	if err != nil {
		t.Fatalf("LoadDataset: %v", err)
	}
	if len(loaded.Records) != 30 {
		t.Fatalf("loaded %d records, want 30", len(loaded.Records))
	}
	if hash != wantHash {
		t.Fatalf("hash = %s, want %s", hash, wantHash)
	}
}

func TestLoadDataset_RejectsWrongRecordCount(t *testing.T) {
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "abc123", Records: validRecords(29)}
	path := writeDataset(t, dataset)
	if _, _, err := LoadDataset(path); err == nil {
		t.Fatalf("expected an error for 29 records, want exactly 30")
	}
}

func TestLoadDataset_RejectsDuplicateBaseID(t *testing.T) {
	records := validRecords(30)
	records[1].BaseID = records[0].BaseID
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "abc123", Records: records}
	path := writeDataset(t, dataset)
	if _, _, err := LoadDataset(path); err == nil {
		t.Fatalf("expected an error for duplicate base_id")
	}
}

func TestLoadDataset_RejectsMissingEvidenceAddress(t *testing.T) {
	records := validRecords(30)
	records[5].EvidenceAddress = ""
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "abc123", Records: records}
	path := writeDataset(t, dataset)
	if _, _, err := LoadDataset(path); err == nil {
		t.Fatalf("expected an error for a missing evidence_address")
	}
}

func TestLoadDataset_RejectsUnknownCategory(t *testing.T) {
	records := validRecords(30)
	records[0].Category = "not-a-real-category"
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "abc123", Records: records}
	path := writeDataset(t, dataset)
	if _, _, err := LoadDataset(path); err == nil {
		t.Fatalf("expected an error for an unknown category")
	}
}

func TestLoadDataset_RejectsMissingSourceHash(t *testing.T) {
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", Records: validRecords(30)}
	path := writeDataset(t, dataset)
	if _, _, err := LoadDataset(path); err == nil {
		t.Fatalf("expected an error when source_artifact_sha256 (P0 hash reference) is missing")
	}
}

func TestDataset_CategoryCountsAndSortedBaseIDs(t *testing.T) {
	dataset := Dataset{Schema: DatasetSchemaR0, SourceBenchmark: "P0", SourceArtifactSHA256: "abc123", Records: validRecords(30)}
	counts := dataset.CategoryCounts()
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 30 {
		t.Fatalf("category counts sum to %d, want 30", total)
	}
	ids := dataset.SortedBaseIDs()
	if len(ids) != 30 || ids[0] != "p0-img-000" {
		t.Fatalf("unexpected sorted ids: %v", ids)
	}
}
