package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

type prioritizeRequest struct {
	Schema     string                   `json:"schema,omitempty"`
	Candidates []visualsearch.Candidate `json:"candidates"`
}

type prioritizeOutput struct {
	Queue           adaptivesearch.Queue `json:"queue"`
	AttemptsAdded   int                  `json:"attempts_added,omitempty"`
	AttemptsPresent int                  `json:"attempts_already_present,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "plan":
		runPlan(os.Args[2:])
	case "prioritize":
		runPrioritize(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func runPlan(args []string) {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	root := fs.String("store", "", "learning-memory root")
	out := fs.String("out", "-", "plan JSON or - for stdout")
	fs.Parse(args)
	store := learningmemory.New(*root)
	events, err := store.LoadAll()
	die(err)
	write(*out, adaptivesearch.BuildPlan(store.Root, events))
}

func runPrioritize(args []string) {
	fs := flag.NewFlagSet("prioritize", flag.ExitOnError)
	root := fs.String("store", "", "learning-memory root")
	in := fs.String("in", "-", "candidate request JSON or - for stdin")
	out := fs.String("out", "-", "priority queue JSON or - for stdout")
	limit := fs.Int("limit", 0, "maximum candidates to return; 0 keeps all")
	record := fs.Bool("record-attempts", false, "persist selected candidates as CHANGE_ATTEMPT events linked to parent failure evidence")
	fs.Parse(args)
	body, err := read(*in)
	die(err)
	var req prioritizeRequest
	decodeStrict(body, &req)
	if req.Schema != "" && req.Schema != adaptivesearch.SchemaR0+".request" {
		die(fmt.Errorf("unexpected schema %q", req.Schema))
	}
	if len(req.Candidates) == 0 {
		die(fmt.Errorf("at least one candidate is required"))
	}
	store := learningmemory.New(*root)
	events, err := store.LoadAll()
	die(err)
	plan := adaptivesearch.BuildPlan(store.Root, events)
	queue := adaptivesearch.Prioritize(plan, req.Candidates, *limit)
	result := prioritizeOutput{Queue: queue}
	if *record {
		attempts := adaptivesearch.ChangeAttemptEvents(queue, req.Candidates)
		if len(attempts) == 0 {
			die(fmt.Errorf("cannot record adaptive attempts without real parent failure evidence"))
		}
		added, skipped, _, err := store.PutAll(attempts)
		die(err)
		result.AttemptsAdded = added
		result.AttemptsPresent = skipped
	}
	write(*out, result)
}

func read(path string) ([]byte, error) {
	if path == "-" {
		return os.ReadFile("/dev/stdin")
	}
	return os.ReadFile(path)
}
func decodeStrict(body []byte, v any) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	die(dec.Decode(v))
}
func write(path string, v any) {
	body, err := json.MarshalIndent(v, "", "  ")
	die(err)
	body = append(body, '\n')
	if path == "-" {
		_, err = os.Stdout.Write(body)
		die(err)
		return
	}
	die(os.WriteFile(path, body, 0o644))
}
func usage() { fmt.Fprintln(os.Stderr, "usage: tlaloc-adaptive-search <plan|prioritize> [flags]") }
func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
