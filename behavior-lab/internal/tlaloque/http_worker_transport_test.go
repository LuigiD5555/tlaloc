package tlaloque

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func residentService(t *testing.T, loads *int32, calls *int32) *httptest.Server {
	t.Helper()
	// A resident service loads its weights once, at construction.
	atomic.AddInt32(loads, 1)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		var req CapabilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CapabilityResponse{
			WorkerID:   "resident-bert",
			Output:     json.RawMessage(`{"node":"` + req.NodeID + `"}`),
			Confidence: 0.9,
		})
	}))
}

// The point of HTTP_JSON is residency: many invocations, one weight load.
// Without this guarantee the transport has no reason to exist over PROCESS.
func TestHTTPWorkerKeepsModelResidentAcrossInvocations(t *testing.T) {
	var loads, calls int32
	server := residentService(t, &loads, &calls)
	defer server.Close()

	worker := HTTPWorker{
		Desc:     CapabilityDescriptor{ID: "resident-bert", Capability: "DETECT_INTENT", Scope: ScopeGeneral, Engine: EngineModel, InputSchema: "text", OutputSchema: "intent", ParameterCount: 12_000_000},
		Endpoint: server.URL,
	}
	const invocations = 32
	for index := 0; index < invocations; index++ {
		if _, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{"text":"x"}`)}); err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt32(&calls); got != invocations {
		t.Fatalf("service received %d requests, want %d", got, invocations)
	}
	if got := atomic.LoadInt32(&loads); got != 1 {
		t.Fatalf("weights loaded %d times, want exactly 1", got)
	}
}

// A resident worker declaring MAX_CONCURRENCY must not be entered beyond it,
// which is how VRAM stays bounded during the scaling runs.
func TestHTTPWorkerUnderSwarmRespectsDeclaredConcurrency(t *testing.T) {
	var inFlight, peak int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&inFlight, 1)
		for {
			observed := atomic.LoadInt32(&peak)
			if current <= observed || atomic.CompareAndSwapInt32(&peak, observed, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CapabilityResponse{WorkerID: "resident", Output: json.RawMessage(`{}`)})
	}))
	defer server.Close()

	descriptor := CapabilityDescriptor{
		ID: "resident", Capability: "SCORE", Scope: ScopeGeneral, Engine: EngineModel,
		InputSchema: "text", OutputSchema: "score", ParameterCount: 12_000_000, MaxConcurrency: 2,
	}
	registry := NewRegistry()
	mustRegister(t, registry, HTTPWorker{Desc: descriptor, Endpoint: server.URL})

	nodes := make([]SwarmNode, 0, 8)
	for index := 0; index < 8; index++ {
		nodes = append(nodes, SwarmNode{ID: string(rune('a' + index)), Capability: "SCORE", WorkerID: "resident"})
	}
	plan := SwarmPlan{ID: "resident-load", MaxParallel: 8, Nodes: nodes}
	report, err := (SwarmRunner{Registry: registry}).Run(context.Background(), plan, "task", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Succeeded {
		t.Fatalf("report=%+v", report)
	}
	if observed := atomic.LoadInt32(&peak); observed > 2 {
		t.Fatalf("resident service saw %d concurrent requests, declared MaxConcurrency=2", observed)
	}
}

func TestHTTPWorkerRejectsMisbehavingServices(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "model not loaded", http.StatusServiceUnavailable)
			},
			want: "model not loaded",
		},
		{
			name: "identity mismatch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(CapabilityResponse{WorkerID: "impostor", Output: json.RawMessage(`{}`)})
			},
			want: "identity mismatch",
		},
		{
			name: "empty output",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"worker_id":"intent-http"}`))
			},
			want: "invalid JSON output",
		},
		{
			name: "undeclared fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"worker_id":"intent-http","output":{},"promote_me":true}`))
			},
			want: "invalid response",
		},
		{
			name: "non json body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`<html>gateway</html>`))
			},
			want: "invalid response",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(testCase.handler)
			defer server.Close()
			worker := HTTPWorker{Desc: generalDescriptor("intent-http", "DETECT_INTENT"), Endpoint: server.URL}
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

// An omitted worker_id is filled in from the descriptor: a worker cannot claim
// an identity it was not registered under, but it need not repeat its own.
func TestHTTPWorkerDefaultsWorkerIDFromDescriptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"output":{"intent":"SEARCH"},"confidence":0.5}`))
	}))
	defer server.Close()
	worker := HTTPWorker{Desc: generalDescriptor("intent-http", "DETECT_INTENT"), Endpoint: server.URL}
	response, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if response.WorkerID != "intent-http" {
		t.Fatalf("worker id=%s, want the descriptor id", response.WorkerID)
	}
}

func TestHTTPWorkerEnforcesTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer func() {
		close(release)
		server.Close()
	}()
	worker := HTTPWorker{Desc: generalDescriptor("slow-http", "SLOW"), Endpoint: server.URL, Timeout: 80 * time.Millisecond}
	start := time.Now()
	if _, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected the timeout to fire")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("timeout did not bound the call: %s", elapsed)
	}
}

func TestHTTPWorkerRequiresEndpoint(t *testing.T) {
	worker := HTTPWorker{Desc: generalDescriptor("no-endpoint", "A")}
	if _, err := worker.Execute(context.Background(), CapabilityRequest{Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected a worker with no endpoint to fail")
	}
}

func TestHTTPWorkerReportsUnreachableService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL
	server.Close()
	worker := HTTPWorker{Desc: generalDescriptor("down", "A"), Endpoint: endpoint, Timeout: time.Second}
	if _, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "t", NodeID: "n", Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected an unreachable resident service to fail")
	}
}
