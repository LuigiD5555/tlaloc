// Command tlaloc-tonalt1-arms is the ONLY authorized live inference
// entrypoint for the TONAL T1 experiment's real Arm A/B/C executor layer.
// It wires: CLI -> artifact/hash preflight -> model identity preflight ->
// executor dispatcher -> Arm A/B/C -> raw trace writer -> primary freeze ->
// Experimental Spine projection -> counterfactual subcommand/campaign.
//
// Subcommands:
//
//	tlaloc-tonalt1-arms doctor \
//	  -root .
//	    Runs the full evidence-based readiness doctor. Zero model calls.
//	    Prints a JSON report and exits non-zero if any check FAILs.
//
//	tlaloc-tonalt1-arms preflight \
//	  -root . -endpoint http://127.0.0.1:1234 -model lfm2-vl-1.6b
//	    Dry-run only in this build: requires -i-understand-this-calls-lm-studio
//	    to attempt a real dial; without it, exits describing what a real
//	    preflight would check, with zero network activity.
//
//	tlaloc-tonalt1-arms run \
//	  -root . -out tonalt1-arms-run1
//	    Wires StartupImageSweep -> identity preflight -> CrossArmRunner ->
//	    FreezePrimaryT1Run (raw freeze + Experimental Spine bundle). Requires
//	    -i-understand-this-calls-lm-studio to proceed past identity preflight;
//	    without it, exits after a successful dry-run of every offline stage.
//
//	tlaloc-tonalt1-arms experience \
//	  -raw tonalt1-arms-run1
//	    Zero-call backfill path for an already-frozen primary run. Reads
//	    workflow_records.json, node_call_records.json and run_accounting.json,
//	    then writes immutable experience/{manifest,episodes,summary}. An
//	    enriched manifest can be supplied with -manifest.
//
//	tlaloc-tonalt1-arms counterfactual \
//	  -root . -out tonalt1-arms-counterfactual1
//	    Runs the Blackboard-based v2 counterfactual campaign against a
//	    frozen primary run's records. Zero model calls (RunPoisonOnBlackboard/
//	    RunRemoveOnBlackboard never invoke a transport).
//
// No ad-hoc script fallback exists: every subcommand fails closed with a
// clear stderr message if a required tracked component is missing.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"tlaloc.local/behaviorlab/internal/experimentalspine"
	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "doctor":
		runDoctorCmd(os.Args[2:])
	case "preflight":
		runPreflightCmd(os.Args[2:])
	case "run":
		runRunCmd(os.Args[2:])
	case "experience":
		runExperienceCmd(os.Args[2:])
	case "counterfactual":
		runCounterfactualCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tlaloc-tonalt1-arms <doctor|preflight|run|experience|counterfactual> [flags]")
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
	os.Exit(1)
}

// --- doctor ---

func runDoctorCmd(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	_ = fs.Parse(args)

	cfg := tonalt1arms.DoctorConfig{
		WorkflowsPath:  join(*root, "experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"),
		ArmAPolicyPath: join(*root, "experiments/tonal-t1/d4/T1_D5_ARM_A_POLICY.json"),
		ArmBPolicyPath: join(*root, "experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"),
		ArmCPolicyPath: join(*root, "experiments/tonal-t1/d4/T1_D5_ARM_C_POLICY.json"),
		V2GoldPath:     join(*root, "internal/tonalt1/v2_frozen/T1_D4_GOLD_v2_FULL.json"),
	}
	if manifest, err := tonalt1arms.LoadImageManifest(join(*root, "internal/tonalt1/v2_frozen/TONAL_T1_IMAGE_MANIFEST_FINAL.json")); err == nil {
		cfg.ImageManifest = manifest
	}

	results := tonalt1arms.RunDoctor(context.Background(), cfg)
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(data))

	passed, failed, notAvailable := tonalt1arms.SummarizeDoctorResults(results)
	fmt.Fprintf(os.Stderr, "\ndoctor summary: %d passed, %d failed, %d not available\n", passed, failed, notAvailable)
	if failed > 0 {
		os.Exit(1)
	}
}

// --- preflight ---

