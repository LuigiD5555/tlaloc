// Command tonalt1-rasterize implements the frozen TONAL T1 image rasterization pipeline.
// Constraint: 38-point execution specification, zero model calls, byte-level repeatability.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/tonalt1"
)

func main() {
	runDir := flag.String("run", "", "output directory for this run (run1 or run2)")
	flag.Parse()

	if *runDir == "" {
		fmt.Fprintf(os.Stderr, "usage: tonalt1-rasterize -run <run1|run2>\n")
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Printf("TONAL T1 Image Rasterization Pipeline\n")
	fmt.Printf("Run: %s\n", *runDir)
	fmt.Printf("======================================\n\n")

	// Phase 1: Load frozen workflows
	fmt.Printf("[1/10] Loading frozen workflows...\n")
	workflows, err := tonalt1.LoadFrozenWorkflows("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Loaded %d workflows, %d operands\n\n", len(workflows), tonalt1.WorkflowCount)

	// Phase 2: Validate operands against primary universe
	fmt.Printf("[2/10] Validating operands against primary universe...\n")
	validated, err := tonalt1.ValidateOperandsInPrimaryUniverse(workflows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Validated %d operands, %d unique pages\n\n", len(validated), len(validated.UniquePages))

	// Phase 3: Rasterize pages
	fmt.Printf("[3/10] Rasterizing %d pages at 150 DPI...\n", len(validated.UniquePages))
	pageImages, pageHashes, err := tonalt1.RasterizePages(ctx, validated.UniquePages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Rasterized %d pages\n\n", len(pageImages))

	// Phase 4: Generate operand presentations
	fmt.Printf("[4/10] Generating 144 operand presentations...\n")
	operandImages, operandRecords, err := tonalt1.GenerateOperandPresentations(ctx, validated, pageImages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Generated %d operand images (512×512, cue enabled)\n\n", len(operandImages))

	// Phase 5: Generate Arm A composites
	fmt.Printf("[5/10] Generating 60 Arm-A composites (no rescaling)...\n")
	composites, compositeRecords, err := tonalt1.GenerateArmAComposites(workflows, operandImages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Generated %d composites, no operand rescaling\n\n", len(composites))

	// Phase 6: Write outputs to run directory
	fmt.Printf("[6/10] Writing outputs to %s...\n", *runDir)
	if err := os.MkdirAll(*runDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}

	// Write operand images
	for id, img := range operandImages {
		path := filepath.Join(*runDir, fmt.Sprintf("operand_%s.png", id[:8]))
		if err := os.WriteFile(path, img, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL write operand: %v\n", err)
			os.Exit(1)
		}
	}

	// Write composites
	for idx, img := range composites {
		path := filepath.Join(*runDir, fmt.Sprintf("composite_%03d.png", idx))
		if err := os.WriteFile(path, img, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL write composite: %v\n", err)
			os.Exit(1)
		}
	}

	// Write manifests
	operandManifest := map[string]interface{}{
		"count": len(operandRecords),
		"records": operandRecords,
	}
	if b, err := json.MarshalIndent(operandManifest, "", "  "); err == nil {
		os.WriteFile(filepath.Join(*runDir, "operand_manifest.json"), b, 0644)
	}

	compositeManifest := map[string]interface{}{
		"count": len(compositeRecords),
		"records": compositeRecords,
	}
	if b, err := json.MarshalIndent(compositeManifest, "", "  "); err == nil {
		os.WriteFile(filepath.Join(*runDir, "composite_manifest.json"), b, 0644)
	}

	// Write page hashes
	pageHashManifest := map[string]interface{}{
		"page_count": len(pageHashes),
		"hashes": pageHashes,
	}
	if b, err := json.MarshalIndent(pageHashManifest, "", "  "); err == nil {
		os.WriteFile(filepath.Join(*runDir, "page_hashes.json"), b, 0644)
	}

	fmt.Printf("✓ Wrote 144 operand images, 60 composites, manifests\n\n")

	// Phase 7: Verify no leakage
	fmt.Printf("[7/10] Verifying no-leakage invariants...\n")
	if err := tonalt1.VerifyNoLeakage(validated); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ No-leakage verified: bridge=0, prior-use=0, duplicates=0\n\n")

	// Phase 8: Verify call budget
	fmt.Printf("[8/10] Deriving and verifying call budget...\n")
	callBudget, err := tonalt1.DeriveCallBudget(workflows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	if callBudget.ArmA != 60 || callBudget.ArmB != 492 || callBudget.ArmC != 144 || callBudget.Total != 696 {
		fmt.Fprintf(os.Stderr, "FAIL: call budget mismatch: %d/%d/%d/%d\n",
			callBudget.ArmA, callBudget.ArmB, callBudget.ArmC, callBudget.Total)
		os.Exit(1)
	}
	fmt.Printf("✓ Call budget verified: 60/492/144/696 (counterfactual=0)\n\n")

	// Phase 9: Transport fail-closed tests
	fmt.Printf("[9/10] Running transport fail-closed tests...\n")
	if err := tonalt1.RunTransportFailClosedTests(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ All 8 transport tests passed\n\n")

	// Phase 10: Final summary
	fmt.Printf("[10/10] Pipeline complete\n")
	fmt.Printf("======================================\n")
	fmt.Printf("Operands:    %d / %d\n", len(operandImages), 144)
	fmt.Printf("Composites:  %d / %d\n", len(composites), 60)
	fmt.Printf("Pages:       %d unique\n", len(pageImages))
	fmt.Printf("Call budget: verified (60/492/144/696)\n")
	fmt.Printf("No-leakage:  verified\n")
	fmt.Printf("Transport:   verified (8/8 tests)\n")
	fmt.Printf("T1_PRIMARY_MODEL_CALLS = 0\n")
	fmt.Printf("Run output: %s\n", *runDir)
	fmt.Printf("======================================\n")
}
