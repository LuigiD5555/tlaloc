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
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
		fmt.Fprintln(os.Stderr, "usage: tlaloc-parrot-perceptual-envelope <prepare|doctor|run-r1a|prepare-r1a1|run-r1a1|prepare-r1b|sanity-r1b|doctor-r1b|run-r1b> [flags]")
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
	case "prepare-r1b":
		prepareR1B(os.Args[2:])
	case "sanity-r1b":
		sanityR1B(os.Args[2:])
	case "doctor-r1b":
		doctorR1B(ctx, os.Args[2:])
	case "run-r1b":
		runR1B(ctx, os.Args[2:])
	case "report-r1b":
		reportR1B(os.Args[2:])
	case "prepare-r1c":
		prepareR1C(os.Args[2:])
	case "glyphbank-r1c":
		glyphbankR1C(os.Args[2:])
	case "sanity-r1c":
		sanityR1C(os.Args[2:])
	case "doctor-r1c":
		doctorR1C(ctx, os.Args[2:])
	case "run-r1c":
		runR1C(ctx, os.Args[2:])
	case "report-r1c":
		reportR1C(os.Args[2:])
	case "prepare-r1d":
		prepareR1D(os.Args[2:])
	case "sanity-r1d":
		sanityR1D(os.Args[2:])
	case "doctor-r1d":
		doctorR1D(ctx, os.Args[2:])
	case "run-r1d":
		runR1D(ctx, os.Args[2:])
	case "report-r1d":
		reportR1D(os.Args[2:])
	case "prepare-r1e":
		prepareR1E(os.Args[2:])
	case "doctor-r1e":
		doctorR1E(ctx, os.Args[2:])
	case "run-r1e":
		runR1E(ctx, os.Args[2:])
	case "report-r1e":
		reportR1E(os.Args[2:])
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

// ---------------------------------------------------------------------------
// R1-B — SCALE / RESOLUTION ENVELOPE
// ---------------------------------------------------------------------------

func gitCommitShort() string {
	out, err := exec.Command("/usr/bin/git", "-C", ".", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func prepareR1B(args []string) {
	fs := flag.NewFlagSet("prepare-r1b", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	fs.Parse(args)

	r1a0 := loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	r1a1 := loadAlloc(filepath.Join(*expDir, "datasets", "R1A1_BASES.json"))
	r1b := loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))

	audit := perceptenvelope.AuditR1BGeometry(*storeDir, r1b, r1a0, r1a1)
	_, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1B_SCALE_DATASET.json"), audit.PerBase)
	die(err)
	auditSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1B_GEOMETRY_AUDIT.json"), audit)
	die(err)

	basesSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))
	policySHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "R1B_CONTEXT_POLICY.json"))
	identitySHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "MODEL_IDENTITY.json"))
	r1a1CurveSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "results", "R1A1_CONTEXT_CURVE.json"))

	ladder := make([]map[string]any, len(perceptenvelope.R1BScaleLadder))
	for i, cond := range perceptenvelope.R1BScaleLadder {
		ladder[i] = map[string]any{"id": cond.ID, "target_line_height_px": cond.LinePx}
	}
	addendum := map[string]any{
		"schema":      "tlaloc.parrot-perceptual-envelope-r1.protocol-addendum.03",
		"title":       "R1-B_SCALE_ENVELOPE — physical submitted-pixel scale ladder",
		"recorded_at": time.Now().UTC().Format(time.RFC3339),
		"authored":    "BEFORE any R1-B model output existed. R1-A0 and R1-A1 raw outputs, curves and checkpoints are immutable and untouched.",
		"NO_R1B_MODEL_OUTPUT_EXISTED_WHEN_SCALE_LADDER_WAS_DEFINED": true,
		"independent_variable":     "TARGET CONTAINING-LINE HEIGHT IN SUBMITTED PIXELS",
		"scale_ladder":             ladder,
		"context_policy":           "A1C0_TARGET (frozen R1B_CONTEXT_POLICY.json " + short8(policySHA) + ")",
		"canvas":                   "512x512 RGB, RGB(200,200,200) neutral background, target centre (256,256) for every condition",
		"source_crop":              "R1-A1 A1C0_TARGET reveal region (cue token bbox + 10 canvas px / 32 px-scale pad), expressed in store units; the SAME store rectangle is resampled for every B0..B5 — no extra context revealed at larger scale",
		"resampler":                perceptenvelope.R1BResampler,
		"no_image_enhancement":     "no sharpen/denoise/threshold/OCR/super-resolution/contrast alteration",
		"cue_rule":                 "cue thickness / nominal line-height held constant (R1-A1: 3 px at 32 px); stroke = max(1, round(3 * LinePx/32))",
		"line_height_tolerance_px": perceptenvelope.R1BLineHeightTolerancePx,
		"expected_records":         perceptenvelope.R1BExpectedRecords,
		"instruction":              perceptenvelope.FrozenInstruction,
		"opcode":                   perceptenvelope.FrozenOpcode,
		"inputs_hashed": map[string]string{
			"R1B_BASES.json":          basesSHA,
			"R1B_CONTEXT_POLICY.json": policySHA,
			"MODEL_IDENTITY.json":     identitySHA,
			"R1A1_CONTEXT_CURVE.json": r1a1CurveSHA,
			"R1B_GEOMETRY_AUDIT.json": auditSHA,
		},
	}
	addSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "R1_PROTOCOL_ADDENDUM_03.json"), addendum)
	die(err)

	manifest := map[string]any{
		"schema":                      "tlaloc.parrot-perceptual-envelope-r1.r1b-prepare-manifest.r1",
		"experiment_id":               perceptenvelope.ExperimentID,
		"r1b_bases_sha256":            basesSHA,
		"r1b_context_policy_sha256":   policySHA,
		"model_identity_sha256":       identitySHA,
		"r1a1_context_curve_sha256":   r1a1CurveSHA,
		"r1b_geometry_audit_sha256":   auditSHA,
		"protocol_addendum_03_sha256": addSHA,
		"scale_ladder_px":             []float64{8, 12, 16, 24, 32, 48},
		"canvas_px":                   512,
		"expected_records":            perceptenvelope.R1BExpectedRecords,
		"ready_r1b_geometry":          audit.ReadyR1BGeometry,
	}
	mSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "manifests", "R1B_PREPARE_MANIFEST.json"), manifest)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"r1b_bases": len(r1b.Bases), "expected_records": perceptenvelope.R1BExpectedRecords,
		"ready_r1b_geometry": audit.ReadyR1BGeometry, "problems": audit.Problems,
		"checks": audit.Checks, "manifest_sha256": mSHA, "addendum_03_sha256": addSHA,
	}))
	if !audit.ReadyR1BGeometry {
		os.Exit(1)
	}
}

func sanityR1B(args []string) {
	fs := flag.NewFlagSet("sanity-r1b", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	n := fs.Int("bases", 3, "bases to render")
	fs.Parse(args)
	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))
	bases := alloc.Bases
	if *n < len(bases) {
		bases = bases[:*n]
	}
	outDir := filepath.Join(*expDir, "runs", "r1b-sanity")
	geos, err := perceptenvelope.RenderR1BSanity(*storeDir, *pdfPath, bases, outDir)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{"rendered_bases": len(geos), "out_dir": outDir, "geometry": geos}))
}

