// Command tlaloc-swarm-lmstudio-probe runs the intent and entity capabilities
// of the swarm-bench decomposition experiment against a real, locally
// resident model in LM Studio, and persists a model-tagged failure log.
//
// This is deliberately NOT part of `go test ./...` — it depends on an
// external local service that may or may not be running, mirroring how
// cmd/tlaloc-real-vlm-campaign is a manual campaign runner rather than a
// unit test. Run it explicitly.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"tlaloc.local/behaviorlab/internal/swarmbench"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "probe":
		probeCmd(os.Args[2:])
	case "baseline":
		baselineCmd(os.Args[2:])
	case "compare":
		compareCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tlaloc-swarm-lmstudio-probe <probe|baseline|compare> --model <id> --out <path.json> [--endpoint url] [--sample N]")
}

func probeCmd(args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	var endpoint, model, out, datasetID string
	var seed int64
	var count, sample int
	fs.StringVar(&endpoint, "endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL")
	fs.StringVar(&model, "model", "", "exact model id as advertised by GET /v1/models (required)")
	fs.StringVar(&out, "out", "", "output path for the probe log JSON (required)")
	fs.StringVar(&datasetID, "dataset-id", "lmstudio-probe", "dataset id")
	fs.Int64Var(&seed, "seed", 2026, "dataset generation seed")
	fs.IntVar(&count, "count", 500, "dataset item count to generate")
	fs.IntVar(&sample, "sample", 60, "how many of the generated items to actually send to the model")
	fs.Parse(args)
	if model == "" || out == "" {
		fmt.Fprintln(os.Stderr, "error: --model and --out are required")
		os.Exit(2)
	}

	dataset, err := swarmbench.Generate(datasetID, seed, count)
	die(err)
	client := swarmbench.LMStudioClient{Endpoint: endpoint, Model: model}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Fprintf(os.Stderr, "probing (per-capability) model=%q endpoint=%s sample=%d/%d ...\n", model, endpoint, sample, count)
	log := swarmbench.RunLMStudioProbe(ctx, client, dataset, sample)
	writeJSON(out, log)
	fmt.Fprintf(os.Stderr, "model=%s intent_accuracy=%.3f entity_accuracy=%.3f cases=%d\n", log.Model, log.IntentAccuracy, log.EntityAccuracy, len(log.Cases))
	fmt.Fprintf(os.Stderr, "log written to %s\n", out)
}

// baselineCmd runs the Phase 5 undecomposed condition: one model resolves
// every field in a single call, scored with the exact ScoreDataset the swarm
// itself uses — directly comparable to `tlaloc-swarm-bench run`'s Score.
func baselineCmd(args []string) {
	fs := flag.NewFlagSet("baseline", flag.ExitOnError)
	var endpoint, model, out, datasetID string
	var seed int64
	var count, sample int
	fs.StringVar(&endpoint, "endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL")
	fs.StringVar(&model, "model", "", "exact model id as advertised by GET /v1/models (required)")
	fs.StringVar(&out, "out", "", "output path for the baseline log JSON (required)")
	fs.StringVar(&datasetID, "dataset-id", "lmstudio-baseline", "dataset id")
	fs.Int64Var(&seed, "seed", 2026, "dataset generation seed")
	fs.IntVar(&count, "count", 500, "dataset item count to generate")
	fs.IntVar(&sample, "sample", 60, "how many of the generated items to actually send to the model")
	fs.Parse(args)
	if model == "" || out == "" {
		fmt.Fprintln(os.Stderr, "error: --model and --out are required")
		os.Exit(2)
	}

	dataset, err := swarmbench.Generate(datasetID, seed, count)
	die(err)
	client := swarmbench.LMStudioClient{Endpoint: endpoint, Model: model}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Fprintf(os.Stderr, "probing (single-shot, undecomposed) model=%q endpoint=%s sample=%d/%d ...\n", model, endpoint, sample, count)
	log := swarmbench.RunLMStudioBaseline(ctx, client, dataset, sample)
	writeJSON(out, log)
	fmt.Fprintf(os.Stderr, "model=%s exact_match_rate=%.3f route_accuracy=%.3f cases=%d\n", log.Model, log.Score.ExactMatchRate, log.Score.RouteAccuracy, len(log.Cases))
	fmt.Fprintf(os.Stderr, "log written to %s\n", out)
}

