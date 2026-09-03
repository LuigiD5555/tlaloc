package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/compiler"
	"tlaloc.local/behaviorlab/internal/distill"
	"tlaloc.local/behaviorlab/internal/spec"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "compile":
		compileCmd(os.Args[2:])
	case "train":
		trainCmd(os.Args[2:])
	case "tlaloque":
		tlaloqueCmd()
	case "receiver-distill":
		receiverDistillCmd(os.Args[2:])
	case "receiver-rank":
		receiverRankCmd(os.Args[2:])
	case "receiver-run":
		receiverRunCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "behaviorlab <compile|train|tlaloque|receiver-distill|receiver-rank|receiver-run> [flags]")
}

func loadSpec(path string) (spec.BehaviorSpec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return spec.BehaviorSpec{}, err
	}
	var s spec.BehaviorSpec
	err = json.Unmarshal(b, &s)
	return s, err
}

func compileCmd(args []string) {
	fs := flag.NewFlagSet("compile", flag.ExitOnError)
	in := fs.String("spec", "profiles/origami/quantum-inspired-r0.json", "behavior spec")
	out := fs.String("out", "generated/origami-quantum-inspired-r0.generic.prompt.md", "output prompt")
	targetName := fs.String("target", "generic", "target profile")
	_ = fs.Parse(args)
	s, err := loadSpec(*in)
	must(err)
	ir, err := compiler.BuildIR(s, *targetName)
	must(err)
	p := compiler.Render(ir)
	must(os.MkdirAll(filepath.Dir(*out), 0755))
	must(os.WriteFile(*out, []byte(p), 0644))
	fmt.Println(*out)
}

func trainCmd(args []string) {
	fs := flag.NewFlagSet("train", flag.ExitOnError)
	in := fs.String("spec", "profiles/origami/quantum-inspired-r0.json", "behavior spec")
	base := fs.String("endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible endpoint")
	model := fs.String("model", "", "model name")
	generations := fs.Int("generations", 6, "max generations")
	_ = fs.Parse(args)
	if *model == "" {
		fmt.Fprintln(os.Stderr, "-model is required")
		os.Exit(2)
	}
	s, err := loadSpec(*in)
	must(err)
	ir, err := compiler.BuildIR(s, *model)
	must(err)
	cases := []tlaloque.Case{
		{ID: "transform-no-collapse", User: `Initial state is SUPERPOSED with A=(0.7071067811865475,0), B=(0.7071067811865475,0). Apply TRANSFORM A->D and B->E, both multiplier (1,0). Return the resulting state JSON.`, ExpectedRaw: `{"kind":"superposed","branches":[{"label":"D","real":0.7071067811865475,"imag":0},{"label":"E","real":0.7071067811865475,"imag":0}]}`},
		{ID: "cancellation", User: `INTERFERE two contributions to C: +0.5 and -0.5. Return state JSON. This is not an observation.`, ExpectedRaw: `{"kind":"superposed","branches":[]}`},
		{ID: "coupled", User: `Create COUPLED members A,B with joint branches 00=(0.7071067811865475,0) and 11=(0.7071067811865475,0). Return state JSON.`, ExpectedRaw: `{"kind":"coupled","members":["A","B"],"branches":[{"label":"00","real":0.7071067811865475,"imag":0},{"label":"11","real":0.7071067811865475,"imag":0}]}`},
	}
	tr := tlaloque.Trainer{Model: target.OpenAICompat{BaseURL: *base, Model: *model, Temperature: 0}, MaxGenerations: *generations}
	h, finalIR, err := tr.Train(context.Background(), ir, cases)
	must(err)
	for _, g := range h {
		fmt.Printf("gen=%d score=%.3f pass=%d fail=%d patches=%d\n", g.Index, g.Score, g.Passed, g.Failed, len(g.Patches))
	}
	must(os.MkdirAll("generated", 0755))
	must(os.WriteFile("generated/origami-quantum-inspired-r0.trained.prompt.md", []byte(compiler.Render(finalIR)), 0644))
}