func doctorR1B(ctx context.Context, args []string) perceptenvelope.DoctorReport {
	fs := flag.NewFlagSet("doctor-r1b", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	noExit := fs.Bool("no-exit", false, "do not exit on not-ready")
	fs.Parse(args)

	poolBody, err := os.ReadFile(filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json"))
	die(err)
	var pool perceptenvelope.SourcePool
	die(json.Unmarshal(poolBody, &pool))
	poolSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "SOURCE_POOL_R1.json"))
	r1aSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1A_BASES.json"))
	r1bSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))

	base := perceptenvelope.Doctor(ctx, perceptenvelope.DoctorInput{
		ModelIdentityPath: filepath.Join(*expDir, "MODEL_IDENTITY.json"),
		Endpoint:          *endpoint, Model: *model,
		SourcePool: pool, SourcePoolSHA: poolSHA,
		R1A:    loadAlloc(filepath.Join(*expDir, "datasets", "R1A_BASES.json")),
		R1B:    loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json")),
		R1ASHA: r1aSHA, R1BSHA: r1bSHA,
	})

	var geomAudit perceptenvelope.R1BGeometryAudit
	if b, rerr := os.ReadFile(filepath.Join(*expDir, "results", "R1B_GEOMETRY_AUDIT.json")); rerr == nil {
		_ = json.Unmarshal(b, &geomAudit)
	}
	ready := base.ModelIdentityOK && base.EndpointReachable && base.EndpointModelListed &&
		base.Disjoint && geomAudit.ReadyR1BGeometry

	report := map[string]any{
		"schema":                "tlaloc.parrot-perceptual-envelope-r1.r1b-doctor.r1",
		"experiment_id":         perceptenvelope.ExperimentID,
		"model_identity_ok":     base.ModelIdentityOK,
		"endpoint_reachable":    base.EndpointReachable,
		"endpoint_model_listed": base.EndpointModelListed,
		"r1b_bases_sha256":      r1bSHA,
		"r1b_base_count":        len(loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json")).Bases),
		"r1a_r1b_disjoint":      base.Disjoint,
		"r1b_geometry_ready":    geomAudit.ReadyR1BGeometry,
		"r1b_geometry_problems": geomAudit.Problems,
		"expected_records":      perceptenvelope.R1BExpectedRecords,
		"model_facing_opcode":   perceptenvelope.FrozenOpcode,
		"READY_R1B":             ready,
	}
	_, werr := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1B_DOCTOR.json"), report)
	die(werr)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(report))
	if !ready && !*noExit {
		os.Exit(1)
	}
	return base
}

func runR1B(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-r1b", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	runID := fs.String("run-id", "r1b-r0", "run id")
	fs.Parse(args)

	// re-gate
	doctorR1B(ctx, []string{"-exp-dir", *expDir, "-endpoint", *endpoint, "-model", *model})

	alloc := loadAlloc(filepath.Join(*expDir, "datasets", "R1B_BASES.json"))
	runDir := filepath.Join(*expDir, "runs", *runID)
	records, geos, err := perceptenvelope.RunR1BScale(ctx, perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: runDir,
	}, alloc)
	die(err)
	finalizeR1B(*expDir, runDir, *model, records, geos)
}

// reportR1B re-aggregates an existing R1-B run (no inference) and rewrites
// the curve, report and checkpoint.
func reportR1B(args []string) {
	fs := flag.NewFlagSet("report-r1b", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	runID := fs.String("run-id", "r1b-r0", "run id")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	fs.Parse(args)
	var records []perceptenvelope.RecordOutcome
	rb, err := os.ReadFile(filepath.Join(*expDir, "results", "R1B_RECORDS.json"))
	die(err)
	die(json.Unmarshal(rb, &records))
	var geos []perceptenvelope.R1BGeometry
	gb, err := os.ReadFile(filepath.Join(*expDir, "results", "R1B_GEOMETRY_FROZEN.json"))
	die(err)
	die(json.Unmarshal(gb, &geos))
	finalizeR1B(*expDir, filepath.Join(*expDir, "runs", *runID), *model, records, geos)
}

func finalizeR1B(expDir, runDir, model string, records []perceptenvelope.RecordOutcome, geos []perceptenvelope.R1BGeometry) {
	recSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1B_RECORDS.json"), records)
	die(err)
	_, err = perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1B_GEOMETRY_FROZEN.json"), geos)
	die(err)
	curve := perceptenvelope.AggregateR1BScaleCurve(records, geos)
	curveSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1B_SCALE_CURVE.json"), curve)
	die(err)

	rawTreeSHA, rawFiles, err := perceptenvelope.SHA256OfTree(filepath.Join(runDir, "raw"))
	die(err)

	var audit perceptenvelope.R1BGeometryAudit
	ab, _ := os.ReadFile(filepath.Join(expDir, "results", "R1B_GEOMETRY_AUDIT.json"))
	_ = json.Unmarshal(ab, &audit)
	basesSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1B_BASES.json"))
	datasetSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1B_SCALE_DATASET.json"))
	auditSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "results", "R1B_GEOMETRY_AUDIT.json"))
	addSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "R1_PROTOCOL_ADDENDUM_03.json"))
	commit := gitCommitShort()

	report := perceptenvelope.RenderR1BReport(perceptenvelope.R1BReportInput{
		Curve: curve, Audit: audit, BasesSHA: basesSHA, DatasetSHA: datasetSHA,
		AuditSHA: auditSHA, AddendumSHA: addSHA, RecordsSHA: recSHA, CurveSHA: curveSHA,
		RawTreeSHA: rawTreeSHA, TlalocCommit: commit, Model: model,
	})
	die(os.WriteFile(filepath.Join(expDir, "results", "R1B_REPORT.md"), []byte(report), 0o644))

	checkpoint := map[string]any{
		"schema":        "tlaloc.parrot-perceptual-envelope-r1.r1b-checkpoint.r1",
		"experiment_id": perceptenvelope.ExperimentID,
		"stage":         "R1-B",
		"status":        "R1-B_SCALE_ENVELOPE_COMPLETE_FROZEN",
		"frozen_at":     time.Now().UTC().Format(time.RFC3339),
		"tlaloc_commit": commit,
		"records":       len(records),
		"errors":        curve.Errors,
		"raw_files":     rawFiles,
		"artifacts": map[string]string{
			"R1B_BASES.json":               basesSHA,
			"R1B_SCALE_DATASET.json":       datasetSHA,
			"R1B_GEOMETRY_AUDIT.json":      auditSHA,
			"R1_PROTOCOL_ADDENDUM_03.json": addSHA,
			"R1B_RECORDS.json":             recSHA,
			"R1B_SCALE_CURVE.json":         curveSHA,
			"raw_tree_sha256":              rawTreeSHA,
		},
		"formal_safe_scale_px":         curve.FormalSafeScalePx,
		"observed_operating_region_px": curve.ObservedOperatingPx,
		"token_regime_constant":        curve.TokenRegimeConstant,
		"overscale_degradation":        curve.OverscaleDegradation,
		"recommended_scale_for_r1c_px": curve.RecommendedR1CScalePx,
		"next_stage_not_started":       []string{"R1-C morphology", "R1-D distractors", "R1-E shortcut controls", "R1-F repeatability", "R1-G recovery"},
		"HARD_STOP":                    "Do not run R1-C/D/E/F/G. Review R1-B first.",
	}
	cpSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "R1B_CHECKPOINT.json"), checkpoint)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"records": len(records), "records_sha256": recSHA, "curve_sha256": curveSHA,
		"raw_tree_sha256": rawTreeSHA, "checkpoint_sha256": cpSHA,
		"formal_safe_scale_px": curve.FormalSafeScalePx, "rows": curve.Rows,
	}))
}

