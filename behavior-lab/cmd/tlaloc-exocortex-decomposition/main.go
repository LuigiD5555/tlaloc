// Command tlaloc-exocortex-decomposition runs T0
// (ONE_OP_DECOMPOSITION_R0 / exocortex-decomposition-r0): it consumes the
// frozen P0 dataset and P2-A CapabilityProfile artifact as read-only
// evidence and drives the internal/exocortex Exocortex slice through the
// T0-A (oracle) and T0-B (real) conditions. It fabricates nothing: `run`
// only ever produces a RecordOutcome from an actual pipeline execution
// against a real endpoint, and `aggregate` only ever summarizes records
// that `run` actually wrote.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/decompositionlab"
	"tlaloc.local/behaviorlab/internal/exocortex"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	switch os.Args[1] {
	case "prepare":
		prepare(os.Args[2:])
	case "doctor":
		doctor(ctx, os.Args[2:])
	case "freeze":
		freeze(os.Args[2:])
	case "run":
		run(ctx, os.Args[2:])
	case "aggregate":
		aggregate(os.Args[2:])
	case "example":
		example()
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `tlaloc-exocortex-decomposition <prepare|doctor|freeze|run|aggregate|example> [flags]

  prepare    deterministic import of the real frozen P0/P2-A experiments:
             --p0-experiment <dir> --p2-experiment <artifact.json> --out <dir>.
             Verifies hashes, compiles the runtime profile, writes the T0-B
             eligibility audit + C0 baseline + manifest. Zero model calls.
  doctor     validate the P0 dataset, the frozen P2-A artifact, and (if
             --endpoint is set) that the endpoint serves --model-id.
  freeze     hash-verify P0/P2-A and write manifest.json to --out.
  run        execute one or more conditions over every P0 record and write
             one raw RecordOutcome JSON file per (condition, base_id) under
             --out.
  aggregate  read every raw RecordOutcome under --in and write the final
             results/EXOCORTEX_DECOMPOSITION_R0.json to --out.
  example    print an example Spec and P0 dataset record.`)
}

func specFlags(fs *flag.FlagSet) *decompositionlab.Spec {
	spec := &decompositionlab.Spec{}
	fs.StringVar(&spec.DatasetPath, "dataset", "", "path to the T0 P0 image dataset (30 records)")
	fs.StringVar(&spec.MicroISAArtifactPath, "microisa", "", "path to the frozen P2-A artifact (results/PARROT_MICRO_ISA_R0.json)")
	fs.StringVar(&spec.ExecutorID, "executor-id", "parrot-lfm2-vl-1.6b", "CapabilityProfile executor id")
	fs.StringVar(&spec.ModelID, "model-id", "lfm2-vl-1.6b", "model id served at --endpoint")
	fs.StringVar(&spec.ProfileVersion, "profile-version", "r0", "CapabilityProfile version tag")
	fs.StringVar(&spec.Endpoint, "endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL (LM Studio default)")
	fs.Float64Var(&spec.Temperature, "temperature", 0, "model temperature")
	fs.IntVar(&spec.MaxOutputTokens, "max-output-tokens", 32, "max generated tokens per Parrot call (one-op, minimal output)")
	fs.Float64Var(&spec.MarginRatio, "margin-ratio", 0.15, "crop expansion margin ratio")
	fs.StringVar(&spec.PDFMemoryStoreDir, "store-dir", "", "pdfmemory store root; required only for B1-B3 REAL conditions")
	fs.StringVar(&spec.OutputDir, "out", "runs/exocortex-decomposition-r0", "output directory")
	return spec
}

func prepare(args []string) {
	fs := flag.NewFlagSet("prepare", flag.ExitOnError)
	p0 := fs.String("p0-experiment", "", "frozen P0 experiment dir (parrot-capability-r0)")
	p2 := fs.String("p2-experiment", "", "frozen P2-A artifact (results/PARROT_MICRO_ISA_R0.json)")
	executorID := fs.String("executor-id", "parrot-lfm2-vl-1.6b", "CapabilityProfile executor id")
	modelID := fs.String("model-id", "lfm2-vl-1.6b", "model id")
	version := fs.String("profile-version", "r0", "profile version tag")
	out := fs.String("out", "experiments/exocortex-decomposition-r0", "output directory")
	fs.Parse(args)
	if strings.TrimSpace(*p0) == "" || strings.TrimSpace(*p2) == "" {
		die(fmt.Errorf("--p0-experiment and --p2-experiment are required"))
	}
	manifest, err := decompositionlab.Prepare(decompositionlab.PrepareInput{
		P0ExperimentDir: *p0, P2AArtifactPath: *p2,
		ExecutorID: *executorID, ModelID: *modelID, ProfileVersion: *version, OutDir: *out,
	})
	die(err)
	write(manifest)
}

