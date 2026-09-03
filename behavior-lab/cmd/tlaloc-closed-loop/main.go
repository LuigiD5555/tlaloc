package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tlaloc.local/behaviorlab/internal/closedloop"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "run":
		run(os.Args[2:])
	case "example":
		example()
	default:
		usage()
		os.Exit(2)
	}
}

func validate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	path := fs.String("config", "", "closed-loop config JSON")
	fs.Parse(args)
	cfg := load(*path)
	if cfg.AutoCandidates {
		die(closedloop.ValidateAutoReady(context.Background(), cfg))
	} else {
		die(closedloop.ValidateReady(cfg))
	}
	fmt.Println("CLOSED_LOOP_READY")
}

func run(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	path := fs.String("config", "", "closed-loop config JSON")
	fs.Parse(args)
	cfg := load(*path)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var report closedloop.Report
	var err error
	if cfg.AutoCandidates {
		report, err = closedloop.RunAuto(ctx, cfg)
	} else {
		report, err = closedloop.Run(ctx, cfg)
	}
	die(err)
	write(report)
}

func load(path string) closedloop.Config {
	if path == "" {
		die(fmt.Errorf("-config is required"))
	}
	b, err := os.ReadFile(path)
	die(err)
	var cfg closedloop.Config
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	die(dec.Decode(&cfg))
	return cfg
}

func example() {
	write(closedloop.Config{
		Schema:                        closedloop.ConfigSchema,
		RunID:                         "temporal-r0-local-001",
		BenchmarkID:                   "origami-temporal-native-r0",
		OutputDir:                     "runs/closed-loop/temporal-r0-local-001",
		MemoryRoot:                    "",
		MasterPrompt:                  "/path/to/origami/generated/MASTER_PROMPT.md",
		OrigamiVersion:                "6.0.0-alpha.15-candidate",
		TlalocVersion:                 "6.0.0-alpha.20-candidate",
		TrialsPerModel:                1,
		CandidatesPerGeneration:       2,
		MaxGenerations:                3,
		MinIncumbentImprovement:       0.01,
		ContinueExplorationWhenStable: false,
		DiagnosticRetries:             true,
		Conditions:                    []string{"NATIVE_PNG_ONLY", "R4_ASSISTED"},
		OutcomeMetric:                 closedloop.OutcomeNative,
		Models: []closedloop.ModelConfig{{
			Name:             "lmstudio-vlm",
			Provider:         "OPENAI_COMPAT",
			BaseURL:          "http://127.0.0.1:1234/v1",
			Model:            "REPLACE_WITH_VISION_MODEL",
			TimeoutSeconds:   180,
			TransportRetries: 1,
		}},
		Baseline:                    closedloop.SpecimenConfig{ID: "signal-chain-r0", PNG: "/path/to/origami-temporal-signal-chain-r0.png"},
		AutoCandidates:              true,
		CandidateBuilder:            []string{"origami-candidate-build"},
		AutoCandidateBaseProfileID:  "origami.temporal-carrier.r0.profile-1",
		AutoCandidatesPerGeneration: 4,
	})
}

func write(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	die(err)
	fmt.Println(string(b))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tlaloc-closed-loop <validate|run|example> [flags]")
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