func short8(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// ---------------------------------------------------------------------------
// R1-C — NUMERIC MORPHOLOGY
// ---------------------------------------------------------------------------

func loadMorphPool(expDir string) perceptenvelope.MorphologyPool {
	body, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1C_POOL.json"))
	die(err)
	var pool perceptenvelope.MorphologyPool
	die(json.Unmarshal(body, &pool))
	return pool
}

func r1cExcludeSet(expDir string) map[string]struct{} {
	ex := map[string]struct{}{}
	for _, f := range []string{"R1A_BASES.json", "R1A1_BASES.json", "R1B_BASES.json"} {
		p := filepath.Join(expDir, "datasets", f)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		a := loadAlloc(p)
		for _, b := range a.Bases {
			ex[b.Candidate.CandidateID] = struct{}{}
		}
	}
	return ex
}

func loadR1CAlloc(expDir string) perceptenvelope.R1CAllocation {
	body, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1C_DATASET.json"))
	die(err)
	var a perceptenvelope.R1CAllocation
	die(json.Unmarshal(body, &a))
	return a
}

func buildR1CBank(storeDir, pdfPath string) *perceptenvelope.GlyphBank {
	bank, err := perceptenvelope.BuildGlyphBank(storeDir, pdfPath)
	die(err)
	return bank
}

// r1cBank loads the frozen glyph bank from the dataset dir, building and
// caching it on first use.
func r1cBank(expDir, storeDir, pdfPath string) *perceptenvelope.GlyphBank {
	bank, err := perceptenvelope.LoadOrBuildGlyphBank(
		filepath.Join(expDir, "datasets", "R1C_GLYPHBANK.json"), storeDir, pdfPath)
	die(err)
	return bank
}

func prepareR1C(args []string) {
	fs := flag.NewFlagSet("prepare-r1c", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	fs.Parse(args)

	pool := loadMorphPool(*expDir)
	alloc := perceptenvelope.AllocateR1C(pool, r1cExcludeSet(*expDir))
	bank := buildR1CBank(*storeDir, *pdfPath)

	realCount, synthCount := 0, 0
	for _, fa := range alloc.Families {
		realCount += len(fa.RealBases)
		synthCount += len(fa.SyntheticBases)
	}

	realDump := map[string]any{}
	synthDump := map[string]any{}
	for _, fa := range alloc.Families {
		realDump[fa.Family] = fa.RealBases
		if len(fa.SyntheticBases) > 0 {
			synthDump[fa.Family] = fa.SyntheticBases
		}
	}
	_, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1C_REAL_BASES.json"), realDump)
	die(err)
	_, err = perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1C_SYNTHETIC_BASES.json"), synthDump)
	die(err)
	datasetSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1C_DATASET.json"), alloc)
	die(err)
	bankSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1C_GLYPHBANK.json"), bank)
	die(err)

	selfTest := perceptenvelope.R1CScorerSelfTest()

	poolSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1C_POOL.json"))
	identitySHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "MODEL_IDENTITY.json"))
	r1bCkptSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "R1B_CHECKPOINT.json"))

	addendum := map[string]any{
		"schema":      "tlaloc.parrot-perceptual-envelope-r1.protocol-addendum.04",
		"title":       "R1-C_NUMERIC_MORPHOLOGY — frozen presentation policy + dual-endpoint scoring",
		"recorded_at": time.Now().UTC().Format(time.RFC3339),
		"authored":    "BEFORE any R1-C model output existed. R1-A0/R1-A1/R1-B raw outputs, curves and checkpoints are immutable and untouched.",
		"NO_R1C_MODEL_OUTPUT_EXISTED_WHEN_SCORING_AND_DATASET_WERE_FROZEN": true,
		"presentation_policy": map[string]any{
			"line_height_px": perceptenvelope.R1CLineHeightPx,
			"context_level":  perceptenvelope.R1CContextLevel,
			"canvas_px":      512,
			"target_centre":  []int{256, 256},
			"neutral_bg":     []int{200, 200, 200},
			"cue":            "R1-B B4 cue semantics, spanning the whole cued operand token",
			"opcode":         perceptenvelope.FrozenOpcode,
			"instruction":    perceptenvelope.FrozenInstruction,
			"temperature":    0,
			"max_tokens":     32,
			"note":           "held identical for every R1-C condition; R1-C varies morphology, not presentation",
		},
		"dual_endpoints": map[string]string{
			"VALUE_CORRECT":        "exact numeric / structured meaning (math/big Int and Rat; never float equality)",
			"SURFACE_FORM_CORRECT": "faithful transcription of the visible string; only outer-whitespace trim and dash-variant folding (U+002D/2010/2011/2012/2013/2014/2212 -> '-') are permitted",
		},
		"strata_policy":           "REAL_DOCUMENT and SYNTHETIC_REALISTIC accuracy are never pooled; synthetic evidence cannot promote a family to real-document RELIABLE",
		"synthetic_families":      []string{perceptenvelope.FamScientific, perceptenvelope.FamEquation, perceptenvelope.FamCoordTuple},
		"synthetic_renderer":      "glyph bank cut from the real corpus (perceptenvelope.BuildGlyphBank " + perceptenvelope.R1CGlyphBankVersion + "); channel-wise darken composite over the neutral canvas; no font dependency, no contrast alteration, no typographic tuning",
		"r1b_limitation_recorded": "R1-B's operational finding is 16 px = conservative tested floor, 32 px = nominal high-reliability point. The 8 px -> 12 px jump is NOT claimed as a universal hard perceptual phase transition: at 8 px the 1-pixel-minimum cue stroke gives a larger cue/line-height ratio than at larger rungs. Geometry sanity confirmed the target stayed intact, so B0 remains useful low-scale evidence, but exact attribution of the 8 px collapse is not required for R1-C.",
		"inputs_hashed": map[string]string{
			"R1C_POOL.json":       poolSHA,
			"R1C_DATASET.json":    datasetSHA,
			"R1C_GLYPHBANK.json":  bankSHA,
			"MODEL_IDENTITY.json": identitySHA,
			"R1B_CHECKPOINT.json": r1bCkptSHA,
		},
		"scorer_self_test_problems": selfTest,
	}
	addSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "R1_PROTOCOL_ADDENDUM_04.json"), addendum)
	die(err)

	manifest := map[string]any{
		"schema":                      "tlaloc.parrot-perceptual-envelope-r1.r1c-prepare-manifest.r1",
		"experiment_id":               perceptenvelope.ExperimentID,
		"r1c_pool_sha256":             poolSHA,
		"r1c_dataset_sha256":          datasetSHA,
		"r1c_glyphbank_sha256":        bankSHA,
		"protocol_addendum_04_sha256": addSHA,
		"model_identity_sha256":       identitySHA,
		"real_bases":                  realCount,
		"synthetic_bases":             synthCount,
		"total_records_expected":      realCount + synthCount,
		"scorer_self_test_ok":         len(selfTest) == 0,
		"families":                    familyBands(alloc),
	}
	mSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "manifests", "R1C_PREPARE_MANIFEST.json"), manifest)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"real_bases": realCount, "synthetic_bases": synthCount,
		"glyph_bank_sha256": bankSHA, "glyphs": len(bank.Glyphs),
		"scorer_self_test_problems": selfTest,
		"manifest_sha256":           mSHA, "addendum_04_sha256": addSHA,
		"families": familyBands(alloc),
	}))
	if len(selfTest) != 0 {
		os.Exit(1)
	}
}