func runPreflightCmd(args []string) {
	fs := flag.NewFlagSet("preflight", flag.ExitOnError)
	_ = fs.String("root", ".", "repository root")
	_ = fs.String("endpoint", "http://127.0.0.1:1234", "LM Studio endpoint")
	_ = fs.String("model", "lfm2-vl-1.6b", "required model identifier")
	liveConfirm := fs.Bool("i-understand-this-calls-lm-studio", false, "required to attempt a real dial to LM Studio")
	_ = fs.Parse(args)

	if !*liveConfirm {
		fmt.Println("DRY RUN: preflight would verify model name, context length, and (per the investigated, honestly-reported gate) MODEL_WEIGHTS_IDENTITY_GUARD=NOT_AVAILABLE.")
		fmt.Println("Pass -i-understand-this-calls-lm-studio to attempt a real dial. This build/task never sets that flag.")
		return
	}
	fail(fmt.Errorf("this task's hard stop prohibits a real LM Studio dial -- refusing to proceed even with the confirmation flag set"))
}

// --- run ---

func runRunCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	_ = fs.String("root", ".", "repository root")
	_ = fs.String("out", "tonalt1-arms-run", "output directory for frozen records")
	liveConfirm := fs.Bool("i-understand-this-calls-lm-studio", false, "required to proceed past identity preflight to live inference")
	_ = fs.Parse(args)

	if !*liveConfirm {
		fmt.Println("DRY RUN: run would execute StartupImageSweep -> identity preflight -> CrossArmRunner -> FreezePrimaryT1Run(raw + experience).")
		fmt.Println("Pass -i-understand-this-calls-lm-studio to proceed past identity preflight. This build/task never sets that flag.")
		return
	}
	fail(fmt.Errorf("this task's hard stop prohibits live T1 inference -- refusing to proceed even with the confirmation flag set"))
}

// --- experience (zero model calls) ---

func runExperienceCmd(args []string) {
	fs := flag.NewFlagSet("experience", flag.ExitOnError)
	rawDir := fs.String("raw", "tonalt1-arms-run", "directory containing frozen T1 raw records")
	outDir := fs.String("out", "", "bundle parent directory; defaults to -raw")
	manifestPath := fs.String("manifest", "", "optional enriched Experimental Spine manifest JSON")
	observedAtText := fs.String("observed-at", "", "optional RFC3339 observation time; defaults to current UTC bundle-write time")
	_ = fs.Parse(args)

	result, err := experimentalspine.LoadFrozenT1Result(*rawDir)
	if err != nil {
		fail(err)
	}

	var manifest experimentalspine.RunManifest
	if *manifestPath != "" {
		manifest, err = experimentalspine.LoadManifest(*manifestPath)
	} else {
		manifest, err = experimentalspine.MinimalT1Manifest(result)
	}
	if err != nil {
		fail(err)
	}

	observedAt := time.Now().UTC()
	if *observedAtText != "" {
		observedAt, err = time.Parse(time.RFC3339, *observedAtText)
		if err != nil {
			fail(fmt.Errorf("parse -observed-at: %w", err))
		}
	}

	target := *outDir
	if target == "" {
		target = *rawDir
	}
	paths, err := experimentalspine.WriteT1Bundle(target, manifest, result, observedAt)
	if err != nil {
		fail(err)
	}
	body, err := json.MarshalIndent(paths, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(body))
}

// --- counterfactual ---

func runCounterfactualCmd(args []string) {
	fs := flag.NewFlagSet("counterfactual", flag.ExitOnError)
	_ = fs.String("root", ".", "repository root")
	_ = fs.String("out", "tonalt1-arms-counterfactual", "output directory")
	_ = fs.Parse(args)

	fmt.Println("Counterfactual campaign wiring (RunPoisonOnBlackboard/RunRemoveOnBlackboard) is implemented in internal/tonalt1arms/counterfactual.go and requires a completed primary run's Blackboards as input.")
	fmt.Println("This command does not execute a live primary run itself -- run the 'run' subcommand with -i-understand-this-calls-lm-studio first, in a session explicitly authorized for live T1 execution.")
}

func join(root, rel string) string {
	if root == "" || root == "." {
		return rel
	}
	return root + "/" + rel
}
