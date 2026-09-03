// Command tlaloc-parrot-capability-lab runs the Tlaloc Phase P experiment:
// it freezes an experiment directory, runs its stages against a small
// multimodal model, aggregates the results, and builds the competence
// envelope. See experiments/parrot-capability-r0/SPEC.md.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"tlaloc.local/behaviorlab/internal/parrotlab"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "doctor":
		doctor(ctx, os.Args[2:])
	case "generate":
		generate(os.Args[2:])
	case "validate":
		validate(os.Args[2:])
	case "audit":
		audit(os.Args[2:])
	case "freeze":
		freeze(os.Args[2:])
	case "run":
		run(ctx, os.Args[2:])
	case "aggregate":
		aggregate(os.Args[2:])
	case "report":
		report(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `tlaloc-parrot-capability-lab <command>

  doctor    --experiment DIR                          check the model endpoint
  generate  --experiment DIR --stage instruction_cliff [--seed N] [--scenes N] [--force]
  generate  --experiment DIR --stage end_to_end --store STORE_DIR [--pdf FILE] [--seed N]
                                                       deterministically build a stage dataset (+images)
  validate  --experiment DIR [--stage NAME]            validate dataset(s)
  audit     --experiment DIR                           write datasets/P0_AUDIT.md, check the freeze gate
  freeze    --experiment DIR --scope global            lock prompt + model config (once)
  freeze    --experiment DIR --scope stage --stage X   lock one stage dataset (never rewritten)
  run       --experiment DIR --stage NAME              run a stage against the model
            [--repetitions N] [--sentinel-only] [--timeout SECONDS] [--allow-unfrozen]
  aggregate --experiment DIR --stage NAME              build results/<stage>.json
  report    --experiment DIR                           build the competence envelope
`)
}

func load(args []string) (*parrotlab.Experiment, *flag.FlagSet) {
	fs := flag.NewFlagSet("parrotlab", flag.ExitOnError)
	dir := fs.String("experiment", "", "experiment directory")
	fs.String("stage", "", "stage id")
	fs.Int("repetitions", 1, "repetitions per case")
	fs.Bool("sentinel-only", false, "run only cases marked sentinel")
	fs.Int("timeout", 180, "per-call timeout seconds")
	fs.String("dataset", "", "optional dataset path override (ad-hoc runs before freeze)")
	fs.Bool("allow-unfrozen", false, "skip the stage-freeze gate (smoke runs only)")
	fs.String("scope", "", "freeze scope: global | stage")
	fs.Int64("seed", 42, "generator seed")
	fs.Int("scenes", 40, "generator scene count")
	fs.Bool("force", false, "overwrite an existing generated dataset")
	fs.Bool("write-model", false, "doctor: fill MODEL.json identity fields from a live probe (pre-freeze)")
	fs.String("store", "", "pdfmemory store directory (P0 end_to_end source)")
	fs.String("pdf", "", "source PDF for page rasterising (defaults to the store's source object)")
	fs.String("pages", "", "P0 end_to_end: explicit comma-separated page list (recommended for OCR'd sources)")
	fs.Parse(args)
	if *dir == "" {
		die(fmt.Errorf("--experiment is required"))
	}
	exp, err := parrotlab.Load(*dir)
	die(err)
	return exp, fs
}

func doctor(ctx context.Context, args []string) {
	exp, fs := load(args)
	report, err := parrotlab.Doctor(ctx, exp)
	die(err)
	if fs.Lookup("write-model").Value.String() == "true" {
		changed, writeErr := parrotlab.WriteModelIdentity(ctx, exp)
		die(writeErr)
		emit(map[string]any{"doctor": report, "model_json_updated": changed})
		return
	}
	emit(report)
}

func validate(args []string) {
	exp, fs := load(args)
	stage := fs.Lookup("stage").Value.String()
	stages := parrotlab.StageOrder
	if stage != "" {
		stages = []string{stage}
	}
	total := 0
	for _, name := range stages {
		path, err := exp.StageDataset(name)
		if err != nil {
			continue
		}
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}
		cases, err := parrotlab.LoadCases(path)
		die(err)
		problems := parrotlab.Validate(cases)
		if name == parrotlab.StageEndToEnd {
			problems = append(problems, parrotlab.ValidateEndToEnd(cases)...)
		}
		total += len(problems)
		for _, problem := range problems {
			fmt.Printf("%s: %s\n", name, problem)
		}
		fmt.Printf("%s: %d cases, %d problem(s)\n", name, len(cases), len(problems))
	}
	if total > 0 {
		os.Exit(1)
	}
}