func familyBands(alloc perceptenvelope.R1CAllocation) []map[string]any {
	var out []map[string]any
	for _, fa := range alloc.Families {
		out = append(out, map[string]any{
			"family": fa.Family, "band": fa.Band, "real_available": fa.RealAvailable,
			"real_bases": len(fa.RealBases), "synthetic_bases": len(fa.SyntheticBases),
		})
	}
	return out
}

func glyphbankR1C(args []string) {
	fs := flag.NewFlagSet("glyphbank-r1c", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	fs.Parse(args)
	bank := buildR1CBank(*storeDir, *pdfPath)
	prev, err := perceptenvelope.EncodeGlyphBankPreview(bank)
	die(err)
	outDir := filepath.Join(*expDir, "runs", "r1c-glyphbank")
	die(os.MkdirAll(outDir, 0o755))
	die(os.WriteFile(filepath.Join(outDir, "glyphbank_preview.png"), prev, 0o644))
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{"sha256": bank.SHA256, "glyphs": bank.Glyphs, "preview": filepath.Join(outDir, "glyphbank_preview.png")}))
}

func sanityR1C(ctxArgs []string) {
	fs := flag.NewFlagSet("sanity-r1c", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	perFam := fs.Int("per-family", 2, "real bases per family to render")
	fs.Parse(ctxArgs)

	alloc := loadR1CAlloc(*expDir)
	bank := r1cBank(*expDir, *storeDir, *pdfPath)
	outDir := filepath.Join(*expDir, "runs", "r1c-sanity", "crops")
	die(os.MkdirAll(outDir, 0o755))

	rendered := 0
	for _, fa := range alloc.Families {
		n := *perFam
		if n > len(fa.RealBases) {
			n = len(fa.RealBases)
		}
		for _, base := range fa.RealBases[:n] {
			img, e := perceptenvelope.RenderR1CReal(*storeDir, *pdfPath, base)
			die(e)
			die(perceptenvelope.WriteRGBA(filepath.Join(outDir, base.BaseID+".png"), img))
			rendered++
		}
		for _, base := range fa.SyntheticBases {
			img, _, e := perceptenvelope.RenderSyntheticNumber(bank, base.SyntheticTarget)
			die(e)
			die(perceptenvelope.WriteRGBA(filepath.Join(outDir, base.BaseID+".png"), img))
			rendered++
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{"rendered": rendered, "out_dir": outDir}))
}

func doctorR1C(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("doctor-r1c", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	fs.Parse(args)

	rep := perceptenvelope.DoctorR1C(ctx, perceptenvelope.DoctorR1CInput{
		ExpDir: *expDir, Endpoint: *endpoint, Model: *model, StoreDir: *storeDir,
	})
	_, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1C_DOCTOR.json"), rep)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(rep))
	if !rep.ReadyR1C {
		os.Exit(1)
	}
}

func runR1C(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-r1c", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	runID := fs.String("run-id", "r1c-r0", "run id")
	fs.Parse(args)

	doctorR1C(ctx, []string{"-exp-dir", *expDir, "-endpoint", *endpoint, "-model", *model, "-store-dir", *storeDir})

	alloc := loadR1CAlloc(*expDir)
	bank := r1cBank(*expDir, *storeDir, *pdfPath)
	runDir := filepath.Join(*expDir, "runs", *runID)
	records, err := perceptenvelope.RunR1C(ctx, perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: runDir,
	}, alloc, bank)
	die(err)
	finalizeR1C(*expDir, runDir, *model, records)
}

func reportR1C(args []string) {
	fs := flag.NewFlagSet("report-r1c", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	runID := fs.String("run-id", "r1c-r0", "run id")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	fs.Parse(args)
	var records []perceptenvelope.R1CRecord
	rb, err := os.ReadFile(filepath.Join(*expDir, "results", "R1C_RECORDS.json"))
	die(err)
	die(json.Unmarshal(rb, &records))
	finalizeR1C(*expDir, filepath.Join(*expDir, "runs", *runID), *model, records)
}

func finalizeR1C(expDir, runDir, model string, records []perceptenvelope.R1CRecord) {
	recSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1C_RECORDS.json"), records)
	die(err)
	table := perceptenvelope.AggregateR1C(records)
	tableSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1C_MORPHOLOGY_TABLE.json"), table)
	die(err)
	taxonomy := map[string]any{}
	for _, row := range table.Rows {
		taxonomy[row.Family+"|"+row.Provenance] = row.FailureClasses
	}
	taxSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1C_FAILURE_TAXONOMY.json"), taxonomy)
	die(err)

	rawTreeSHA, rawFiles, err := perceptenvelope.SHA256OfTree(filepath.Join(runDir, "raw"))
	die(err)

	alloc := loadR1CAlloc(expDir)
	datasetSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1C_DATASET.json"))
	bankSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1C_GLYPHBANK.json"))
	addSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "R1_PROTOCOL_ADDENDUM_04.json"))
	storeSHA := alloc.Seed // placeholder if store hash unavailable
	if b, e := os.ReadFile(filepath.Join(expDir, "datasets", "R1C_POOL.json")); e == nil {
		var p perceptenvelope.MorphologyPool
		if json.Unmarshal(b, &p) == nil {
			storeSHA = p.StoreRootSHA256
		}
	}
	commit := gitCommitShort()
	selfTest := perceptenvelope.R1CScorerSelfTest()
	scorerNote := "0 problems"
	if len(selfTest) > 0 {
		scorerNote = fmt.Sprintf("%d problems", len(selfTest))
	}

	report := perceptenvelope.RenderR1CReport(perceptenvelope.R1CReportInput{
		Table: table, Alloc: alloc, Model: model, GlyphBankSHA: bankSHA, DatasetSHA: datasetSHA,
		ScorerNote: scorerNote, RecordsSHA: recSHA, TableSHA: tableSHA, TaxonomySHA: taxSHA,
		RawTreeSHA: rawTreeSHA, AddendumSHA: addSHA, TlalocCommit: commit,
	})
	die(os.WriteFile(filepath.Join(expDir, "results", "R1C_REPORT.md"), []byte(report), 0o644))

	checkpoint := map[string]any{
		"schema":        "tlaloc.parrot-perceptual-envelope-r1.r1c-checkpoint.r1",
		"experiment_id": perceptenvelope.ExperimentID,
		"stage":         "R1-C",
		"status":        "R1-C_NUMERIC_MORPHOLOGY_COMPLETE_FROZEN",
		"frozen_at":     time.Now().UTC().Format(time.RFC3339),
		"tlaloc_commit": commit,
		"records":       len(records),
		"errors":        table.Errors,
		"raw_files":     rawFiles,
		"hashes": map[string]string{
			"R1C_RECORDS.json":             recSHA,
			"R1C_MORPHOLOGY_TABLE.json":    tableSHA,
			"R1C_FAILURE_TAXONOMY.json":    taxSHA,
			"R1C_DATASET.json":             datasetSHA,
			"R1C_GLYPHBANK.json":           bankSHA,
			"R1_PROTOCOL_ADDENDUM_04.json": addSHA,
			"raw_tree_sha256":              rawTreeSHA,
			"source_store_root_sha256":     storeSHA,
		},
		"scorer_self_test_problems": selfTest,
		"verdicts":                  table.Verdicts,
		"HARD_STOP":                 "Do not run R1-D/E/F/G. Return the morphology table for review.",
	}
	cpSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "R1C_CHECKPOINT.json"), checkpoint)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"records": len(records), "errors": table.Errors,
		"records_sha256": recSHA, "table_sha256": tableSHA, "checkpoint_sha256": cpSHA,
		"verdicts": table.Verdicts, "answers": table.Answers,
	}))
}

