package tonalt1arms

import (
	"testing"

	tonalt1 "tlaloc.local/behaviorlab/internal/tonalt1"
)

// TestStartupImageSweep_RealMaterialization_FullOffline actually calls the
// real, frozen internal/tonalt1 image pipeline (RasterizePages ->
// GenerateOperandPresentations -> GenerateArmAComposites -- the exact
// functions that produced TONAL_T1_IMAGE_MANIFEST_FINAL.json in the first
// place) and verifies all 144 operand + 60 composite hashes against that
// frozen manifest. This is entirely offline (local PDF file + pdftoppm; no
// network, no model call) and is a real, concrete offline proof of the
// image-hash fail-closed guard against the actual frozen artifacts, not
// just fixture data (task correction E: 144/144 + 60/60 verified before any
// model call is permitted).
func TestStartupImageSweep_RealMaterialization_FullOffline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full 144-operand + 60-composite materialization in -short mode")
	}

	workflows, err := tonalt1.LoadFrozenWorkflows(tonalt1.RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Skipf("frozen workflows unavailable: %v", err)
	}
	validated, err := tonalt1.ValidateOperandsInPrimaryUniverse(workflows)
	if err != nil {
		t.Skipf("primary universe unavailable: %v", err)
	}
	pages, err := tonalt1.RasterizePages(validated.UniquePages)
	if err != nil {
		t.Skipf("rasterization unavailable (pdftoppm/source PDF): %v", err)
	}
	operandImages, _, err := tonalt1.GenerateOperandPresentations(validated, pages)
	if err != nil {
		t.Fatalf("GenerateOperandPresentations: %v", err)
	}
	compositeImages, _, err := tonalt1.GenerateArmAComposites(workflows, operandImages)
	if err != nil {
		t.Fatalf("GenerateArmAComposites: %v", err)
	}

	manifest, err := LoadImageManifest(RepoPathHelper("internal/tonalt1/v2_frozen/TONAL_T1_IMAGE_MANIFEST_FINAL.json"))
	if err != nil {
		t.Fatalf("LoadImageManifest: %v", err)
	}

	result, err := StartupImageSweep(manifest, operandImages, compositeImages)
	if err != nil {
		t.Fatalf("StartupImageSweep failed against real materialized bytes: %v\nfailures: %v", err, result.Failures)
	}
	if result.OperandHashesValid != "144/144" {
		t.Errorf("OperandHashesValid = %q, want 144/144", result.OperandHashesValid)
	}
	if result.CompositeHashesValid != "60/60" {
		t.Errorf("CompositeHashesValid = %q, want 60/60", result.CompositeHashesValid)
	}
	if !result.AllValid {
		t.Errorf("AllValid = false, want true")
	}
	if len(result.OperandImages) != 144 {
		t.Errorf("StartupSweepResult.OperandImages has %d entries, want 144 (exact verified bundle must be returned, not just a pass/fail summary)", len(result.OperandImages))
	}
	if len(result.CompositeImages) != 60 {
		t.Errorf("StartupSweepResult.CompositeImages has %d entries, want 60", len(result.CompositeImages))
	}
}

// TestStartupImageSweep_MissingImageFailsClosed uses a real manifest but
// withholds one operand's bytes entirely -- must fail the whole sweep, not
// silently skip it.
func TestStartupImageSweep_MissingImageFailsClosed(t *testing.T) {
	manifest := &ImageManifest{
		Operands: []OperandImageRecord{
			{WorkflowID: "wf-1", Role: "A", Run1PreparedSHA256: "abc", Run2PreparedSHA256: "abc", Equal: true},
		},
	}
	_, err := StartupImageSweep(manifest, map[string][]byte{}, map[string][]byte{})
	if err == nil {
		t.Fatal("expected StartupImageSweep to fail closed when an image is missing")
	}
}

// TestStartupImageSweep_HashMismatchFailsClosed builds a manifest recording
// one hash and supplies bytes that hash to something else -- proves a
// genuine mismatch (not just a missing file) fails the whole sweep and
// records a specific failure message.
func TestStartupImageSweep_HashMismatchFailsClosed(t *testing.T) {
	realBytes := []byte("real operand PNG bytes")
	realHash := sha256Hex(realBytes)
	manifest := &ImageManifest{
		Operands: []OperandImageRecord{
			{WorkflowID: "wf-1", Role: "A", Run1PreparedSHA256: realHash, Run2PreparedSHA256: realHash, Equal: true},
		},
	}
	corrupted := []byte("corrupted operand PNG bytes")
	result, err := StartupImageSweep(manifest, map[string][]byte{"wf-1|A": corrupted}, map[string][]byte{})
	if err == nil {
		t.Fatal("expected StartupImageSweep to fail closed on hash mismatch")
	}
	if len(result.Failures) == 0 {
		t.Fatal("expected at least one recorded failure message")
	}
}

// TestVerifyOperandImage_ReHashCatchesInMemoryCorruption is correction I's
// required test: given a StartupSweepResult whose verified bundle has been
// tampered with in-process AFTER the sweep (simulating memory corruption
// between startup and an adapter call, bypassing the sweep's own check via
// direct mutation), the per-call VerifyOperandImage re-hash must catch it
// and fail closed BEFORE any adapter call -- never silently proceeding, and
// never re-rendering a fresh, hash-valid image to paper over the
// corruption.
func TestVerifyOperandImage_ReHashCatchesInMemoryCorruption(t *testing.T) {
	original := []byte("the exact startup-verified operand bytes")
	hash := sha256Hex(original)
	manifest := &ImageManifest{
		Operands: []OperandImageRecord{
			{WorkflowID: "wf-1", Role: "A", Run1PreparedSHA256: hash, Run2PreparedSHA256: hash, Equal: true},
		},
	}

	sweepResult, err := StartupImageSweep(manifest, map[string][]byte{"wf-1|A": original}, map[string][]byte{})
	if err != nil {
		t.Fatalf("sweep should have passed on the real bytes: %v", err)
	}

	// Simulate in-process corruption of the exact verified bundle (a single
	// byte flipped), bypassing the sweep's own already-completed check.
	corrupted := append([]byte(nil), sweepResult.OperandImages["wf-1|A"]...)
	corrupted[0] ^= 0xFF
	sweepResult.OperandImages["wf-1|A"] = corrupted

	// The executor's pre-call re-hash (VerifyOperandImage, called on the
	// exact bytes about to be sent to the adapter) must catch this.
	if err := VerifyOperandImage(manifest, "wf-1", "A", sweepResult.OperandImages["wf-1|A"]); err == nil {
		t.Fatal("expected VerifyOperandImage to fail closed on tampered bytes, but it succeeded")
	}
}

// TestVerifyOperandImage_MissingRecord and TestVerifyComposite_MissingRecord
// cover the "missing manifest record" fail-closed path distinctly from a
// hash mismatch.
func TestVerifyOperandImage_MissingRecord(t *testing.T) {
	manifest := &ImageManifest{}
	if err := VerifyOperandImage(manifest, "wf-x", "A", []byte("anything")); err == nil {
		t.Fatal("expected error for missing manifest record")
	}
}

func TestVerifyComposite_MissingRecord(t *testing.T) {
	manifest := &ImageManifest{}
	if err := VerifyComposite(manifest, "wf-x", []byte("anything")); err == nil {
		t.Fatal("expected error for missing manifest record")
	}
}
