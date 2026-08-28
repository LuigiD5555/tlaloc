package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/compiler"
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
	default:
		usage()
		os.Exit(2)
	}
}

func usage() { fmt.Fprintln(os.Stderr, "behaviorlab <compile|train|tlaloque> [flags]") }

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
