package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func main() {
	if len(os.Args) < 2 { usage(); os.Exit(2) }
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	switch os.Args[1] {
	case "validate": validate(os.Args[2:])
	case "catalog": catalog(os.Args[2:])
	case "run": run(ctx, os.Args[2:])
	case "example": example()
	default: usage(); os.Exit(2)
	}
}

func manifestFlag(name string, args []string) (string, *flag.FlagSet) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	var manifest string
	fs.StringVar(&manifest, "manifest", "", "tlaloque swarm manifest JSON")
	fs.Parse(args)
	if manifest == "" { die(fmt.Errorf("--manifest is required")) }
	return manifest, fs
}

func validate(args []string) {
	path, _ := manifestFlag("validate", args)
	m, err := tlaloque.LoadSwarmManifest(path); die(err)
	r, err := m.Registry(); die(err)
	write(map[string]any{"status":"PASS", "schema":m.Schema, "plan":m.Plan, "workers":r.Descriptors()})
}

func catalog(args []string) {
	path, _ := manifestFlag("catalog", args)
	m, err := tlaloque.LoadSwarmManifest(path); die(err)
	r, err := m.Registry(); die(err)
	write(map[string]any{"workers":r.Descriptors()})
}

func run(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var manifest, inputPath, taskID string
	fs.StringVar(&manifest, "manifest", "", "tlaloque swarm manifest JSON")
	fs.StringVar(&inputPath, "input", "", "task input JSON file")
	fs.StringVar(&taskID, "task", "", "task id")
	fs.Parse(args)
	if manifest == "" || inputPath == "" { die(fmt.Errorf("--manifest and --input are required")) }
	m, err := tlaloque.LoadSwarmManifest(manifest); die(err)
	r, err := m.Registry(); die(err)
	input, err := os.ReadFile(inputPath); die(err)
	if !json.Valid(input) { die(fmt.Errorf("input is not valid JSON")) }
	report, runErr := (tlaloque.SwarmRunner{Registry:r}).Run(ctx, m.Plan, taskID, json.RawMessage(input))
	if runErr != nil {
		write(map[string]any{"report":report, "error":runErr.Error()})
		os.Exit(1)
	}
	write(report)
}

func example() {
	m := tlaloque.SwarmManifest{
		Schema: tlaloque.SwarmManifestSchemaR0,
		Workers: []tlaloque.ProcessWorkerSpec{
			{Descriptor:tlaloque.CapabilityDescriptor{Schema:tlaloque.CapabilitySchemaR0, ID:"intent-general-r0", Capability:"DETECT_INTENT", Scope:tlaloque.ScopeGeneral, Engine:tlaloque.EngineModel, InputSchema:"tlaloc.text.r0", OutputSchema:"tlaloc.intent.r0", ParameterCount:18_000_000, MaxConcurrency:1}, Command:[]string{"python3","workers/intent.py"}, TimeoutMS:5000},
			{Descriptor:tlaloque.CapabilityDescriptor{Schema:tlaloque.CapabilitySchemaR0, ID:"entity-general-r0", Capability:"EXTRACT_ENTITY", Scope:tlaloque.ScopeGeneral, Engine:tlaloque.EngineModel, InputSchema:"tlaloc.text.r0", OutputSchema:"tlaloc.entities.r0", ParameterCount:20_000_000, MaxConcurrency:1}, Command:[]string{"python3","workers/entities.py"}, TimeoutMS:5000},
			{Descriptor:tlaloque.CapabilityDescriptor{Schema:tlaloque.CapabilitySchemaR0, ID:"router-r0", Capability:"ROUTE", Scope:tlaloque.ScopeGeneral, Engine:tlaloque.EngineDeterministic, InputSchema:"tlaloc.swarm-context.r0", OutputSchema:"tlaloc.route.r0", Deterministic:true, MaxConcurrency:8}, Command:[]string{"./workers/router"}, TimeoutMS:1000},
		},
		Plan:tlaloque.SwarmPlan{Schema:tlaloque.SwarmSchemaR0, ID:"micro-document-router-r0", MaxParallel:2, Nodes:[]tlaloque.SwarmNode{
			{ID:"intent", Capability:"DETECT_INTENT"},
			{ID:"entities", Capability:"EXTRACT_ENTITY"},
			{ID:"route", Capability:"ROUTE", DependsOn:[]string{"intent","entities"}, PreferDeterministic:true},
		}},
	}
	write(m)
}

func write(v any) {
	body, err := json.MarshalIndent(v, "", "  "); die(err)
	fmt.Println(string(body))
}
func die(err error) { if err != nil { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) } }
func usage() { fmt.Fprintln(os.Stderr, "usage: tlaloc-tlaloque-swarm <validate|catalog|run|example> [flags]") }