// ---------------------------------------------------------------------------
// R1-D — LABEL/VALUE ASSOCIATION + DISTRACTOR DENSITY
// ---------------------------------------------------------------------------

func loadLVPool(expDir string) perceptenvelope.LabelValuePool {
	body, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1D_POOL.json"))
	die(err)
	var p perceptenvelope.LabelValuePool
	die(json.Unmarshal(body, &p))
	return p
}

func loadR1DAlloc(expDir string) perceptenvelope.R1DAllocation {
	body, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1D_ASSOCIATION_DATASET.json"))
	die(err)
	var a perceptenvelope.R1DAllocation
	die(json.Unmarshal(body, &a))
	return a
}

func prepareR1D(args []string) {
	fs := flag.NewFlagSet("prepare-r1d", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	fs.Parse(args)

	_ = *storeDir
	_ = *pdfPath
	pool := loadLVPool(*expDir)
	alloc := perceptenvelope.AllocateR1D(pool)

	var geos []perceptenvelope.R1DGeometry
	for _, b := range alloc.Bases {
		if !b.Eligible {
			continue
		}
		g, err := perceptenvelope.DeriveR1DGeometry(b)
		die(err)
		geos = append(geos, g)
	}

	realBases := []perceptenvelope.R1DBase{}
	for _, b := range alloc.Bases {
		if b.Eligible {
			realBases = append(realBases, b)
		}
	}
	_, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1D_REAL_BASES.json"), realBases)
	die(err)
	datasetSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1D_ASSOCIATION_DATASET.json"), alloc)
	die(err)
	distDump := map[string]any{}
	for _, b := range realBases {
		distDump[b.BaseID] = b.DistractorValues
	}
	distSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1D_DISTRACTOR_DATASET.json"), map[string]any{
		"seed":            perceptenvelope.R1DSeed,
		"ladder":          []int{0, 1, 2, 4, 8},
		"rule":            "plain 2-4 digit integers, != answer, distinct, digit-length balanced, deterministic from sha256(seed||'distractor'||candidate_id)",
		"placement":       "frozen ring of 12 candidate slots (2 distance bands) around the label/value pair; first K non-overlapping in fixed order; pair never moves",
		"per_base_values": distDump,
	})
	die(err)
	auditSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1D_GEOMETRY_AUDIT.json"), map[string]any{
		"schema":         "tlaloc.parrot-perceptual-envelope-r1.r1d-geometry-audit.r1",
		"eligible_bases": alloc.EligibleCount,
		"pool_count":     alloc.PoolCount,
		"proceed":        alloc.Proceed,
		"per_base":       geos,
		"ineligible":     ineligibleR1D(alloc),
	})
	die(err)

	poolSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1D_POOL.json"))
	identitySHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "MODEL_IDENTITY.json"))
	r1cCkptSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "R1C_CHECKPOINT.json"))

	addendum := map[string]any{
		"schema":      "tlaloc.parrot-perceptual-envelope-r1.protocol-addendum.05",
		"title":       "R1-D_ASSOCIATION_DISTRACTOR — label/value association + controlled distractor density",
		"recorded_at": time.Now().UTC().Format(time.RFC3339),
		"authored":    "BEFORE any R1-D model output existed. R1-A0/A1/B/C artifacts immutable and untouched.",
		"NO_R1D_MODEL_OUTPUT_EXISTED_WHEN_DATASET_AND_DISTRACTOR_RULES_WERE_FROZEN": true,
		"assoc_opcode":       perceptenvelope.R1DAssocOpcode,
		"assoc_instruction":  perceptenvelope.R1DAssocInstruction,
		"microisa_promotion": "READ_ASSOCIATED_NUMBER is defined in the behaviour lab only; promotion to the T0 Micro-ISA vocabulary is a separate decision, deferred pending this evidence.",
		"presentation": map[string]any{
			"line_height_px": perceptenvelope.R1DLineHeightPx, "canvas_px": 512,
			"viewport":                   "single containing text line; other-line pixels masked to RGB(200,200,200)",
			"primary_operand_morphology": "MULTI_DIGIT_INTEGER only (fragile R1-C morphologies excluded from D0/D1 primary aggregate)",
			"temperature":                0, "max_tokens": 32,
		},
		"d0_conditions": map[string]string{
			"D0V_VALUE_CUED": "cue the value token; instruction EXTRACT_NUMBER; per-base atomic-read control",
			"D0L_LABEL_CUED": "cue the label token; instruction READ_ASSOCIATED_NUMBER; requires association",
		},
		"d0_shared_pixels": "both D0 conditions render from ONE per-base viewport; only the cue rectangle differs",
		"d1_track":         "CONTROLLED_COMPOSITE — never pooled with D0; base = D0L viewport; distractor ladder K=0/1/2/4/8; original line pixels + cue byte-identical across K",
		"distractor_rules": "plain 2-4 digit integers, deterministic seed " + perceptenvelope.R1DSeed + ", all != answer + distinct + digit-length balanced; frozen 12-slot placement ring, no overlap with label/value/cue",
		"geometry_gate":    "D1 is canonical only if D0V value accuracy >= 0.90 AND Wilson 95% lower bound >= 0.70; otherwise D1 is exploratory",
		"eligibility_rule": "value is a 2-5 digit plain integer; value unique in line; label precedes value and has a >=3-letter non-stopword token; label-value span at 32px <= 480 canvas px. >=18 eligible -> proceed with all eligible; <18 -> STOP.",
		"eligible_count":   alloc.EligibleCount,
		"proceed":          alloc.Proceed,
		"inputs_hashed": map[string]string{
			"R1D_POOL.json":                poolSHA,
			"R1D_ASSOCIATION_DATASET.json": datasetSHA,
			"R1D_DISTRACTOR_DATASET.json":  distSHA,
			"R1D_GEOMETRY_AUDIT.json":      auditSHA,
			"MODEL_IDENTITY.json":          identitySHA,
			"R1C_CHECKPOINT.json":          r1cCkptSHA,
		},
	}
	addSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "R1_PROTOCOL_ADDENDUM_05.json"), addendum)
	die(err)

	manifest := map[string]any{
		"schema":                         "tlaloc.parrot-perceptual-envelope-r1.r1d-prepare-manifest.r1",
		"experiment_id":                  perceptenvelope.ExperimentID,
		"r1d_pool_sha256":                poolSHA,
		"r1d_association_dataset_sha256": datasetSHA,
		"r1d_distractor_dataset_sha256":  distSHA,
		"r1d_geometry_audit_sha256":      auditSHA,
		"protocol_addendum_05_sha256":    addSHA,
		"eligible_bases":                 alloc.EligibleCount,
		"pool_candidates":                alloc.PoolCount,
		"proceed":                        alloc.Proceed,
		"expected_d0_records":            alloc.EligibleCount * 2,
		"expected_d1_records":            alloc.EligibleCount * 5,
	}
	mSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "manifests", "R1D_PREPARE_MANIFEST.json"), manifest)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"eligible_bases": alloc.EligibleCount, "pool_candidates": alloc.PoolCount,
		"proceed": alloc.Proceed, "min_required": alloc.MinRequired,
		"ineligible":      ineligibleR1D(alloc),
		"manifest_sha256": mSHA, "addendum_05_sha256": addSHA,
		"expected_d0_records": alloc.EligibleCount * 2, "expected_d1_records": alloc.EligibleCount * 5,
	}))
	if !alloc.Proceed {
		fmt.Fprintln(os.Stderr, "STOP: fewer than 18 eligible label/value bases; R1-D not started")
		os.Exit(1)
	}
}