// compareCmd is Phase 5's head-to-head: the same live model, the same
// dataset sample, in one condition decomposed across the real five-Tlaloque
// swarm (intent and entity backed by real calls to the resident model, date/
// amount/route/verify deterministic) and in the other resolving every field
// in a single undecomposed call. Both conditions are scored with the exact
// same ScoreDataset, so the comparison is not approximated by a calibrated
// stand-in on either side.
func compareCmd(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	var endpoint, model, out, datasetID string
	var seed int64
	var count, sample int
	fs.StringVar(&endpoint, "endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL")
	fs.StringVar(&model, "model", "", "exact model id as advertised by GET /v1/models (required)")
	fs.StringVar(&out, "out", "", "output path for the comparison JSON (required)")
	fs.StringVar(&datasetID, "dataset-id", "lmstudio-compare", "dataset id")
	fs.Int64Var(&seed, "seed", 2026, "dataset generation seed")
	fs.IntVar(&count, "count", 500, "dataset item count to generate")
	fs.IntVar(&sample, "sample", 40, "how many of the generated items to actually send to the model")
	fs.Parse(args)
	if model == "" || out == "" {
		fmt.Fprintln(os.Stderr, "error: --model and --out are required")
		os.Exit(2)
	}

	dataset, err := swarmbench.Generate(datasetID, seed, count)
	die(err)
	if sample > 0 && sample < len(dataset.Items) {
		dataset.Items = dataset.Items[:sample]
	}
	client := swarmbench.LMStudioClient{Endpoint: endpoint, Model: model}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Fprintf(os.Stderr, "decomposed (live swarm): model=%q sample=%d ...\n", model, len(dataset.Items))
	registry, err := swarmbench.BuildInProcessRegistryWithLogic(1_600_000_000, 1_600_000_000, swarmbench.IntentLogicLive(client), swarmbench.EntityLogicLive(client))
	die(err)
	plan := swarmbench.BuildFanInPlan("compare-decomposed", 1)
	decomposedRun, err := swarmbench.Execute(ctx, registry, plan, dataset, swarmbench.FanInTerminalNode)
	die(err)
	fmt.Fprintf(os.Stderr, "  exact_match_rate=%.3f route_accuracy=%.3f\n", decomposedRun.Score.ExactMatchRate, decomposedRun.Score.RouteAccuracy)

	fmt.Fprintf(os.Stderr, "single-shot (undecomposed): model=%q sample=%d ...\n", model, len(dataset.Items))
	baseline := swarmbench.RunLMStudioBaseline(ctx, client, dataset, len(dataset.Items))
	fmt.Fprintf(os.Stderr, "  exact_match_rate=%.3f route_accuracy=%.3f\n", baseline.Score.ExactMatchRate, baseline.Score.RouteAccuracy)

	comparison := struct {
		Schema     string                 `json:"schema"`
		Model      string                 `json:"model"`
		Endpoint   string                 `json:"endpoint"`
		ItemCount  int                    `json:"item_count"`
		Decomposed swarmbench.Run         `json:"decomposed"`
		SingleShot swarmbench.BaselineLog `json:"single_shot"`
	}{
		Schema: "tlaloc.swarm-bench-lmstudio-compare.r0", Model: model, Endpoint: endpoint, ItemCount: len(dataset.Items),
		Decomposed: decomposedRun, SingleShot: baseline,
	}
	writeJSON(out, comparison)
	fmt.Fprintf(os.Stderr, "comparison written to %s\n", out)
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