func generate(args []string) {
	exp, fs := load(args)
	stage := fs.Lookup("stage").Value.String()
	datasetDir := filepath.Join(exp.Root, "datasets")
	seed := int64(mustAtoi(fs.Lookup("seed").Value.String()))

	switch stage {
	case parrotlab.StageInstructionCliff:
		scenes := mustAtoi(fs.Lookup("scenes").Value.String())
		force := fs.Lookup("force").Value.String() == "true"
		written, err := parrotlab.GenerateInstructionCliff(datasetDir, seed, scenes, force)
		die(err)
		emit(map[string]any{"stage": stage, "scenes": scenes, "cases_written": written,
			"dataset": filepath.Join(datasetDir, "instruction-cliff.jsonl")})
	case parrotlab.StageEndToEnd:
		store := fs.Lookup("store").Value.String()
		if store == "" {
			die(fmt.Errorf("--store <pdfmemory store dir> is required for --stage end_to_end"))
		}
		provider, err := parrotlab.NewPDFMemoryProvider(store, fs.Lookup("pdf").Value.String())
		die(err)
		var pageList []int
		for _, token := range strings.Split(fs.Lookup("pages").Value.String(), ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			pageList = append(pageList, mustAtoi(token))
		}
		report, err := parrotlab.GenerateEndToEnd(provider, datasetDir, parrotlab.P0Options{Seed: seed, Pages: pageList})
		die(err)
		cases, loadErr := parrotlab.LoadCases(report.Dataset)
		die(loadErr)
		problems := parrotlab.ValidateEndToEnd(cases)
		messages := make([]string, 0, len(problems))
		for _, problem := range problems {
			messages = append(messages, problem.Error())
		}
		emit(map[string]any{"report": report, "validation_problems": messages})
		if len(problems) > 0 {
			os.Exit(1)
		}
	default:
		die(fmt.Errorf("generate supports --stage instruction_cliff | end_to_end"))
	}
}

func audit(args []string) {
	exp, _ := load(args)
	path, ready, err := parrotlab.WriteP0Audit(exp)
	die(err)
	emit(map[string]any{"audit_file": path, "gate_green": ready})
	if !ready {
		os.Exit(1)
	}
}

func freeze(args []string) {
	exp, fs := load(args)
	scope := fs.Lookup("scope").Value.String()
	switch scope {
	case "global":
		frozen, err := parrotlab.FreezeGlobal(exp)
		die(err)
		emit(map[string]any{"scope": "global", "frozen": frozen})
	case "stage":
		stage := fs.Lookup("stage").Value.String()
		if stage == "" {
			die(fmt.Errorf("--stage is required with --scope stage"))
		}
		frozen, err := parrotlab.FreezeStage(exp, stage)
		die(err)
		emit(map[string]any{"scope": "stage", "stage": stage, "frozen": frozen})
	default:
		die(fmt.Errorf("--scope must be global or stage"))
	}
}

func run(ctx context.Context, args []string) {
	exp, fs := load(args)
	stage := fs.Lookup("stage").Value.String()
	if stage == "" {
		die(fmt.Errorf("--stage is required"))
	}
	report, err := parrotlab.RunStage(ctx, exp, parrotlab.RunOptions{
		Stage:          stage,
		DatasetPath:    fs.Lookup("dataset").Value.String(),
		Repetitions:    mustAtoi(fs.Lookup("repetitions").Value.String()),
		SentinelOnly:   fs.Lookup("sentinel-only").Value.String() == "true",
		TimeoutSeconds: mustAtoi(fs.Lookup("timeout").Value.String()),
		AllowUnfrozen:  fs.Lookup("allow-unfrozen").Value.String() == "true",
	})
	die(err)
	emit(report)
}

func aggregate(args []string) {
	exp, fs := load(args)
	stage := fs.Lookup("stage").Value.String()
	if stage == "" {
		die(fmt.Errorf("--stage is required"))
	}
	result, err := parrotlab.Aggregate(exp, stage)
	die(err)
	path, err := parrotlab.WriteStageResult(exp, result)
	die(err)
	emit(map[string]any{"stage": stage, "result_file": path, "summary": result.Summary})
}

func report(args []string) {
	exp, _ := load(args)
	envelope, err := parrotlab.BuildEnvelope(exp)
	die(err)
	path, err := parrotlab.WriteEnvelope(exp, envelope)
	die(err)
	emit(map[string]any{"envelope_file": path, "envelope": envelope})
}

func mustAtoi(text string) int {
	var value int
	fmt.Sscanf(text, "%d", &value)
	return value
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	die(encoder.Encode(value))
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