func ineligibleR1D(alloc perceptenvelope.R1DAllocation) []map[string]string {
	var out []map[string]string
	for _, b := range alloc.Bases {
		if !b.Eligible {
			out = append(out, map[string]string{"base_id": b.BaseID, "label": b.Label, "value": b.Value, "reason": b.IneligibleReason})
		}
	}
	return out
}

func sanityR1D(args []string) {
	fs := flag.NewFlagSet("sanity-r1d", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	n := fs.Int("bases", 3, "eligible bases to render")
	fs.Parse(args)
	alloc := loadR1DAlloc(*expDir)
	bank := r1cBank(*expDir, *storeDir, *pdfPath)
	outDir := filepath.Join(*expDir, "runs", "r1d-sanity", "crops")
	die(os.MkdirAll(outDir, 0o755))
	rendered := 0
	done := 0
	for _, base := range alloc.Bases {
		if !base.Eligible || done >= *n {
			continue
		}
		done++
		imgs, err := perceptenvelope.RenderR1DSanity(*storeDir, *pdfPath, base, bank)
		die(err)
		for name, img := range imgs {
			die(perceptenvelope.WriteRGBA(filepath.Join(outDir, base.BaseID+"_"+name+".png"), img))
			rendered++
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{"bases": done, "rendered": rendered, "out_dir": outDir}))
}

func doctorR1D(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("doctor-r1d", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	fs.Parse(args)
	rep := perceptenvelope.DoctorR1D(ctx, perceptenvelope.DoctorR1DInput{
		ExpDir: *expDir, Endpoint: *endpoint, Model: *model, StoreDir: *storeDir,
	})
	_, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1D_DOCTOR.json"), rep)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(rep))
	if !rep.ReadyR1D {
		os.Exit(1)
	}
}

func runR1D(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-r1d", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	runID := fs.String("run-id", "r1d-r0", "run id")
	fs.Parse(args)

	doctorR1D(ctx, []string{"-exp-dir", *expDir, "-endpoint", *endpoint, "-model", *model, "-store-dir", *storeDir})

	alloc := loadR1DAlloc(*expDir)
	bank := r1cBank(*expDir, *storeDir, *pdfPath)
	runDir := filepath.Join(*expDir, "runs", *runID)
	cfg := perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: runDir,
	}
	d0, err := perceptenvelope.RunR1D0(ctx, cfg, alloc)
	die(err)
	d0Table := perceptenvelope.AggregateR1D0(d0)
	// D0 integrity: only structural errors are integrity failures, not accuracy.
	d0Errs := 0
	for _, r := range d0 {
		if r.Error != "" {
			d0Errs++
		}
	}
	if d0Errs > 0 {
		die(fmt.Errorf("D0 integrity: %d records errored; STOP before D1", d0Errs))
	}
	d1, err := perceptenvelope.RunR1D1(ctx, cfg, alloc, bank)
	die(err)
	finalizeR1D(*expDir, runDir, *model, alloc, d0, d0Table, d1)
}

func reportR1D(args []string) {
	fs := flag.NewFlagSet("report-r1d", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	runID := fs.String("run-id", "r1d-r0", "run id")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	fs.Parse(args)
	var d0, d1 []perceptenvelope.R1DRecord
	b0, err := os.ReadFile(filepath.Join(*expDir, "results", "R1D_ASSOCIATION_RECORDS.json"))
	die(err)
	die(json.Unmarshal(b0, &d0))
	b1, err := os.ReadFile(filepath.Join(*expDir, "results", "R1D_DISTRACTOR_RECORDS.json"))
	die(err)
	die(json.Unmarshal(b1, &d1))
	alloc := loadR1DAlloc(*expDir)
	finalizeR1D(*expDir, filepath.Join(*expDir, "runs", *runID), *model, alloc, d0, perceptenvelope.AggregateR1D0(d0), d1)
}

func finalizeR1D(expDir, runDir, model string, alloc perceptenvelope.R1DAllocation, d0 []perceptenvelope.R1DRecord, d0Table perceptenvelope.R1D0Table, d1 []perceptenvelope.R1DRecord) {
	assocRecSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1D_ASSOCIATION_RECORDS.json"), d0)
	die(err)
	assocTblSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1D_ASSOCIATION_TABLE.json"), d0Table)
	die(err)
	distRecSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1D_DISTRACTOR_RECORDS.json"), d1)
	die(err)
	d1Curve := perceptenvelope.AggregateR1D1(d1, d0Table.RealAssocGeometryValid)
	distCurveSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1D_DISTRACTOR_CURVE.json"), d1Curve)
	die(err)
	verdict := perceptenvelope.R1DProvisionalVerdict(d0Table, d1Curve)

	tax := map[string]any{}
	for _, r := range d0Table.Rows {
		tax["D0|"+r.Condition] = r.FailureClasses
	}
	for _, r := range d1Curve.Rows {
		tax["D1|"+r.Condition] = r.FailureClasses
	}
	taxSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1D_FAILURE_TAXONOMY.json"), tax)
	die(err)

	rawTreeSHA, rawFiles, err := perceptenvelope.SHA256OfTree(runDir)
	die(err)
	datasetSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1D_ASSOCIATION_DATASET.json"))
	addSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "R1_PROTOCOL_ADDENDUM_05.json"))
	commit := gitCommitShort()

	report := perceptenvelope.RenderR1DReport(perceptenvelope.R1DReportInput{
		Alloc: alloc, D0: d0Table, D1: d1Curve, Verdict: verdict, Model: model,
		AssocRecSHA: assocRecSHA, AssocTblSHA: assocTblSHA, DistRecSHA: distRecSHA,
		DistCurveSHA: distCurveSHA, TaxonomySHA: taxSHA, DatasetSHA: datasetSHA,
		AddendumSHA: addSHA, RawTreeSHA: rawTreeSHA, TlalocCommit: commit,
	})
	die(os.WriteFile(filepath.Join(expDir, "results", "R1D_REPORT.md"), []byte(report), 0o644))

	checkpoint := map[string]any{
		"schema":                          "tlaloc.parrot-perceptual-envelope-r1.r1d-checkpoint.r1",
		"experiment_id":                   perceptenvelope.ExperimentID,
		"stage":                           "R1-D",
		"status":                          "R1-D_ASSOCIATION_DISTRACTOR_COMPLETE_FROZEN",
		"frozen_at":                       time.Now().UTC().Format(time.RFC3339),
		"tlaloc_commit":                   commit,
		"d0_records":                      len(d0),
		"d1_records":                      len(d1),
		"raw_files":                       rawFiles,
		"real_association_geometry_valid": d0Table.RealAssocGeometryValid,
		"provisional_verdict":             verdict,
		"hashes": map[string]string{
			"R1D_ASSOCIATION_RECORDS.json": assocRecSHA,
			"R1D_ASSOCIATION_TABLE.json":   assocTblSHA,
			"R1D_DISTRACTOR_RECORDS.json":  distRecSHA,
			"R1D_DISTRACTOR_CURVE.json":    distCurveSHA,
			"R1D_FAILURE_TAXONOMY.json":    taxSHA,
			"R1D_ASSOCIATION_DATASET.json": datasetSHA,
			"R1_PROTOCOL_ADDENDUM_05.json": addSHA,
			"raw_tree_sha256":              rawTreeSHA,
		},
		"HARD_STOP": "Do not run R1-E/F/G. Return the full R1-D report for review.",
	}
	cpSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "R1D_CHECKPOINT.json"), checkpoint)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"d0_records": len(d0), "d1_records": len(d1),
		"assoc_table_sha256": assocTblSHA, "distractor_curve_sha256": distCurveSHA,
		"checkpoint_sha256": cpSHA, "geometry_valid": d0Table.RealAssocGeometryValid,
		"verdict": verdict,
	}))
}

