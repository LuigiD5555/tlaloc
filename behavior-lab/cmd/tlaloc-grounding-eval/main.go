package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/tlaloque/groundingautomaton"
)

type report struct {
	Cases   []groundingautomaton.EvalObservation `json:"cases"`
	Metrics groundingautomaton.EvalMetrics       `json:"metrics"`
}

func main() {
	input := flag.String("input", "testdata/grounding/core-r0.jsonl", "JSONL evaluation corpus")
	failOnFalseSupport := flag.Bool("fail-on-false-supported-contradiction", true, "exit non-zero when a contradiction is marked supported")
	flag.Parse()

	cases, err := readCases(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	observations, metrics := groundingautomaton.Evaluate(cases)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report{Cases: observations, Metrics: metrics}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *failOnFalseSupport && metrics.FalseSupportedContradiction > 0 {
		os.Exit(1)
	}
}

func readCases(path string) ([]groundingautomaton.EvalCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open grounding corpus: %w", err)
	}
	defer f.Close()

	cases := make([]groundingautomaton.EvalCase, 0)
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c groundingautomaton.EvalCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			return nil, fmt.Errorf("decode line %d: %w", line, err)
		}
		if c.ID == "" || c.Answer == "" || c.Evidence == "" || c.Expected == "" {
			return nil, fmt.Errorf("line %d: id, answer, evidence and expected are required", line)
		}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan grounding corpus: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("grounding corpus is empty")
	}
	return cases, nil
}
