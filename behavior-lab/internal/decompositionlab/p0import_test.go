package decompositionlab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportP0Baseline_ImageVariantsPairedByBaseID(t *testing.T) {
	skipIfNoFrozen(t)
	outcomes, prov, err := ImportP0Baseline(testP0Dir)
	if err != nil {
		t.Fatalf("ImportP0Baseline: %v", err)
	}
	if prov.DatasetSHA256 == "" || prov.DatasetSHA256 != prov.DatasetFreezeHash {
		t.Fatalf("dataset hash not verified against FREEZE.json: on-disk %q frozen %q", prov.DatasetSHA256, prov.DatasetFreezeHash)
	}
	if len(outcomes) != prov.ImageRecords || len(outcomes) == 0 {
		t.Fatalf("imported %d outcomes, provenance says %d", len(outcomes), prov.ImageRecords)
	}
	for id, o := range outcomes {
		if o.BaseID != id {
			t.Fatalf("outcome keyed %q but base_id %q", id, o.BaseID)
		}
		if o.Category == "" {
			t.Fatalf("outcome %q has no category (provenance join failed)", id)
		}
	}
}

func TestImportP0Baseline_RejectsTamperedDataset(t *testing.T) {
	skipIfNoFrozen(t)
	// Copy the frozen experiment, tamper the dataset, expect a hash failure.
	dir := t.TempDir()
	copyTree(t, testP0Dir, dir)
	dsPath := filepath.Join(dir, "datasets", "end-to-end.jsonl")
	body, _ := os.ReadFile(dsPath)
	if err := os.WriteFile(dsPath, append(body, []byte("\n{}\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ImportP0Baseline(dir); err == nil {
		t.Fatalf("expected a dataset hash mismatch after tampering")
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyTree(t, s, d)
			continue
		}
		body, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(d, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
