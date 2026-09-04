// Command tlaloc-tonalt1-select runs the deterministic TONAL T1 D3
// selector: it scans the canonical 1152-page store, excludes every
// prior-used physical instance, applies the frozen R1 perceptual-envelope
// and geometry rules, and freezes the held-out operand universe.
//
// No model. No scorer. No expected-answer feedback. No manual selection.
//
//	tlaloc-tonalt1-select \
//	  -root . \
//	  -store experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0 \
//	  -out  experiments/tonal-t1/d3
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/tonalt1"
)

func main() {
	root := flag.String("root", ".", "behavior-lab repository root (holds experiments/)")
	store := flag.String("store", "experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0", "canonical store directory (relative to root or absolute)")
	out := flag.String("out", "experiments/tonal-t1/d3", "output directory for frozen artifacts (relative to root)")
	verifyTwice := flag.Bool("verify-twice", true, "run the pipeline twice and assert identical freeze hashes")
	flag.Parse()

	storeDir := *store
	if !isAbs(storeDir) {
		storeDir = join(*root, storeDir)
	}
	outDir := join(*root, *out)

	first, err := tonalt1.RunD3(*root, storeDir)
	if err != nil {
		fail(err)
	}

	if *verifyTwice {
		second, err := tonalt1.RunD3(*root, storeDir)
		if err != nil {
			fail(err)
		}
		if first.Freeze.ArtifactHashes[tonalt1.FileCandidatesAll] != second.Freeze.ArtifactHashes[tonalt1.FileCandidatesAll] ||
			hashOf(first.Freeze) != hashOf(second.Freeze) {
			fail(fmt.Errorf("DETERMINISM VIOLATION: freeze hashes differ between runs"))
		}
	}

	if err := first.Write(outDir); err != nil {
		fail(err)
	}

	report := map[string]any{
		"TONAL_T1_D3_FROZEN":               first.Freeze.TONALT1D3Frozen,
		"out_dir":                          outDir,
		"scan_total":                       first.Stats.ScanTotal,
		"pages_scanned":                    first.Stats.PagesScanned,
		"regions_scanned":                  first.Stats.RegionsScanned,
		"prior_physical_identity_excluded": first.Stats.PriorPhysicalIdentityExcluded,
		"r1_envelope_rejected":             first.Stats.R1EnvelopeRejected,
		"geometry_rejected":                first.Stats.GeometryRejected,
		"final_held_out_available":         first.Stats.FinalHeldOutAvailable,
		"required_unique_operand_demand":   first.Stats.RequiredUniqueOperandDemand,
		"headroom_ratio":                   first.Stats.HeadroomRatio,
		"downstream_allocation_feasible":   first.Stats.AllocationFeasible,
		"hard_invariants":                  first.Freeze.HardInvariants,
		"exclusions_by_experiment":         first.Stats.ExclusionsByExperiment,
		"exclusions_by_key":                first.Stats.ExclusionsByKey,
		"total_prior_instances":            first.Inventory.TotalInstances,
		"freeze_artifact_hashes":           first.Freeze.ArtifactHashes,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
}

func hashOf(value any) string {
	data, _ := json.Marshal(value)
	sum := 0
	for _, b := range data {
		sum = sum*31 + int(b)
	}
	return fmt.Sprintf("%x", sum)
}

func isAbs(path string) bool { return len(path) > 0 && path[0] == '/' }

func join(base, rest string) string {
	if isAbs(rest) {
		return rest
	}
	if base == "" || base == "." {
		return rest
	}
	if base[len(base)-1] == '/' {
		return base + rest
	}
	return base + "/" + rest
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tlaloc-tonalt1-select:", err)
	os.Exit(1)
}
