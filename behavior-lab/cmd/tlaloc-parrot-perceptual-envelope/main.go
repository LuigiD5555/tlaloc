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
	case "scale-audit":
		scaleAudit(os.Args[2:])
	case "run-diagnostic":
		runDiagnostic(ctx, os.Args[2:])
	case "prepare-r1a1":
		prepareR1A1(os.Args[2:])
	case "sanity-r1a1":
		sanityR1A1(os.Args[2:])
	case "run-r1a1":
		runR1A1(ctx, os.Args[2:])
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

func scaleAudit(args []string) {
	fs := flag.NewFlagSet("scale-audit", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	fs.Parse(args)
	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	rep := perceptenvelope.ScaleAudit(alloc.Bases, filepath.Join(*expDir, "runs", "r1a-r0", "pages"))
	sha, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "SCALE_AUDIT_R1A.json"), rep)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{"scale_audit_sha256": sha, "vision_preprocessing": rep.VisionPreproc, "bases": len(rep.Bases)}))
}

func runDiagnostic(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-diagnostic", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	nbases := fs.Int("bases", 6, "number of predeclared diagnostic bases (first N of R1A_BASES)")
	runID := fs.String("run-id", "r1a-diagnostic-r0", "run id")
	fs.Parse(args)

	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	bases := perceptenvelope.DiagnosticBases(alloc, *nbases)
	levels := []perceptenvelope.ContextLevel{
		perceptenvelope.A0TargetOnly, perceptenvelope.A2LocalBlock,
		perceptenvelope.A4QuarterPage, perceptenvelope.A6FullPage,
	}
	records, err := perceptenvelope.RunScaleConfoundDiagnostic(ctx, perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: filepath.Join(*expDir, "runs", *runID),
	}, bases, levels)
	die(err)
	recSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A_DIAGNOSTIC_RECORDS.json"), records)
	die(err)
	cmp := perceptenvelope.AggregateDiagnostic(records, levels)
	cmpSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A_SCALE_CONFOUND_DIAGNOSTIC.json"), cmp)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"records": len(records), "records_sha256": recSHA, "diagnostic_sha256": cmpSHA,
		"natural_crop_semantic_by_level":          cmp.NaturalCurve,
		"fixed_canvas_semantic_by_level":          cmp.FixedCurve,
		"fixed_minus_natural_by_level":            cmp.PerLevelDelta,
		"max_abs_delta":                           cmp.MaxAbsDelta,
		"CURRENT_R1A_CONTEXT_IS_SCALE_CONFOUNDED": cmp.Confounded,
		"confound_note":                           cmp.ConfoundNote,
	}))
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

func prepareR1A1(args []string) {
	fs := flag.NewFlagSet("prepare-r1a1", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	fs.Parse(args)

	poolBody, err := os.ReadFile(filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json"))
	die(err)
	var pool perceptenvelope.SourcePool
	die(json.Unmarshal(poolBody, &pool))
	poolSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json"))

	r1a0 := loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	r1b := loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))
	exclude := map[string]struct{}{}
	for _, b := range r1a0.Bases {
		exclude[b.Candidate.CandidateID] = struct{}{}
	}
	for _, b := range r1b.Bases {
		exclude[b.Candidate.CandidateID] = struct{}{}
	}
	r1a1 := perceptenvelope.AllocateR1A1(pool, exclude, poolSHA)
	r1a1SHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1A1_BASES.json"), r1a1)
	die(err)

	audit := perceptenvelope.AuditR1A1Geometry(*storeDir, r1a1, r1a0, r1b)
	_, err = perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1A1_CONTEXT_DATASET.json"), audit.PerBase)
	die(err)
	auditSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A1_GEOMETRY_AUDIT.json"), audit)
	die(err)

	manifest := map[string]any{
		"schema":                     "tlaloc.parrot-perceptual-envelope-r1.r1a1-prepare-manifest.r1",
		"experiment_id":              perceptenvelope.ExperimentID,
		"source_pool_sha256":         poolSHA,
		"r1a1_bases_sha256":          r1a1SHA,
		"r1a1_geometry_audit_sha256": auditSHA,
		"r1a1_base_ids":              r1a1.BaseIDs,
		"canvas_px":                  perceptenvelope.CanvasPx,
		"target_line_height_px":      perceptenvelope.TargetLineHeightPx,
		"ready_r1a1_geometry":        audit.ReadyR1A1Geom,
	}
	mSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "manifests", "R1A1_PREPARE_MANIFEST.json"), manifest)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"r1a1_bases": len(r1a1.Bases), "r1a1_base_ids": r1a1.BaseIDs,
		"ready_r1a1_geometry": audit.ReadyR1A1Geom, "problems": audit.Problems,
		"checks": audit.Checks, "manifest_sha256": mSHA,
	}))
	if !audit.ReadyR1A1Geom {
		os.Exit(1)
	}
}

func runR1A1(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-r1a1", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	runID := fs.String("run-id", "r1a1-r0", "run id")
	fs.Parse(args)

	audBody, err := os.ReadFile(filepath.Join(*expDir, "results", "R1A1_GEOMETRY_AUDIT.json"))
	die(err)
	var aud perceptenvelope.R1A1GeometryAudit
	die(json.Unmarshal(audBody, &aud))
	if !aud.ReadyR1A1Geom {
		die(fmt.Errorf("R1A1_GEOMETRY_AUDIT.ready_r1a1_geometry is false; run prepare-r1a1 first"))
	}

	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1A1_BASES.json"))
	runDir := filepath.Join(*expDir, "runs", *runID)
	records, geos, err := perceptenvelope.RunR1A1Context(ctx, perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: runDir,
	}, alloc)
	die(err)

	recSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A1_RECORDS.json"), records)
	die(err)
	_, err = perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A1_GEOMETRY_FROZEN.json"), geos)
	die(err)
	curve := perceptenvelope.AggregateR1A1Curve(records)
	curveSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1A1_CONTEXT_CURVE.json"), curve)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"records": len(records), "records_sha256": recSHA, "curve_sha256": curveSHA, "curve": curve,
	}))
}

func sanityR1A1(args []string) {
	fs := flag.NewFlagSet("sanity-r1a1", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	n := fs.Int("bases", 3, "bases to render")
	fs.Parse(args)
	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1A1_BASES.json"))
	bases := alloc.Bases
	if *n < len(bases) {
		bases = bases[:*n]
	}
	outDir := filepath.Join(*expDir, "runs", "r1a1-sanity")
	geos, err := perceptenvelope.RenderR1A1Sanity(*storeDir, *pdfPath, bases, outDir)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{"rendered_bases": len(geos), "out_dir": outDir, "geometry": geos}))
}
