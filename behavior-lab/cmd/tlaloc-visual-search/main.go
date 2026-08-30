package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/visualsearch"
)

type input struct {
	Schema        string                   `json:"schema"`
	BaseProfileID string                   `json:"base_profile_id"`
	Baseline      visualsearch.Metrics     `json:"baseline"`
	Policy        visualsearch.Policy      `json:"policy,omitempty"`
	Candidates    []visualsearch.Candidate `json:"candidates"`
	Evidence      []visualsearch.Evidence  `json:"evidence"`
}

func main() {
	in := flag.String("in", "-", "search input JSON or - for stdin")
	out := flag.String("out", "-", "tournament report JSON or - for stdout")
	flag.Parse()
	body, err := read(*in)
	die(err)
	var request input
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	die(dec.Decode(&request))
	if request.Schema != "" && request.Schema != visualsearch.SchemaR0+".request" {
		die(fmt.Errorf("unexpected schema %q", request.Schema))
	}
	report, err := visualsearch.Rank(request.BaseProfileID, request.Baseline, request.Candidates, request.Evidence, request.Policy)
	die(err)
	encoded, err := json.MarshalIndent(report, "", "  ")
	die(err)
	encoded = append(encoded, '\n')
	if *out == "-" {
		_, err = os.Stdout.Write(encoded)
		die(err)
		return
	}
	die(os.WriteFile(*out, encoded, 0o644))
}

func read(path string) ([]byte, error) {
	if path == "-" { return os.ReadFile("/dev/stdin") }
	return os.ReadFile(path)
}
func die(err error) { if err != nil { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) } }
