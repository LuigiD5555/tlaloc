// Command tlaloc-swarm-bench is the driver for the decomposition-vs-
// replication experiment: generate the dataset, run the real five- or
// eight-Tlaloque swarm against it (in-process, or replicated), sweep
// populations, and emit a manifest wiring the real PROCESS/HTTP_JSON
// binaries for use with `tlaloc swarm run --manifest`.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/swarmbench"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "dataset":
		datasetCmd(os.Args[2:])
	case "run":
		runCmd(os.Args[2:])
	case "replicate":
		replicateCmd(os.Args[2:])
	case "sweep":
		sweepCmd(os.Args[2:])
	case "manifest":
		manifestCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tlaloc-swarm-bench <dataset|run|replicate|sweep|manifest> [flags]")
}

func datasetCmd(args []string) {
	fs := flag.NewFlagSet("dataset", flag.ExitOnError)
	var id, out string
	var seed int64
	var count int
	fs.StringVar(&id, "dataset-id", "swarm-bench", "dataset id")
	fs.Int64Var(&seed, "seed", 2026, "generation seed")
	fs.IntVar(&count, "count", 500, "item count")
	fs.StringVar(&out, "out", "", "output path (required)")
	fs.Parse(args)
	if out == "" {
		die(fmt.Errorf("--out is required"))
	}
	dataset, err := swarmbench.Generate(id, seed, count)
	die(err)
	die(dataset.Validate())
	writeJSON(out, dataset)
	fmt.Fprintf(os.Stderr, "dataset %q: %d items written to %s\n", id, count, out)
}

// swarmFlags is shared by run/replicate/sweep: which topology and which
// classifier profile to run the swarm with.
type swarmFlags struct {
	decomposed bool
	profile    string
}

func (f *swarmFlags) register(fs *flag.FlagSet) {
	fs.BoolVar(&f.decomposed, "decomposed", false, "use the 8-node decomposed topology instead of the 5-node baseline")
	fs.StringVar(&f.profile, "profile", "exhaustive", "intent/entity classifier profile: exhaustive or lfm2vl-proxy")
}

func (f swarmFlags) build() (*tlaloque.Registry, tlaloque.SwarmPlan, string, error) {
	var intentLogic swarmbench.IntentLogicFunc
	var intentParameters int64 = 12_000_000
	switch f.profile {
	case "exhaustive":
	case "lfm2vl-proxy":
		intentLogic = swarmbench.IntentWorkerLogicLFM2VLProxy
		intentParameters = 1_600_000_000
	default:
		return nil, tlaloque.SwarmPlan{}, "", fmt.Errorf("unknown --profile %q (want exhaustive or lfm2vl-proxy)", f.profile)
	}
	if f.decomposed {
		registry, err := swarmbench.BuildDecomposedRegistry(intentParameters, 4_000_000, 4_000_000)
		if err != nil {
			return nil, tlaloque.SwarmPlan{}, "", err
		}
		return registry, swarmbench.BuildDecomposedPlan("swarm-bench-decomposed", 8), swarmbench.FanInTerminalNode, nil
	}
	registry, err := swarmbench.BuildInProcessRegistryWithLogic(intentParameters, 18_000_000, intentLogic, nil)
	if err != nil {
		return nil, tlaloque.SwarmPlan{}, "", err
	}
	return registry, swarmbench.BuildFanInPlan("swarm-bench-baseline", 8), swarmbench.FanInTerminalNode, nil
}

func loadDataset(path string) swarmbench.Dataset {
	body, err := os.ReadFile(path)
	die(err)
	var dataset swarmbench.Dataset
	die(json.Unmarshal(body, &dataset))
	die(dataset.Validate())
	return dataset
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var datasetPath, out string
	var sf swarmFlags
	sf.register(fs)
	fs.StringVar(&datasetPath, "dataset", "", "dataset JSON path (required)")
	fs.StringVar(&out, "out", "", "output path (required)")
	fs.Parse(args)
	if datasetPath == "" || out == "" {
		die(fmt.Errorf("--dataset and --out are required"))
	}
	dataset := loadDataset(datasetPath)
	registry, plan, terminal, err := sf.build()
	die(err)
	run, err := swarmbench.Execute(context.Background(), registry, plan, dataset, terminal)
	die(err)
	writeJSON(out, run)
	fmt.Fprintf(os.Stderr, "exact_match_rate=%.4f route_accuracy=%.4f nodes=%d depth=%d edges=%d\n",
		run.Score.ExactMatchRate, run.Score.RouteAccuracy, run.Topology.Nodes, run.Topology.Depth, run.Topology.Edges)
}