func receiverDistillCmd(args []string) {
	fs := flag.NewFlagSet("receiver-distill", flag.ExitOnError)
	tracePath := fs.String("trace", "testdata/receiver/swarm-trace-r0.json", "successful semantic swarm trace JSON")
	promptPath := fs.String("prompt", "testdata/receiver/universal-bootstrap-r0.md", "receiver prompt candidate")
	outPath := fs.String("out", "generated/origami-hybrid-receiver-r0.candidate.json", "legacy distilled candidate JSON")
	hybridOutPath := fs.String("hybrid-out", "generated/origami-hybrid-receiver-r0.artifact-set.json", "complete hybrid artifact-set JSON for Origami import")
	window := fs.Int("window", 4000, "maximum active model-facing token-equivalent")
	_ = fs.Parse(args)

	traceBytes, err := os.ReadFile(*tracePath)
	must(err)
	var trace []distill.SwarmStep
	must(json.Unmarshal(traceBytes, &trace))
	prompt, err := os.ReadFile(*promptPath)
	must(err)

	candidate, err := distill.Distill(string(prompt), trace)
	must(err)
	b, err := json.MarshalIndent(candidate, "", "  ")
	must(err)
	b = append(b, '\n')
	must(os.MkdirAll(filepath.Dir(*outPath), 0755))
	must(os.WriteFile(*outPath, b, 0644))

	hybrid, err := distill.BuildHybridArtifactSet(candidate, *window)
	must(err)
	hb, err := json.MarshalIndent(hybrid, "", "  ")
	must(err)
	hb = append(hb, '\n')
	must(os.MkdirAll(filepath.Dir(*hybridOutPath), 0755))
	must(os.WriteFile(*hybridOutPath, hb, 0644))

	fmt.Printf("CANDIDATE_ID=%s\n", candidate.ID)
	fmt.Printf("SOURCE_TRACE_SHA256=%s\n", candidate.SourceTraceSHA256)
	fmt.Printf("WORKING_WINDOW_TOKEN_EQ=%d\n", hybrid.WorkingWindow)
	fmt.Printf("CANDIDATE=%s\n", *outPath)
	fmt.Printf("HYBRID_ARTIFACT_SET=%s\n", *hybridOutPath)
}

func receiverRankCmd(args []string) {
	fs := flag.NewFlagSet("receiver-rank", flag.ExitOnError)
	inPath := fs.String("in", "testdata/receiver/scored-candidates-r0.json", "scored receiver candidates JSON")
	outPath := fs.String("out", "generated/origami-hybrid-receiver-r0.ranking.json", "ranked output JSON")
	window := fs.Int("window", 4000, "maximum active model-facing token-equivalent")
	_ = fs.Parse(args)

	b, err := os.ReadFile(*inPath)
	must(err)
	var candidates []distill.ScoredCandidate
	must(json.Unmarshal(b, &candidates))
	ranked, err := distill.Rank(candidates, *window)
	must(err)
	winner, err := distill.Winner(candidates, *window)
	must(err)

	out, err := json.MarshalIndent(ranked, "", "  ")
	must(err)
	out = append(out, '\n')
	must(os.MkdirAll(filepath.Dir(*outPath), 0755))
	must(os.WriteFile(*outPath, out, 0644))
	fmt.Printf("WINNER=%s\n", winner.Candidate.ID)
	fmt.Printf("SCORE=%.6f\n", winner.Score)
	fmt.Println(*outPath)
}

func receiverRunCmd(args []string) {
	fs := flag.NewFlagSet("receiver-run", flag.ExitOnError)
	endpoint := fs.String("endpoint", "http://127.0.0.1:1234/v1", "OpenAI-compatible endpoint")
	model := fs.String("model", "", "vision/tool-capable model identifier")
	promptPath := fs.String("prompt", "", "Origami public/MASTER_PROMPT.md")
	carrierPath := fs.String("carrier", "", "Origami public/carrier.png")
	packetPath := fs.String("packet", "", "Origami public/model_packet.json")
	origamiTool := fs.String("origami-tool", "origami-hybrid-tool", "Origami Hybrid tool executable")
	question := fs.String("question", "", "held-out receiver question")
	maxTurns := fs.Int("max-turns", 16, "maximum model/tool turns")
	outPath := fs.String("out", "", "optional JSON result path; stdout when empty")
	_ = fs.Parse(args)
	if *model == "" || *promptPath == "" || *carrierPath == "" || *packetPath == "" || *question == "" {
		fmt.Fprintln(os.Stderr, "receiver-run requires -model -prompt -carrier -packet -question")
		os.Exit(2)
	}
	prompt, err := os.ReadFile(*promptPath)
	must(err)
	carrier, err := os.ReadFile(*carrierPath)
	must(err)

	client := target.OpenAICompat{BaseURL: *endpoint, Model: *model, Temperature: 0}
	executor := target.OrigamiCLIExecutor{Binary: *origamiTool, Carrier: *carrierPath, Packet: *packetPath}
	result, err := client.CompleteHybrid(context.Background(), target.HybridInput{
		SystemPrompt: string(prompt),
		Question:     *question,
		ImagePNG:     carrier,
		Tools:        target.OrigamiHybridTools(),
		Executor:     executor,
		MaxTurns:     *maxTurns,
	})
	must(err)
	b, err := json.MarshalIndent(result, "", "  ")
	must(err)
	b = append(b, '\n')
	if *outPath == "" {
		fmt.Print(string(b))
		return
	}
	must(os.MkdirAll(filepath.Dir(*outPath), 0755))
	must(os.WriteFile(*outPath, b, 0644))
	fmt.Println(*outPath)
}

func tlaloqueCmd() {
	for _, agent := range tlaloque.DefaultTlaloque() {
		fmt.Println(agent.Name())
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