// ---- R1-E: visual dependence / shortcut controls ----------------------------

func loadR1EDataset(expDir string) perceptenvelope.R1EDataset {
	body, err := os.ReadFile(filepath.Join(expDir, "datasets", "R1E_DATASET.json"))
	die(err)
	var ds perceptenvelope.R1EDataset
	die(json.Unmarshal(body, &ds))
	return ds
}

func prepareR1E(args []string) {
	fs := flag.NewFlagSet("prepare-r1e", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	fs.Parse(args)

	alloc := loadR1DAlloc(*expDir)
	elig := perceptenvelope.EligibleR1DBases(alloc)
	if len(elig) < 6 {
		die(fmt.Errorf("only %d eligible R1-D bases (<6); cannot run R1-E", len(elig)))
	}
	ds, err := perceptenvelope.BuildR1EDataset(elig)
	die(err)

	basesSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1E_BASES.json"), elig)
	die(err)
	wrongSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1E_WRONG_IMAGE_MAP.json"), ds.WrongMap)
	die(err)
	dsSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "datasets", "R1E_DATASET.json"), ds)
	die(err)

	r1dDatasetSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "datasets", "R1D_ASSOCIATION_DATASET.json"))
	r1dCheckpointSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "R1D_CHECKPOINT.json"))
	miSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(*expDir, "MODEL_IDENTITY.json"))

	addendum := map[string]any{
		"schema":       "tlaloc.parrot-perceptual-envelope-r1.protocol-addendum.06",
		"title":        "R1-E_VISUAL_DEPENDENCE — no-image / wrong-image / correct-image shortcut controls",
		"recorded_at":  time.Now().UTC().Format(time.RFC3339),
		"authored":     "BEFORE any R1-E model output existed. R1-A0/A1/B/C/D artifacts immutable and untouched.",
		"NO_R1E_MODEL_OUTPUT_EXISTED_WHEN_INTERVENTIONS_WERE_FROZEN": true,
		"INTERVENTION_REUSE_OF_R1D_BASES":                            true,
		"primary_capability":                                        "READ_ASSOCIATED_NUMBER",
		"positive_calibration_control":                              string(perceptenvelope.FrozenOpcode),
		"conditions":                                                perceptenvelope.R1EConditions,
		"wrong_image_pairing_rule":                                  ds.WrongMap.Rule,
		"scoring": map[string]string{
			"TASK_GOLD_CORRECT": "model returned the base's own associated value Y",
			"IMAGE_CONSISTENT":  "model returned the value Y2 actually visible in the (wrong) image, Y2 != Y",
		},
		"secondary_capabilities_note": "SELECT_ONE / READ_SHORT_LABEL / EXTRACT_ENTITY have no frozen suitable stimulus " +
			"set in the R1 perceptual-envelope experiment; per protocol §9 R1-E does not manufacture one. " +
			"The T0-B SELECT_ONE shortcut check is deferred to a dedicated stage.",
		"inputs_hashed": map[string]string{
			"R1D_ASSOCIATION_DATASET.json": r1dDatasetSHA,
			"R1D_CHECKPOINT.json":          r1dCheckpointSHA,
			"MODEL_IDENTITY.json":          miSHA,
			"R1E_BASES.json":               basesSHA,
			"R1E_WRONG_IMAGE_MAP.json":     wrongSHA,
			"R1E_DATASET.json":             dsSHA,
		},
	}
	addSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "R1_PROTOCOL_ADDENDUM_06.json"), addendum)
	die(err)

	matched := 0
	for _, p := range ds.WrongMap.Pairs {
		if p.DigitLenMatched {
			matched++
		}
	}
	manifest := map[string]any{
		"schema":                     "tlaloc.parrot-perceptual-envelope-r1.r1e-prepare-manifest.r1",
		"experiment_id":              perceptenvelope.ExperimentID,
		"prepared_at":                time.Now().UTC().Format(time.RFC3339),
		"eligible_intervention_bases": len(elig),
		"capabilities":               len(perceptenvelope.R1ECapabilities),
		"conditions":                 len(perceptenvelope.R1EConditions),
		"expected_records":           len(elig) * len(perceptenvelope.R1ECapabilities) * len(perceptenvelope.R1EConditions),
		"digit_length_matched_pairs": matched,
		"r1e_bases_sha256":           basesSHA,
		"r1e_wrong_image_map_sha256": wrongSHA,
		"r1e_dataset_sha256":         dsSHA,
		"protocol_addendum_06_sha256": addSHA,
	}
	mSHA, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "manifests", "R1E_PREPARE_MANIFEST.json"), manifest)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"eligible_intervention_bases": len(elig),
		"expected_records":            manifest["expected_records"],
		"digit_length_matched_pairs":  matched,
		"dataset_sha256":              dsSHA,
		"wrong_image_map_sha256":      wrongSHA,
		"addendum_06_sha256":          addSHA,
		"manifest_sha256":             mSHA,
	}))
}