func doctor(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	spec := specFlags(fs)
	fs.Parse(args)
	result, err := decompositionlab.Doctor(ctx, *spec)
	die(err)
	write(result)
	if !result.Ready {
		os.Exit(1)
	}
}

func freeze(args []string) {
	fs := flag.NewFlagSet("freeze", flag.ExitOnError)
	spec := specFlags(fs)
	fs.Parse(args)
	manifest, dataset, err := decompositionlab.Freeze(*spec)
	die(err)
	die(os.MkdirAll(spec.OutputDir, 0o755))
	manifestPath := filepath.Join(spec.OutputDir, "manifest.json")
	die(decompositionlab.WriteManifest(manifestPath, manifest))
	write(map[string]any{"manifest_path": manifestPath, "manifest": manifest, "dataset_records": len(dataset.Records)})
}

func run(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	spec := specFlags(fs)
	conditionsFlag := fs.String("conditions", "C0_PARROT_DIRECT,C1_ORACLE_CROP_PARROT,C2_ORACLE_CROP_PARROT_NORMALIZE,C3_ORACLE_CROP_PARROT_NORMALIZE_VERIFY", "comma-separated Condition list")
	fs.Parse(args)

	_, dataset, err := decompositionlab.Freeze(*spec)
	die(err)
	profile, err := exocortex.CompileParrotProfile(spec.MicroISAArtifactPath, spec.ExecutorID, spec.ModelID, spec.ProfileVersion)
	die(err)
	die(exocortex.VerifySourceArtifact(profile))

	conditions := parseConditions(*conditionsFlag)
	for _, c := range conditions {
		if !c.IsOracle() && strings.TrimSpace(spec.PDFMemoryStoreDir) == "" {
			die(fmt.Errorf("condition %s requires --store-dir (real locate needs an Origami pdfmemory store)", c))
		}
	}

	blackboardRoot := filepath.Join(spec.OutputDir, "blackboard")
	cropDir := filepath.Join(spec.OutputDir, "crops")
	rawDir := filepath.Join(spec.OutputDir, "raw")
	die(os.MkdirAll(cropDir, 0o755))
	die(os.MkdirAll(rawDir, 0o755))

	store := blackboard.New(blackboardRoot)
	registry, err := decompositionlab.NewRegistry(profile, exocortex.ParrotEndpoint{
		BaseURL: spec.Endpoint, Model: spec.ModelID, Temperature: spec.Temperature, MaxTokens: spec.MaxOutputTokens,
	})
	die(err)
	cfg := decompositionlab.RunnerConfig{Registry: registry, Store: store, MarginRatio: spec.MarginRatio, CropDir: cropDir, StoreDir: spec.PDFMemoryStoreDir}

	total, attempted := 0, 0
	for _, condition := range conditions {
		condDir := filepath.Join(rawDir, string(condition))
		die(os.MkdirAll(condDir, 0o755))
		for _, record := range dataset.Records {
			select {
			case <-ctx.Done():
				die(ctx.Err())
			default:
			}
			outcome := decompositionlab.RunRecord(ctx, cfg, record, condition)
			total++
			if outcome.Attempted {
				attempted++
			}
			body, err := json.MarshalIndent(outcome, "", "  ")
			die(err)
			die(os.WriteFile(filepath.Join(condDir, record.BaseID+".json"), body, 0o644))
			fmt.Fprintf(os.Stderr, "%s/%s: contract=%v semantic=%v error=%q\n", condition, record.BaseID, outcome.ContractSuccess, outcome.SemanticCorrect, outcome.Error)
		}
	}
	write(map[string]any{"raw_dir": rawDir, "records_run": total, "records_attempted": attempted})
}