func replicateCmd(args []string) {
	fs := flag.NewFlagSet("replicate", flag.ExitOnError)
	var datasetPath, out string
	var replicas int
	var sf swarmFlags
	sf.register(fs)
	fs.StringVar(&datasetPath, "dataset", "", "dataset JSON path (required)")
	fs.IntVar(&replicas, "replicas", 1, "concurrent replica count")
	fs.StringVar(&out, "out", "", "output path (required)")
	fs.Parse(args)
	if datasetPath == "" || out == "" {
		die(fmt.Errorf("--dataset and --out are required"))
	}
	dataset := loadDataset(datasetPath)
	registry, plan, terminal, err := sf.build()
	die(err)
	run, err := swarmbench.ExecuteReplicated(context.Background(), registry, plan, dataset, terminal, replicas)
	die(err)
	writeJSON(out, run)
	fmt.Fprintf(os.Stderr, "replicas=%d exact_match_rate=%.4f wall_clock_ms=%d items_per_second=%.1f\n",
		run.ReplicaCount, run.Score.ExactMatchRate, run.WallClockMS, run.ItemsPerSecond)
}

func sweepCmd(args []string) {
	fs := flag.NewFlagSet("sweep", flag.ExitOnError)
	var datasetPath, widths, outDir string
	var sf swarmFlags
	sf.register(fs)
	fs.StringVar(&datasetPath, "dataset", "", "dataset JSON path (required)")
	fs.StringVar(&widths, "widths", "1,2,4,8,16,32,64,128", "comma-separated replica counts to sweep")
	fs.StringVar(&outDir, "out-dir", "", "directory to write one JSON file per width plus summary.json (required)")
	fs.Parse(args)
	if datasetPath == "" || outDir == "" {
		die(fmt.Errorf("--dataset and --out-dir are required"))
	}
	die(os.MkdirAll(outDir, 0o755))
	dataset := loadDataset(datasetPath)
	registry, plan, terminal, err := sf.build()
	die(err)

	type summaryRow struct {
		Replicas       int     `json:"replicas"`
		WallClockMS    int64   `json:"wall_clock_ms"`
		ItemsPerSecond float64 `json:"items_per_second"`
		ExactMatchRate float64 `json:"exact_match_rate"`
		RouteAccuracy  float64 `json:"route_accuracy"`
	}
	summary := struct {
		Schema     string              `json:"schema"`
		Decomposed bool                `json:"decomposed"`
		Profile    string              `json:"profile"`
		Topology   swarmbench.Topology `json:"topology"`
		Rows       []summaryRow        `json:"rows"`
		StartedAt  time.Time           `json:"started_at"`
	}{
		Schema: "tlaloc.swarm-bench-sweep-summary.r0", Decomposed: sf.decomposed, Profile: sf.profile,
		Topology: swarmbench.AnalyzeTopology(plan), StartedAt: time.Now().UTC(),
	}

	for _, widthStr := range strings.Split(widths, ",") {
		width, convErr := strconv.Atoi(strings.TrimSpace(widthStr))
		if convErr != nil || width <= 0 {
			die(fmt.Errorf("invalid width %q", widthStr))
		}
		run, runErr := swarmbench.ExecuteReplicated(context.Background(), registry, plan, dataset, terminal, width)
		die(runErr)
		writeJSON(filepath.Join(outDir, fmt.Sprintf("width-%03d.json", width)), run)
		summary.Rows = append(summary.Rows, summaryRow{
			Replicas: width, WallClockMS: run.WallClockMS, ItemsPerSecond: run.ItemsPerSecond,
			ExactMatchRate: run.Score.ExactMatchRate, RouteAccuracy: run.Score.RouteAccuracy,
		})
		fmt.Fprintf(os.Stderr, "width=%d wall_clock_ms=%d items_per_second=%.1f exact_match_rate=%.4f\n",
			width, run.WallClockMS, run.ItemsPerSecond, run.Score.ExactMatchRate)
	}
	writeJSON(filepath.Join(outDir, "summary.json"), summary)
	fmt.Fprintf(os.Stderr, "sweep complete: %s\n", outDir)
}

