package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain lets this test binary re-execute itself as a PROCESS Tlaloque, so
// the transport is exercised against a real child process without shipping
// fixture executables.
func TestMain(m *testing.M) {
	if behavior := os.Getenv("TLALOQUE_TEST_WORKER"); behavior != "" {
		os.Exit(runTestWorker(behavior))
	}
	os.Exit(m.Run())
}

func runTestWorker(behavior string) int {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker could not read stdin:", err)
		return 1
	}
	var req CapabilityRequest
	if err := json.Unmarshal(body, &req); err != nil {
		fmt.Fprintln(os.Stderr, "worker received invalid request:", err)
		return 1
	}
	switch behavior {
	case "echo":
		response := CapabilityResponse{
			WorkerID:   "process-echo",
			Output:     json.RawMessage(`{"task":"` + req.TaskID + `","node":"` + req.NodeID + `","dependencies":` + fmt.Sprint(len(req.Context)) + `}`),
			Confidence: 0.75,
		}
		_ = json.NewEncoder(os.Stdout).Encode(response)
	case "wrong-identity":
		_ = json.NewEncoder(os.Stdout).Encode(CapabilityResponse{WorkerID: "someone-else", Output: json.RawMessage(`{}`)})
	case "not-json":
		fmt.Fprintln(os.Stdout, "this is not a capability response")
	case "empty-output":
		fmt.Fprintln(os.Stdout, `{"worker_id":"process-echo"}`)
	case "extra-fields":
		fmt.Fprintln(os.Stdout, `{"worker_id":"process-echo","output":{},"promotion":"self"}`)
	case "crash":
		fmt.Fprintln(os.Stderr, "worker exploded")
		return 3
	case "hang":
		time.Sleep(10 * time.Second)
	}
	return 0
}

func processWorkerWithBehavior(t *testing.T, behavior string, timeout time.Duration) ProcessWorker {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLALOQUE_TEST_WORKER", "")
	return ProcessWorker{
		Desc:    generalDescriptor("process-echo", "ECHO"),
		Command: []string{"/usr/bin/env", "TLALOQUE_TEST_WORKER=" + behavior, executable, "-test.run=TestMain"},
		Timeout: timeout,
	}
}

// The PROCESS contract: one CapabilityRequest on stdin, one CapabilityResponse
// on stdout. This is what lets Go, Rust, Python or shell workers be equal.
func TestProcessWorkerRoundTripsTheBoundedContract(t *testing.T) {
	worker := processWorkerWithBehavior(t, "echo", 10*time.Second)
	response, err := worker.Execute(context.Background(), CapabilityRequest{
		TaskID:  "task-7",
		NodeID:  "route",
		Input:   json.RawMessage(`{"text":"hello"}`),
		Context: map[string]json.RawMessage{"intent": json.RawMessage(`{"intent":"SEARCH"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.WorkerID != "process-echo" {
		t.Fatalf("worker id=%s", response.WorkerID)
	}
	if response.Confidence != 0.75 {
		t.Fatalf("confidence=%v", response.Confidence)
	}
	var decoded struct {
		Task         string `json:"task"`
		Node         string `json:"node"`
		Dependencies int    `json:"dependencies"`
	}
	if err := json.Unmarshal(response.Output, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Task != "task-7" || decoded.Node != "route" {
		t.Fatalf("request not delivered intact: %+v", decoded)
	}
	if decoded.Dependencies != 1 {
		t.Fatalf("dependency context not delivered: %+v", decoded)
	}
}

func TestProcessWorkerRejectsMisbehavingChildren(t *testing.T) {
	cases := []struct {
		name     string
		behavior string
		want     string
	}{
		{name: "identity mismatch", behavior: "wrong-identity", want: "identity mismatch"},
		{name: "non json stdout", behavior: "not-json", want: "invalid response"},
		{name: "empty output", behavior: "empty-output", want: "invalid JSON output"},
		{name: "undeclared fields", behavior: "extra-fields", want: "invalid response"},
		{name: "non zero exit", behavior: "crash", want: "worker exploded"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			worker := processWorkerWithBehavior(t, testCase.behavior, 10*time.Second)
			_, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{}`)})
			if err == nil {
				t.Fatalf("expected %s to be refused", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error %q does not mention %q", err, testCase.want)
			}
		})
	}
}

// A Tlaloque that hangs must not hang the swarm: the per-individual timeout is
// what keeps a 128-worker run bounded.
func TestProcessWorkerEnforcesTimeout(t *testing.T) {
	worker := processWorkerWithBehavior(t, "hang", 100*time.Millisecond)
	start := time.Now()
	_, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{}`)})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected the timeout to fire")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("timeout did not bound the call: %s", elapsed)
	}
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error does not report the deadline: %v", err)
	}
}

func TestProcessWorkerRequiresCommand(t *testing.T) {
	worker := ProcessWorker{Desc: generalDescriptor("no-command", "A")}
	if _, err := worker.Execute(context.Background(), CapabilityRequest{Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected a worker with no command to fail")
	}
}
