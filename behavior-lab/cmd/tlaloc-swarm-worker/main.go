// Command tlaloc-swarm-worker is the PROCESS-transport binary for the three
// deterministic, zero-parameter Tlaloques in the swarm-bench decomposition
// experiment: date-number, router and verifier. It is a thin stdin/stdout
// adapter over internal/swarmbench's pure logic — the exact functions Phase 2
// unit-tests directly, so the binary provably behaves the way the tests say.
//
// Contract: one CapabilityRequest JSON document on stdin, one
// CapabilityResponse JSON document on stdout, selected by the single
// argument naming which Tlaloque to run.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"tlaloc.local/behaviorlab/internal/swarmbench"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type taskInput struct {
	Text          string `json:"text"`
	ReferenceDate string `json:"reference_date"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tlaloc-swarm-worker <date-number|router|verifier>")
		os.Exit(2)
	}

	body, err := io.ReadAll(os.Stdin)
	die(err)
	var req tlaloque.CapabilityRequest
	die(json.Unmarshal(body, &req))

	var resp tlaloque.CapabilityResponse
	switch os.Args[1] {
	case "date-number":
		resp, err = runDateNumber(req)
	case "router":
		resp, err = runRouter(req)
	case "verifier":
		resp, err = runVerifier(req)
	default:
		fmt.Fprintf(os.Stderr, "unknown worker %q\n", os.Args[1])
		os.Exit(2)
	}
	die(err)

	out, err := json.Marshal(resp)
	die(err)
	fmt.Println(string(out))
}

func decodeTaskInput(raw json.RawMessage) (taskInput, error) {
	var input taskInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return taskInput{}, fmt.Errorf("task input: %w", err)
	}
	return input, nil
}

func runDateNumber(req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	input, err := decodeTaskInput(req.Input)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	dateISO, amountCents, err := swarmbench.DateNumberWorkerLogic(input.Text, input.ReferenceDate)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	output, err := json.Marshal(struct {
		DateISO     string `json:"date_iso"`
		AmountCents int64  `json:"amount_cents"`
	}{DateISO: dateISO, AmountCents: amountCents})
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: "date-number-r0", Output: output, Confidence: 1}, nil
}

func runRouter(req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	input, err := decodeTaskInput(req.Input)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	var intentOut struct {
		Intent string `json:"intent"`
	}
	var entityOut struct {
		Organization string `json:"organization"`
	}
	var dateOut struct {
		DateISO     string `json:"date_iso"`
		AmountCents int64  `json:"amount_cents"`
	}
	// Context is keyed by DAG node id, matching swarmbench.BuildFanInPlan's
	// node names — see the same contract documented in swarmbench/swarm.go.
	if raw, ok := req.Context["intent"]; ok {
		_ = json.Unmarshal(raw, &intentOut)
	}
	if raw, ok := req.Context["entity"]; ok {
		_ = json.Unmarshal(raw, &entityOut)
	}
	if raw, ok := req.Context["date-number"]; ok {
		_ = json.Unmarshal(raw, &dateOut)
	}
	fields, err := swarmbench.RouteWorkerLogic(intentOut.Intent, entityOut.Organization, dateOut.AmountCents, dateOut.DateISO, input.ReferenceDate)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	output, err := json.Marshal(fields)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: "router-r0", Output: output, Confidence: 1}, nil
}

func runVerifier(req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	input, err := decodeTaskInput(req.Input)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	var fields swarmbench.Fields
	if raw, ok := req.Context["route"]; ok {
		if err := json.Unmarshal(raw, &fields); err != nil {
			return tlaloque.CapabilityResponse{}, fmt.Errorf("verifier: router output: %w", err)
		}
	}
	corrected, _, err := swarmbench.VerifyWorkerLogic(fields, input.ReferenceDate)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	output, err := json.Marshal(corrected)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: "verifier-r0", Output: output, Confidence: 1}, nil
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
