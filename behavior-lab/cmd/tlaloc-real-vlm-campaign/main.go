package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tlaloc.local/behaviorlab/internal/realcampaign"
	"tlaloc.local/behaviorlab/internal/target"
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
	case "prepare":
		prepare(ctx, os.Args[2:])
	case "run":
		run(ctx, os.Args[2:])
	case "record-observation":
		recordObservation(os.Args[2:])
	case "example":
		example()
	default:
		usage()
		os.Exit(2)
	}
}

func flags(name string, args []string) (realcampaign.Spec, *flag.FlagSet) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	var s realcampaign.Spec
	fs.StringVar(&s.CampaignID, "id", "origami-temporal-real-vlm-r0", "campaign id")
	fs.StringVar(&s.Phase, "phase", "SMOKE", "SMOKE or EVIDENCE")
	fs.StringVar(&s.Endpoint, "endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL")
	fs.StringVar(&s.Model, "model", "", "exact model id; auto-selected only when endpoint reports exactly one model")
	fs.StringVar(&s.Compatibility, "compatibility", target.CompatibilityLMStudio, "multimodal payload strategy: lm-studio|openai|minimal")
	fs.StringVar(&s.TransportCondition, "transport-condition", "", "experimental transport identity: DIRECT_IMAGE_API|PLATFORM_MEDIATED|custom")
	fs.StringVar(&s.InteropMemoryRoot, "interop-memory", "", "persistent per-model working-configuration registry; defaults to XDG/local state")
	fs.StringVar(&s.APIKeyEnv, "api-key-env", "", "environment variable containing API key")
	fs.StringVar(&s.Program, "program", "", "canonical Origami signal-chain TemporalProgram JSON")
	fs.StringVar(&s.TemporalCarrier, "carrier", "origami-temporal-carrier", "Origami temporal carrier executable")
	fs.StringVar(&s.CandidateBuilder, "builder", "origami-candidate-build", "Origami candidate builder executable")
	fs.StringVar(&s.OutputDir, "out", "runs/real-vlm/origami-temporal-r0", "campaign output directory")
	fs.StringVar(&s.MasterPrompt, "master-prompt", "", "optional Origami Master Prompt for R4_ASSISTED evidence phase")
	fs.Float64Var(&s.Temperature, "temperature", 0, "target model temperature")
	fs.IntVar(&s.TimeoutSeconds, "timeout", 180, "per-call timeout seconds")
	fs.IntVar(&s.TransportRetries, "transport-retries", 1, "transport retries per call")
	fs.IntVar(&s.TrialsPerModel, "trials", 0, "trials per model; defaults to 1 smoke / 3 evidence")
	fs.IntVar(&s.CandidatesPerGen, "candidates-per-generation", 0, "candidate budget per generation")
	fs.IntVar(&s.MaxGenerations, "generations", 0, "maximum experimental generations")
	fs.Parse(args)
	return s, fs
}

func doctor(ctx context.Context, args []string) {
	s, _ := flags("doctor", args)
	r, err := realcampaign.Doctor(ctx, s)
	die(err)
	write(r)
}

func prepare(ctx context.Context, args []string) {
	s, _ := flags("prepare", args)
	r, err := realcampaign.Prepare(ctx, s)
	die(err)
	write(r)
}

func run(ctx context.Context, args []string) {
	s, _ := flags("run", args)
	prepared, report, err := realcampaign.Run(ctx, s)
	die(err)
	write(map[string]any{"prepared": prepared, "report": report})
}

func recordObservation(args []string) {
	fs := flag.NewFlagSet("record-observation", flag.ExitOnError)
	var model, compatibility, transport, endpoint, stage, outcome, responseFile, notes, memory string
	var temperature float64
	fs.StringVar(&model, "model", "", "exact model id")
	fs.StringVar(&compatibility, "compatibility", "platform", "observed compatibility/provider strategy label")
	fs.StringVar(&transport, "transport-condition", realcampaign.TransportPlatformMediated, "PLATFORM_MEDIATED|DIRECT_IMAGE_API|custom")
	fs.StringVar(&endpoint, "endpoint", "platform://manual", "observed endpoint/platform identity")
	fs.StringVar(&stage, "stage", "MANUAL_PLATFORM_OBSERVATION", "observation stage")
	fs.StringVar(&outcome, "outcome", "OBSERVED", "observed outcome; does not imply benchmark PASS")
	fs.StringVar(&responseFile, "response-file", "", "file containing the model response")
	fs.StringVar(&notes, "notes", "", "optional provenance notes")
	fs.StringVar(&memory, "interop-memory", "", "persistent per-model interoperability memory root")
	fs.Float64Var(&temperature, "temperature", 0, "observed temperature when known")
	fs.Parse(args)
	if responseFile == "" {
		die(fmt.Errorf("--response-file is required"))
	}
	body, err := os.ReadFile(responseFile)
	die(err)
	obs, err := realcampaign.BuildExternalObservation(model, compatibility, transport, endpoint, stage, outcome, string(body), notes, temperature)
	die(err)
	path, err := realcampaign.RecordExternalObservation(memory, obs)
	die(err)
	write(map[string]any{"path": path, "observation": obs})
}

func example() {
	write(realcampaign.Spec{
		Schema:             realcampaign.SpecSchema,
		CampaignID:         "origami-temporal-real-vlm-r0",
		Phase:              realcampaign.PhaseSmoke,
		Endpoint:           "http://127.0.0.1:1234/v1",
		Compatibility:      target.CompatibilityLMStudio,
		TransportCondition: realcampaign.TransportDirectImageAPI,
		Program:            "/path/to/origami/experiments/temporal-automaton-r0/signal-chain.json",
		TemporalCarrier:    "origami-temporal-carrier",
		CandidateBuilder:   "origami-candidate-build",
		OutputDir:          "runs/real-vlm/origami-temporal-r0",
		TimeoutSeconds:     180,
		TransportRetries:   1,
	})
}

func write(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	die(err)
	fmt.Println(string(b))
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tlaloc-real-vlm-campaign <doctor|prepare|run|record-observation|example> [flags]")
}