func doctorR1E(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("doctor-r1e", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	fs.Parse(args)
	rep := perceptenvelope.DoctorR1E(ctx, perceptenvelope.DoctorR1EInput{
		ExpDir: *expDir, Endpoint: *endpoint, Model: *model, StoreDir: *storeDir,
	})
	_, err := perceptenvelope.WriteJSON(filepath.Join(*expDir, "results", "R1E_DOCTOR.json"), rep)
	die(err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(rep))
	if !rep.ReadyR1E {
		os.Exit(1)
	}
}

func runR1E(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run-r1e", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	storeDir := fs.String("store-dir", defaultStore, "reconstructed pdfmemory store root")
	pdfPath := fs.String("pdf", "", "source PDF override")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234", "model endpoint")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	temp := fs.Float64("temperature", 0, "sampling temperature")
	maxTokens := fs.Int("max-tokens", 32, "max output tokens")
	runID := fs.String("run-id", "r1e-r0", "run id")
	fs.Parse(args)

	doctorR1E(ctx, []string{"-exp-dir", *expDir, "-endpoint", *endpoint, "-model", *model, "-store-dir", *storeDir})

	ds := loadR1EDataset(*expDir)
	runDir := filepath.Join(*expDir, "runs", *runID)
	cfg := perceptenvelope.RunConfig{
		StoreDir: *storeDir, PDFPath: *pdfPath, Endpoint: *endpoint, Model: *model,
		Temperature: *temp, MaxTokens: *maxTokens, RunDir: runDir,
	}
	records, err := perceptenvelope.RunR1E(ctx, cfg, ds)
	die(err)
	structErrs := 0
	for _, r := range records {
		if r.Error != "" {
			structErrs++
		}
	}
	if structErrs > 0 {
		die(fmt.Errorf("R1-E integrity: %d records errored", structErrs))
	}
	finalizeR1E(*expDir, runDir, *model, ds, records)
}

func reportR1E(args []string) {
	fs := flag.NewFlagSet("report-r1e", flag.ExitOnError)
	expDir := fs.String("exp-dir", defaultExpDir, "experiment directory")
	runID := fs.String("run-id", "r1e-r0", "run id")
	model := fs.String("model", "lfm2-vl-1.6b", "model id")
	fs.Parse(args)
	body, err := os.ReadFile(filepath.Join(*expDir, "results", "R1E_RECORDS.json"))
	die(err)
	var records []perceptenvelope.R1ERecord
	die(json.Unmarshal(body, &records))
	ds := loadR1EDataset(*expDir)
	finalizeR1E(*expDir, filepath.Join(*expDir, "runs", *runID), *model, ds, records)
}

func finalizeR1E(expDir, runDir, model string, ds perceptenvelope.R1EDataset, records []perceptenvelope.R1ERecord) {
	recSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1E_RECORDS.json"), records)
	die(err)
	table := perceptenvelope.AggregateR1E(records, ds)
	tableSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "results", "R1E_VISUAL_DEPENDENCE_TABLE.json"), table)
	die(err)

	rawTreeSHA, rawFiles, err := perceptenvelope.SHA256OfTree(runDir)
	die(err)
	dsSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1E_DATASET.json"))
	wrongSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "datasets", "R1E_WRONG_IMAGE_MAP.json"))
	addSHA, _ := perceptenvelope.SHA256OfFile(filepath.Join(expDir, "R1_PROTOCOL_ADDENDUM_06.json"))
	commit := gitCommitShort()

	report := perceptenvelope.RenderR1EReport(perceptenvelope.R1EReportInput{
		Dataset: ds, Table: table, Model: model,
		RecordsSHA: recSHA, TableSHA: tableSHA, DatasetSHA: dsSHA, WrongMapSHA: wrongSHA,
		AddendumSHA: addSHA, RawTreeSHA: rawTreeSHA, TlalocCommit: commit,
	})
	die(os.WriteFile(filepath.Join(expDir, "results", "R1E_REPORT.md"), []byte(report), 0o644))

	disp, why := perceptenvelope.R1EReadAssocDisposition(table)
	classes := map[string]string{}
	for _, c := range table.Capabilities {
		classes[c.Capability] = c.Classification
	}
	checkpoint := map[string]any{
		"schema":        "tlaloc.parrot-perceptual-envelope-r1.r1e-checkpoint.r1",
		"experiment_id": perceptenvelope.ExperimentID,
		"stage":         "R1-E",
		"status":        "R1-E_VISUAL_DEPENDENCE_COMPLETE_FROZEN",
		"frozen_at":     time.Now().UTC().Format(time.RFC3339),
		"tlaloc_commit": commit,
		"records":       len(records),
		"raw_files":     rawFiles,
		"INTERVENTION_REUSE_OF_R1D_BASES":                   true,
		"NO_R1E_MODEL_OUTPUT_EXISTED_WHEN_INTERVENTIONS_WERE_FROZEN": true,
		"classifications":            classes,
		"read_associated_number_disposition": map[string]string{"disposition": disp, "basis": why},
		"hashes": map[string]string{
			"R1E_RECORDS.json":                  recSHA,
			"R1E_VISUAL_DEPENDENCE_TABLE.json":  tableSHA,
			"R1E_DATASET.json":                  dsSHA,
			"R1E_WRONG_IMAGE_MAP.json":          wrongSHA,
			"R1_PROTOCOL_ADDENDUM_06.json":      addSHA,
			"raw_tree_sha256":                   rawTreeSHA,
		},
		"HARD_STOP": "Do not run R1-F or R1-G. Return the complete visual-dependence table for review.",
	}
	cpSHA, err := perceptenvelope.WriteJSON(filepath.Join(expDir, "R1E_CHECKPOINT.json"), checkpoint)
	die(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	die(enc.Encode(map[string]any{
		"records":           len(records),
		"table_sha256":      tableSHA,
		"checkpoint_sha256": cpSHA,
		"classifications":   classes,
		"disposition":       disp,
	}))
}
