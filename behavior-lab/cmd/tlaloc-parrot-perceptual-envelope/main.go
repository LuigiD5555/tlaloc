// Command tlaloc-parrot-perceptual-envelope drives the Parrot Perceptual
// Envelope R1 experiment. Subcommands:
//
//	prepare   scan the store, build + freeze SOURCE_POOL_R1 / R1A_BASES /
//	          R1B_BASES / R1C_POOL / R1D_POOL and the manifest
//	doctor    run every pre-inference integrity check -> READY_R1A
//	run-r1a   execute R1-A (30 bases x 7 context levels = 210 calls),
//	          aggregate the context curve
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"tlaloc.local/behaviorlab/internal/perceptenvelope"
)

const defaultExpDir = "experiments/parrot-perceptual-envelope-r1"
const defaultStore = "experiments/exocortex-decomposition-r0/stores/fold-bench-reconstructed-r0"

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tlaloc-parrot-perceptual-envelope <prepare|doctor|run-r1a> [flags]")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	switch os.Args[1] {
	case "prepare":
		prepare(os.Args[2:])
	case "doctor":
		doctor(ctx, os.Args[2:])
	case "run-r1a":
		runR1A(ctx, os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand:", os.Args[1])
		os.Exit(2)
	}
}

func prepare(args []string) {
	fs := flag.NewFlagSet("prepare", flag.ExitOnError)
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	fs.Parse(args)

	pool, err := perceptenvelope.ScanSourcePool(*storeDir)
	die(err)
	poolPath := filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json")
	poolSHA, err := perceptenvelope.WriteJSON(poolPath, pool)
	die(err)

	r1a, r1b := perceptenvelope.Allocate(pool, poolSHA)
	r1aSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1A_BASES.json"), r1a)
	die(err)
	r1bSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1B_BASES.json"), r1b)
	die(err)

	excluded := map[string]struct{}{}
	for _, b := range r1a.Bases {
		excluded[b.Candidate.CandidateID] = struct{}{}
	}
	for _, b := range r1b.Bases {
		excluded[b.Candidate.CandidateID] = struct{}{}
	}
	morph, err := perceptenvelope.ScanMorphologyPool(*storeDir, excluded)
	die(err)
	morphSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1C_POOL.json"), morph)
	die(err)
	lv, err := perceptenvelope.ScanLabelValuePool(*storeDir, excluded)
	die(err)
	lvSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1D_POOL.json"), lv)
	die(err)

	manifest := map[string]any{
		"schema":                      "tlaloc.parrot-perceptual-envelope-r1.prepare-manifest.r1",
		"experiment_id":               perceptenvelope.ExperimentID,
		"store_dir":                   *storeDir,
		"source_pdf_sha256":           pool.SourcePDFSHA256,
		"store_root_sha256":           pool.StoreRootSHA256,
		"selection_algorithm_version": perceptenvelope.SelectionAlgorithmVersion,
		"seed":                        perceptenvelope.Seed,
		"pages_scanned":               pool.PagesScanned,
		"regions_scanned":             pool.RegionsScanned,
		"digit_tokens_seen":           pool.DigitTokensSeen,
		"primary_candidate_count":     pool.PrimaryCandidates,
		"rejection_counts":            pool.RejectionCounts,
		"artifacts": map[string]string{
			"SOURCE_POOL_R1.json": poolSHA,
			"R1A_BASES.json":      r1aSHA,
			"R1B_BASES.json":      r1bSHA,
			"R1C_POOL.json":       morphSHA,
			"R1D_POOL.json":       lvSHA,
		},
		"r1a_base_ids":         r1a.BaseIDs,
		"r1b_base_ids":         r1b.BaseIDs,
		"r1c_family_available": morph.FamilyAvailable,
		"r1d_candidate_count":  lv.Count,
	}
	mSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "manifests", "PREPARE_MANIFEST_R1.json"), manifest)
	die(err)

	summary := map[string]any{
		"primary_candidates":   pool.PrimaryCandidates,
		"pages_scanned":        pool.PagesScanned,
		"rejection_counts":     pool.RejectionCounts,
		"r1a_bases":            len(r1a.Bases),
		"r1b_bases":            len(r1b.Bases),
		"r1c_family_available": morph.FamilyAvailable,
		"r1d_candidates":       lv.Count,
		"source_pool_sha256":   poolSHA,
		"manifest_sha256":      mSHA,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(summary))
}

func loadAlloc(path string) perceptenvelope.Allocation {
	body, err := os.ReadFile(path)
	die(err)
	var a perceptenvelope.Allocation
	die(json.Unmarshal(body, &a))
	return a
}

func doctor(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	fs.Parse(args)

	poolBody, err := os.ReadFile(filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json"))
	die(err)
	var pool perceptenvelope.SourcePool
	die(json.Unmarshal(poolBody, &pool))
	poolSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json"))
	r1aSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	r1bSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))

	rep := perceptenvelope.Doctor(ctx, perceptenvelope.DoctorInput{
		ModelIdentityPath: filepath.Join(*expDir, "MODEL_IDENTITY.json"),
		Endpoint:          *endpoint, Model: *model,
		SourcePool: pool, SourcePoolSHA: poolSHA,
		R1A:    loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json")),
		R1B:    loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json")),
		R1ASHA: r1aSHA, R1BSHA: r1bSHA,
	})
	_, err = perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A_DOCTOR.json"), rep)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(rep))
	if !rep.ReadyR1A {
		os.Exit(1)
	}
}

func runR1A(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-r1a", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override (default: store's own object)")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	runID := fs.String("run-id", "r1a-r0", "run id")
	fs.Parse(args)

	// re-gate
	doctor(ctx, []string{"-exp-dir", *expDir, "-endpoint", *endpoint, "-model", *model})

	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	runDir := filepath.Join(*expDir, "runs", *runID)
	records, err := perceptenvelope.RunContextEnvelope(ctx, perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: runDir,
	}, alloc)
	die(err)

	recSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A_RECORDS.json"), records)
	die(err)
	curve := perceptenvelope.AggregateContextCurve(records)
	curveSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A_CONTEXT_CURVE.json"), curve)
	die(err)

	out := map[string]any{
		"records":                  len(records),
		"r1a_records_sha256":       recSHA,
		"r1a_context_curve_sha256": curveSHA,
		"curve":                    curve,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(out))
}