// manifestCmd emits a real SwarmManifest wiring the actual PROCESS/HTTP_JSON
// binaries built alongside this one — tlaloc-swarm-worker for the three
// deterministic Tlaloques, tlaloc-swarm-model-server (started separately,
// resident) for intent/entity — so the result is directly runnable with
// `tlaloc swarm run --manifest <path> --input <item.json>` against the real
// generic swarm runtime, not just this experiment's in-process shortcut.
func manifestCmd(args []string) {
	fs := flag.NewFlagSet("manifest", flag.ExitOnError)
	var out, workerPath, intentEndpoint, entityEndpoint string
	fs.StringVar(&out, "out", "", "output path (required)")
	fs.StringVar(&workerPath, "worker-path", "", "path to the installed tlaloc-swarm-worker binary (defaults to this binary's own directory)")
	fs.StringVar(&intentEndpoint, "intent-endpoint", "http://127.0.0.1:9101/infer", "resident intent model-server endpoint")
	fs.StringVar(&entityEndpoint, "entity-endpoint", "http://127.0.0.1:9102/infer", "resident entity model-server endpoint")
	fs.Parse(args)
	if out == "" {
		die(fmt.Errorf("--out is required"))
	}
	if workerPath == "" {
		executable, err := os.Executable()
		die(err)
		workerPath = filepath.Join(filepath.Dir(executable), "tlaloc-swarm-worker")
	}

	manifest := tlaloque.SwarmManifest{
		Schema: tlaloque.SwarmManifestSchemaR0,
		Workers: []tlaloque.WorkerSpec{
			{Descriptor: tlaloque.CapabilityDescriptor{Schema: tlaloque.CapabilitySchemaR0, ID: "intent-lexicon-r0", Capability: swarmbench.CapabilityDetectIntent, Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineModel, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.intent.r0", ParameterCount: 12_000_000, MaxConcurrency: 4}, Transport: tlaloque.TransportHTTPJSON, Endpoint: intentEndpoint, TimeoutMS: 5000},
			{Descriptor: tlaloque.CapabilityDescriptor{Schema: tlaloque.CapabilitySchemaR0, ID: "entity-gazetteer-r0", Capability: swarmbench.CapabilityExtractEntity, Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineModel, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.entity.r0", ParameterCount: 18_000_000, MaxConcurrency: 4}, Transport: tlaloque.TransportHTTPJSON, Endpoint: entityEndpoint, TimeoutMS: 5000},
			{Descriptor: tlaloque.CapabilityDescriptor{Schema: tlaloque.CapabilitySchemaR0, ID: "date-number-r0", Capability: swarmbench.CapabilityResolveDateAmount, Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.text.r0", OutputSchema: "tlaloc.date-amount.r0", Deterministic: true, MaxConcurrency: 8}, Transport: tlaloque.TransportProcess, Command: []string{workerPath, "date-number"}, TimeoutMS: 2000},
			{Descriptor: tlaloque.CapabilityDescriptor{Schema: tlaloque.CapabilitySchemaR0, ID: "router-r0", Capability: swarmbench.CapabilityRoute, Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.swarm-context.r0", OutputSchema: "tlaloc.fields.r0", Deterministic: true, MaxConcurrency: 8, Dependencies: []string{swarmbench.CapabilityDetectIntent, swarmbench.CapabilityExtractEntity, swarmbench.CapabilityResolveDateAmount}}, Transport: tlaloque.TransportProcess, Command: []string{workerPath, "router"}, TimeoutMS: 2000},
			{Descriptor: tlaloque.CapabilityDescriptor{Schema: tlaloque.CapabilitySchemaR0, ID: "verifier-r0", Capability: swarmbench.CapabilityVerify, Scope: tlaloque.ScopeGeneral, Engine: tlaloque.EngineDeterministic, InputSchema: "tlaloc.fields.r0", OutputSchema: "tlaloc.fields.r0", Deterministic: true, MaxConcurrency: 8, Dependencies: []string{swarmbench.CapabilityRoute}}, Transport: tlaloque.TransportProcess, Command: []string{workerPath, "verifier"}, TimeoutMS: 2000},
		},
		Plan: tlaloque.SwarmPlan{Schema: tlaloque.SwarmSchemaR0, ID: "swarm-bench-real-transport", MaxParallel: 4, Nodes: []tlaloque.SwarmNode{
			{ID: "intent", Capability: swarmbench.CapabilityDetectIntent},
			{ID: "entity", Capability: swarmbench.CapabilityExtractEntity},
			{ID: "date-number", Capability: swarmbench.CapabilityResolveDateAmount},
			{ID: "route", Capability: swarmbench.CapabilityRoute, DependsOn: []string{"intent", "entity", "date-number"}},
			{ID: "verify", Capability: swarmbench.CapabilityVerify, DependsOn: []string{"route"}},
		}},
	}
	writeJSON(out, manifest)
	fmt.Fprintf(os.Stderr, "manifest written to %s\n", out)
	fmt.Fprintln(os.Stderr, "start the resident model servers first, e.g.:")
	fmt.Fprintf(os.Stderr, "  tlaloc-swarm-model-server --capability DETECT_INTENT --addr 127.0.0.1:9101 &\n")
	fmt.Fprintf(os.Stderr, "  tlaloc-swarm-model-server --capability EXTRACT_ENTITY --addr 127.0.0.1:9102 &\n")
	fmt.Fprintf(os.Stderr, "then: tlaloc swarm run --manifest %s --input <item.json>\n", out)
}

func writeJSON(path string, v any) {
	body, err := json.MarshalIndent(v, "", "  ")
	die(err)
	die(os.WriteFile(path, append(body, '\n'), 0o644))
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
