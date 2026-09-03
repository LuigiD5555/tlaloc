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
