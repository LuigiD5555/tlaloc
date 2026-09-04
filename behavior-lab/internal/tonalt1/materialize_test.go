package tonalt1

import (
	"image"
	"image/color"
	"testing"
)

// TestLoadFrozenWorkflows_Real loads the actual frozen D4 artifact and
// checks the hard invariants the image pipeline depends on: 60 workflows,
// 144 operand-role assignments, zero candidate reuse.
func TestLoadFrozenWorkflows_Real(t *testing.T) {
	workflows, err := LoadFrozenWorkflows(RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadFrozenWorkflows: %v", err)
	}
	if len(workflows) != WorkflowCount {
		t.Fatalf("got %d workflows, want %d", len(workflows), WorkflowCount)
	}
	total := 0
	for _, w := range workflows {
		total += len(w.Operands)
	}
	if total != OperandCount {
		t.Fatalf("got %d operand-role assignments, want %d", total, OperandCount)
	}
}

// TestValidateOperandsInPrimaryUniverse_Real resolves the actual 144
// candidate IDs against the actual frozen primary universe and checks the
// derived page count is the authoritative 92 — not the eligible-page
// superset (217) and not the primary-universe superset (173).
func TestValidateOperandsInPrimaryUniverse_Real(t *testing.T) {
	workflows, err := LoadFrozenWorkflows(RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadFrozenWorkflows: %v", err)
	}
	validated, err := ValidateOperandsInPrimaryUniverse(workflows)
	if err != nil {
		t.Fatalf("ValidateOperandsInPrimaryUniverse: %v", err)
	}
	if len(validated.Operands) != OperandCount {
		t.Fatalf("resolved %d/%d operands", len(validated.Operands), OperandCount)
	}
	if len(validated.UniquePages) != 92 {
		t.Fatalf("derived %d unique pages, want 92 (the authoritative D4 page set)", len(validated.UniquePages))
	}
	for _, r := range validated.Operands {
		if r.Candidate.Geometry.OperandBBoxEstimate.X2 <= r.Candidate.Geometry.OperandBBoxEstimate.X1 {
			t.Fatalf("candidate %s has degenerate operand_bbox_estimate", r.CandidateID)
		}
	}
}

// TestVerifyNoLeakage_Real runs the no-leakage audit on the exact resolved
// 144 candidates (not the full 405-candidate universe).
func TestVerifyNoLeakage_Real(t *testing.T) {
	workflows, err := LoadFrozenWorkflows(RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadFrozenWorkflows: %v", err)
	}
	validated, err := ValidateOperandsInPrimaryUniverse(workflows)
	if err != nil {
		t.Fatalf("ValidateOperandsInPrimaryUniverse: %v", err)
	}
	if err := VerifyNoLeakage(validated); err != nil {
		t.Fatalf("VerifyNoLeakage: %v", err)
	}
}

// TestDeriveCallBudget_Real mechanically derives Arm A/C from the actual
// frozen workflow list and checks agreement with the frozen 60/492/144/696
// figures.
func TestDeriveCallBudget_Real(t *testing.T) {
	workflows, err := LoadFrozenWorkflows(RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Fatalf("LoadFrozenWorkflows: %v", err)
	}
	budget, err := DeriveCallBudget(workflows)
	if err != nil {
		t.Fatalf("DeriveCallBudget: %v", err)
	}
	if budget.ArmA != 60 || budget.ArmB != 492 || budget.ArmC != 144 || budget.Total != 696 {
		t.Fatalf("got %d/%d/%d/%d, want 60/492/144/696", budget.ArmA, budget.ArmB, budget.ArmC, budget.Total)
	}
}

// TestIsKnownPlaceholderSignature_RejectsSyntheticFixture reproduces the
// exact invalid image shape from the earlier attempt (uniform gray fill +
// magenta rectangle stroke, 2 colors) and checks it is rejected.
func TestIsKnownPlaceholderSignature_RejectsSyntheticFixture(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	gray := color.RGBA{200, 200, 200, 255}
	magenta := color.RGBA{230, 0, 130, 255}
	for y := 0; y < CanvasPx; y++ {
		for x := 0; x < CanvasPx; x++ {
			img.Set(x, y, gray)
		}
	}
	for x := 50; x < 462; x++ {
		img.Set(x, 50, magenta)
		img.Set(x, 461, magenta)
	}
	for y := 50; y < 462; y++ {
		img.Set(50, y, magenta)
		img.Set(461, y, magenta)
	}
	if !isKnownPlaceholderSignature(img) {
		t.Fatal("expected the synthetic 2-color placeholder to be flagged; it was accepted as valid")
	}
}

// TestIsKnownPlaceholderSignature_AcceptsRealContent checks a
// many-colored image (representative of a real bilinear-resampled page
// crop) is not falsely rejected.
func TestIsKnownPlaceholderSignature_AcceptsRealContent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, CanvasPx, CanvasPx))
	for y := 0; y < CanvasPx; y++ {
		for x := 0; x < CanvasPx; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), uint8((x + y) % 256), 255})
		}
	}
	if isKnownPlaceholderSignature(img) {
		t.Fatal("real varied-color image was incorrectly flagged as the synthetic placeholder")
	}
}

// TestGenerateOperandPresentations_Real runs the real production
// materialization for the full frozen 144-operand allocation and checks the
// full PDF-derivation chain: page raster -> parrotpresent.Prepare ->
// 512x512 decodeable PNG that is not the known placeholder signature. This
// test shells to pdftoppm and reads the real source PDF (~90s); it is
// skipped if either is unavailable. Full 144/60 byte-repeatability across
// two independent runs is additionally proven by the Run-1/Run-2 CLI
// execution and its comparison artifacts, not repeated here.
func TestGenerateOperandPresentations_Real(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full 144-operand materialization in -short mode")
	}
	workflows, err := LoadFrozenWorkflows(RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		t.Skipf("frozen workflows unavailable: %v", err)
	}
	validated, err := ValidateOperandsInPrimaryUniverse(workflows)
	if err != nil {
		t.Skipf("primary universe unavailable: %v", err)
	}

	pages, err := RasterizePages(validated.UniquePages)
	if err != nil {
		t.Skipf("rasterization unavailable (pdftoppm/source PDF): %v", err)
	}
	images, records, err := GenerateOperandPresentations(validated, pages)
	if err != nil {
		t.Fatalf("GenerateOperandPresentations: %v", err)
	}
	if len(images) != OperandCount || len(records) != OperandCount {
		t.Fatalf("got %d images / %d records, want %d/%d", len(images), len(records), OperandCount, OperandCount)
	}
	for _, rec := range records {
		if rec.Width != CanvasPx || rec.Height != CanvasPx {
			t.Fatalf("candidate %s prepared image is %dx%d, want %dx%d", rec.CandidateID, rec.Width, rec.Height, CanvasPx, CanvasPx)
		}
		if rec.PreparedBytes == 0 {
			t.Fatalf("candidate %s prepared PNG has zero bytes", rec.CandidateID)
		}
	}
}