func aggregate(args []string) {
	fs := flag.NewFlagSet("aggregate", flag.ExitOnError)
	inDir := fs.String("in", "", "raw records directory (the run command's --out/raw)")
	outPath := fs.String("out", "results/EXOCORTEX_DECOMPOSITION_R0.json", "final results JSON path")
	fs.Parse(args)
	if strings.TrimSpace(*inDir) == "" {
		die(fmt.Errorf("--in is required"))
	}

	byCondition := map[decompositionlab.Condition][]decompositionlab.RecordOutcome{}
	entries, err := os.ReadDir(*inDir)
	die(err)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		condition := decompositionlab.Condition(e.Name())
		files, err := os.ReadDir(filepath.Join(*inDir, e.Name()))
		die(err)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			body, err := os.ReadFile(filepath.Join(*inDir, e.Name(), f.Name()))
			die(err)
			var outcome decompositionlab.RecordOutcome
			die(json.Unmarshal(body, &outcome))
			byCondition[condition] = append(byCondition[condition], outcome)
		}
	}
	if len(byCondition) == 0 {
		die(fmt.Errorf("no raw records found under %s; run the `run` subcommand first", *inDir))
	}

	result := map[string]any{"schema": "tlaloc.exocortex-decomposition-t0.r0.results"}
	aggregates := map[string]decompositionlab.ConditionAggregate{}
	conditionNames := make([]string, 0, len(byCondition))
	for c := range byCondition {
		conditionNames = append(conditionNames, string(c))
	}
	sort.Strings(conditionNames)
	for _, name := range conditionNames {
		c := decompositionlab.Condition(name)
		aggregates[name] = decompositionlab.AggregateCondition(c, byCondition[c])
	}
	result["conditions"] = aggregates

	pair := func(from, to decompositionlab.Condition) *decompositionlab.Transition {
		fromRecords, fromOK := byCondition[from]
		toRecords, toOK := byCondition[to]
		if !fromOK || !toOK {
			return nil
		}
		t := decompositionlab.PairTransition(from, to, decompositionlab.IndexByBaseID(fromRecords), decompositionlab.IndexByBaseID(toRecords))
		return &t
	}
	transitions := map[string]*decompositionlab.Transition{
		"C0_to_C1": pair(decompositionlab.ConditionC0ParrotDirect, decompositionlab.ConditionC1OracleCrop),
		"C1_to_C2": pair(decompositionlab.ConditionC1OracleCrop, decompositionlab.ConditionC2Normalize),
		"C2_to_C3": pair(decompositionlab.ConditionC2Normalize, decompositionlab.ConditionC3Verify),
		"C0_to_C3": pair(decompositionlab.ConditionC0ParrotDirect, decompositionlab.ConditionC3Verify),
		"C1_vs_B1": pair(decompositionlab.ConditionC1OracleCrop, decompositionlab.ConditionB1RealCrop),
		"C2_vs_B2": pair(decompositionlab.ConditionC2Normalize, decompositionlab.ConditionB2Normalize),
		"C3_vs_B3": pair(decompositionlab.ConditionC3Verify, decompositionlab.ConditionB3Verify),
	}
	result["transitions"] = transitions

	die(os.MkdirAll(filepath.Dir(*outPath), 0o755))
	body, err := json.MarshalIndent(result, "", "  ")
	die(err)
	body = append(body, '\n')
	die(os.WriteFile(*outPath, body, 0o644))
	write(map[string]any{"results_path": *outPath, "conditions_aggregated": conditionNames})
}

func example() {
	spec := decompositionlab.Spec{
		DatasetPath: "results/T0_P0_IMAGE_DATASET.json", MicroISAArtifactPath: "results/PARROT_MICRO_ISA_R0.json",
		ExecutorID: "parrot-lfm2-vl-1.6b", ModelID: "lfm2-vl-1.6b", ProfileVersion: "r0",
		Endpoint: "http://127.0.0.1:1234/v1", MaxOutputTokens: 32, MarginRatio: 0.15,
		OutputDir: "runs/exocortex-decomposition-r0",
	}
	write(spec)
}

func parseConditions(raw string) []decompositionlab.Condition {
	var out []decompositionlab.Condition
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, decompositionlab.Condition(part))
	}
	return out
}

func write(v any) {
	body, err := json.MarshalIndent(v, "", "  ")
	die(err)
	fmt.Println(string(body))
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
