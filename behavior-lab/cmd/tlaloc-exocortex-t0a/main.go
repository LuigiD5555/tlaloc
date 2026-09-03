// Command tlaloc-exocortex-t0a runs T0-A (CONTROLLED EXTERNAL COMPOSITION
// R0). `generate` freezes the new T0-A dataset; `doctor` checks readiness;
// `run` executes conditions D0-D3 against a real endpoint; `aggregate`
// summarizes. It never fabricates a record.
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

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/t0alab"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	switch os.Args[1] {
	case "generate":
		generate(os.Args[2:])
	case "doctor":
		doctor(os.Args[2:])
	case "run":
		run(ctx, os.Args[2:])
	case "aggregate":
		aggregate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `tlaloc-exocortex-t0a <generate|doctor|run|aggregate> [flags]

  generate  --seed N --count 40 --out experiments/exocortex-t0a-r0
            renders + freezes the T0-A controlled-composition dataset.
  doctor    --dataset <t0a_dataset.json> --p2-experiment <artifact> [--endpoint URL --model-id ID]
  run       --dataset <t0a_dataset.json> --p2-experiment <artifact> --endpoint URL --model-id ID
            --conditions D0_DIRECT_TWO_OP,D1_EXTERNAL_SEQUENCING,D2_EXTERNAL_OP1,D3_NORMALIZE_VERIFY
            --out runs/exocortex-t0a-r0
  aggregate --in <run dir> --dataset <t0a_dataset.json> --out results/EXOCORTEX_T0A_R0.json`)
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func write(v any) {
	body, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(body))
}

func generate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	seed := fs.Int64("seed", 20260903, "deterministic seed")
	count := fs.Int("count", 40, "number of base stimuli")
	out := fs.String("out", "experiments/exocortex-t0a-r0", "output dir")
	fs.Parse(args)
	datasetDir := filepath.Join(*out, "datasets")
	dataset, hash, err := parrotlab.GenerateT0A(*seed, *count, datasetDir)
	die(err)
	write(map[string]any{
		"dataset_path":   filepath.Join(datasetDir, "t0a_dataset.json"),
		"dataset_sha256": hash, "count": dataset.Count, "seed": dataset.Seed, "task_family": dataset.TaskFamily,
	})
}

func compileProfile(p2 string) exocortex.CapabilityProfile {
	profile, err := exocortex.CompileParrotProfileReal(p2, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	die(err)
	die(exocortex.VerifySourceArtifact(profile))
	return profile
}

func doctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	datasetPath := fs.String("dataset", "", "t0a_dataset.json")
	p2 := fs.String("p2-experiment", "", "frozen P2-A artifact")
	fs.Parse(args)
	dataset, hash, err := parrotlab.LoadT0ADataset(*datasetPath)
	die(err)
	profile := compileProfile(*p2)
	write(map[string]any{
		"dataset_sha256": hash, "records": dataset.Count,
		"profile_id": profile.ProfileID, "max_safe_ops": profile.MaxSafeOps,
		"extract_number_deployable": deployable(profile, exocortex.OpExtractNumber),
		"ready":                     dataset.Count > 0 && deployable(profile, exocortex.OpExtractNumber),
	})
}

func deployable(p exocortex.CapabilityProfile, op string) bool {
	e, ok := p.Entry(op)
	return ok && (e.DeploymentRecommendation == exocortex.DeploymentDeploy || e.DeploymentRecommendation == exocortex.DeploymentDeployConstrained)
}

func run(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	datasetPath := fs.String("dataset", "", "t0a_dataset.json")
	p2 := fs.String("p2-experiment", "", "frozen P2-A artifact")
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL")
	modelID := fs.String("model-id", "lfm2-vl-1.6b", "model id")
	condsFlag := fs.String("conditions", "D0_DIRECT_TWO_OP,D1_EXTERNAL_SEQUENCING,D2_EXTERNAL_OP1,D3_NORMALIZE_VERIFY", "comma-separated conditions")
	maxTok := fs.Int("max-output-tokens", 16, "max generated tokens per Parrot call")
	out := fs.String("out", "runs/exocortex-t0a-r0", "output dir")
	fs.Parse(args)

	dataset, _, err := parrotlab.LoadT0ADataset(*datasetPath)
	die(err)
	profile := compileProfile(*p2)
	endpointCfg := exocortex.ParrotEndpoint{BaseURL: *endpoint, Model: *modelID, MaxTokens: *maxTok}
	registry, err := t0alab.NewRegistry(profile, endpointCfg)
	die(err)

	datasetDir := filepath.Dir(*datasetPath)
	cfg := t0alab.Config{Profile: profile, Endpoint: endpointCfg, Store: blackboard.New(filepath.Join(*out, "blackboard")), DatasetDir: datasetDir, MaxOutTok: *maxTok}

	rawDir := filepath.Join(*out, "raw")
	die(os.MkdirAll(rawDir, 0o755))
	total := 0
	for _, condRaw := range strings.Split(*condsFlag, ",") {
		condition := t0alab.Condition(strings.TrimSpace(condRaw))
		condDir := filepath.Join(rawDir, string(condition))
		die(os.MkdirAll(condDir, 0o755))
		for _, record := range dataset.Records {
			select {
			case <-ctx.Done():
				die(ctx.Err())
			default:
			}
			outcome := t0alab.RunStimulus(ctx, cfg, registry, record, condition)
			body, _ := json.MarshalIndent(outcome, "", "  ")
			die(os.WriteFile(filepath.Join(condDir, record.ID+".json"), body, 0o644))
			total++
			fmt.Fprintf(os.Stderr, "%s/%s contract=%v semantic=%v err=%q\n", condition, record.ID, outcome.ContractSuccess, outcome.SemanticCorrect, outcome.Error)
		}
	}
	write(map[string]any{"raw_dir": rawDir, "records_run": total})
}

func aggregate(args []string) {
	fs := flag.NewFlagSet("aggregate", flag.ExitOnError)
	inDir := fs.String("in", "", "run dir (the run command's --out/raw)")
	datasetPath := fs.String("dataset", "", "t0a_dataset.json")
	outPath := fs.String("out", "results/EXOCORTEX_T0A_R0.json", "results JSON path")
	fs.Parse(args)
	_, datasetHash, err := parrotlab.LoadT0ADataset(*datasetPath)
	die(err)

	byCondition := map[t0alab.Condition][]t0alab.StimulusOutcome{}
	entries, err := os.ReadDir(*inDir)
	die(err)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		condition := t0alab.Condition(e.Name())
		files, err := os.ReadDir(filepath.Join(*inDir, e.Name()))
		die(err)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			body, err := os.ReadFile(filepath.Join(*inDir, e.Name(), f.Name()))
			die(err)
			var outcome t0alab.StimulusOutcome
			die(json.Unmarshal(body, &outcome))
			byCondition[condition] = append(byCondition[condition], outcome)
		}
	}
	if len(byCondition) == 0 {
		die(fmt.Errorf("no raw outcomes under %s", *inDir))
	}
	results := t0alab.Aggregate(datasetHash, byCondition)
	die(os.MkdirAll(filepath.Dir(*outPath), 0o755))
	body, _ := json.MarshalIndent(results, "", "  ")
	die(os.WriteFile(*outPath, append(body, '\n'), 0o644))
	write(map[string]any{"results_path": *outPath})
}
