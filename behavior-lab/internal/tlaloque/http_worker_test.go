package tlaloque

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPWorkerKeepsProtocolBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		var req CapabilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.TaskID != "task" || req.NodeID != "intent" {
			t.Fatalf("req=%+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CapabilityResponse{WorkerID: "intent-http", Output: json.RawMessage(`{"intent":"SEARCH"}`), Confidence: 0.98})
	}))
	defer server.Close()

	worker := HTTPWorker{Desc: CapabilityDescriptor{ID: "intent-http", Capability: "DETECT_INTENT", Scope: ScopeGeneral, Engine: EngineModel, InputSchema: "text", OutputSchema: "intent", ParameterCount: 18_000_000}, Endpoint: server.URL}
	resp, err := worker.Execute(context.Background(), CapabilityRequest{TaskID: "task", NodeID: "intent", Input: json.RawMessage(`{"text":"find it"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if resp.WorkerID != "intent-http" || string(resp.Output) != `{"intent":"SEARCH"}` {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestRegistryRefusesSpecificWorkerWithoutDomainEvidence(t *testing.T) {
	r := NewRegistry()
	worker := testWorker{desc: CapabilityDescriptor{ID: "cfdi-only", Capability: "EXTRACT_ENTITY", Scope: ScopeSpecific, Domain: "CFDI", Engine: EngineModel, InputSchema: "text", OutputSchema: "entities", ParameterCount: 5_000_000}}
	if err := r.Register(worker); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Select(SelectionRequest{Capability: "EXTRACT_ENTITY"}); err == nil {
		t.Fatal("expected routing refusal without domain evidence")
	}
}
