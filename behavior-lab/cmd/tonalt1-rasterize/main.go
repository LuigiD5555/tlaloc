// Command tonalt1-rasterize materializes the frozen TONAL T1 image
// pipeline: it loads the frozen D4 workflow allocation (60 workflows, 144
// operand-role assignments), resolves every candidate against the frozen
// fresh-corpus primary universe, rasterizes exactly the pages those 144
// operands require, renders each operand through the real production
// parrotpresent.Prepare, and builds the 60 Arm-A composites. Zero model
// calls; zero selection; zero synthetic imagery.
package main

import (
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

	fmt.Printf("TONAL T1 Image Materialization Pipeline\n")
	fmt.Printf("Run: %s\n", *runDir)
	fmt.Printf("======================================\n\n")

	// Phase 1: Load frozen D4 workflows verbatim.
	fmt.Printf("[1/9] Loading frozen D4 workflows...\n")
	workflows, err := tonalt1.LoadFrozenWorkflows(tonalt1.RepoPath("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Loaded %d workflows, %d operand-role assignments\n\n", len(workflows), tonalt1.OperandCount)

	// Phase 2: Resolve every candidate_id against the frozen primary
	// universe and derive the authoritative page set from those 144
	// candidates only.
	fmt.Printf("[2/9] Resolving 144 candidate IDs against frozen primary universe...\n")
	validated, err := tonalt1.ValidateOperandsInPrimaryUniverse(workflows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Resolved %d/%d candidates, derived %d unique pages\n\n", len(validated.Operands), tonalt1.OperandCount, len(validated.UniquePages))

	// Phase 3: Rasterize exactly the derived page set, fresh, at 150 DPI.
	fmt.Printf("[3/9] Rasterizing %d pages fresh from source PDF at 150 DPI...\n", len(validated.UniquePages))
	pages, err := tonalt1.RasterizePages(validated.UniquePages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Rasterized %d/%d pages\n\n", len(pages), len(validated.UniquePages))

	// Phase 4: Real operand presentations via parrotpresent.Prepare.
	fmt.Printf("[4/9] Generating 144 operand presentations via parrotpresent.Prepare...\n")
	operandImages, operandRecords, err := tonalt1.GenerateOperandPresentations(validated, pages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Generated %d/%d operand images (512×512, real PDF-derived pixels)\n\n", len(operandImages), tonalt1.OperandCount)

	// Phase 5: Arm-A composites (real pixel stacking, no rescale).
	fmt.Printf("[5/9] Generating 60 Arm-A composites (vertical stack, zero rescale)...\n")
	composites, compositeRecords, err := tonalt1.GenerateArmAComposites(workflows, operandImages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Generated %d/%d composites\n\n", len(composites), tonalt1.WorkflowCount)

	// Phase 6: Write outputs.
	fmt.Printf("[6/9] Writing outputs to %s...\n", *runDir)
	if err := os.MkdirAll(*runDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	for key, img := range operandImages {
		safe := sanitizeKey(key)
		path := filepath.Join(*runDir, fmt.Sprintf("operand_%s.png", safe))
		if err := os.WriteFile(path, img, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL write operand: %v\n", err)
			os.Exit(1)
		}
	}
	for workflowID, img := range composites {
		path := filepath.Join(*runDir, fmt.Sprintf("composite_%s.png", workflowID))
		if err := os.WriteFile(path, img, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL write composite: %v\n", err)
			os.Exit(1)
		}
	}
	writeJSON(filepath.Join(*runDir, "operand_manifest.json"), map[string]any{"count": len(operandRecords), "records": operandRecords})
	writeJSON(filepath.Join(*runDir, "composite_manifest.json"), map[string]any{"count": len(compositeRecords), "records": compositeRecords})
	writeJSON(filepath.Join(*runDir, "page_manifest.json"), map[string]any{"count": len(pages), "pages": pages})
	fmt.Printf("✓ Wrote %d operand images, %d composites, manifests\n\n", len(operandImages), len(composites))

	// Phase 7: No-leakage audit on the exact resolved 144.
	fmt.Printf("[7/9] Verifying no-leakage on the exact resolved 144 candidates...\n")
	if err := tonalt1.VerifyNoLeakage(validated); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ No-leakage verified\n\n")

	// Phase 8: Call budget, mechanically derived.
	fmt.Printf("[8/9] Deriving call budget from actual frozen workflows...\n")
	budget, err := tonalt1.DeriveCallBudget(workflows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Call budget: %d/%d/%d/%d\n\n", budget.ArmA, budget.ArmB, budget.ArmC, budget.Total)

	// Phase 9: Summary. Transport fail-closed tests are run separately via
	// `go test ./internal/target/... -run TestT1Transport_`, not shelled
	// out from this command.
	fmt.Printf("[9/9] Pipeline complete\n")
	fmt.Printf("======================================\n")
	fmt.Printf("Pages:       %d\n", len(pages))
	fmt.Printf("Operands:    %d / %d\n", len(operandImages), tonalt1.OperandCount)
	fmt.Printf("Composites:  %d / %d\n", len(composites), tonalt1.WorkflowCount)
	fmt.Printf("Call budget: %d/%d/%d/%d\n", budget.ArmA, budget.ArmB, budget.ArmC, budget.Total)
	fmt.Printf("T1_PRIMARY_MODEL_CALLS = 0\n")
	fmt.Printf("Run output: %s\n", *runDir)
	fmt.Printf("======================================\n")
}

func sanitizeKey(key string) string {
	out := make([]rune, 0, len(key))
	for _, r := range key {
		if r == '|' || r == '/' || r == ' ' {
			out = append(out, '_')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL marshal %s: %v\n", path, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL write %s: %v\n", path, err)
		os.Exit(1)
	}
}
