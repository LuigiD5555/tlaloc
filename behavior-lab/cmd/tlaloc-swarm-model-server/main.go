// Command tlaloc-swarm-model-server is the HTTP_JSON-transport binary for the
// two small-model-shaped Tlaloques in the swarm-bench decomposition
// experiment: intent detection and entity extraction. It starts once and
// stays resident, answering many CapabilityRequest/CapabilityResponse round
// trips without reloading anything — the mechanism a real BERT-class
// resident service would use, exercised here before any real weights exist.
//
// Today it wraps internal/swarmbench's heuristic classifiers (a lexicon and
// a gazetteer, not a trained model) — see swarmbench.IntentWorkerLogic and
// swarmbench.EntityWorkerLogic. Swapping in real small-model inference later
// means replacing only the handler body; the transport, residency and
// CapabilityResponse contract stay exactly as tested here.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"tlaloc.local/behaviorlab/internal/swarmbench"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type taskInput struct {
	Text          string `json:"text"`
	ReferenceDate string `json:"reference_date"`
}

func main() {
	var capability, addr, workerID string
	var parameterCount int64
	flag.StringVar(&capability, "capability", "", "DETECT_INTENT or EXTRACT_ENTITY")
	flag.StringVar(&addr, "addr", "127.0.0.1:9101", "listen address")
	flag.StringVar(&workerID, "worker-id", "", "worker id this service answers as")
	flag.Int64Var(&parameterCount, "parameter-count", 0, "declared parameter count for the manifest descriptor")
	flag.Parse()

	capability = strings.ToUpper(strings.TrimSpace(capability))
	var handler func(taskInput) (json.RawMessage, float64, error)
	switch capability {
	case "DETECT_INTENT":
		if workerID == "" {
			workerID = "intent-lexicon-r0"
		}
		handler = intentHandler
	case "EXTRACT_ENTITY":
		if workerID == "" {
			workerID = "entity-gazetteer-r0"
		}
		handler = entityHandler
	default:
		log.Fatalf("--capability must be DETECT_INTENT or EXTRACT_ENTITY, got %q", capability)
	}

	http.HandleFunc("/infer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req tlaloque.CapabilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var input taskInput
		if err := json.Unmarshal(req.Input, &input); err != nil {
			http.Error(w, "task input: "+err.Error(), http.StatusBadRequest)
			return
		}
		output, confidence, err := handler(input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{WorkerID: workerID, Output: output, Confidence: confidence})
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("tlaloc-swarm-model-server: %s resident on %s as %s (parameter_count=%d)\n", capability, addr, workerID, parameterCount)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func intentHandler(input taskInput) (json.RawMessage, float64, error) {
	intent, confidence := swarmbench.IntentWorkerLogic(input.Text)
	output, err := json.Marshal(struct {
		Intent string `json:"intent"`
	}{Intent: intent})
	return output, confidence, err
}

func entityHandler(input taskInput) (json.RawMessage, float64, error) {
	organization, confidence := swarmbench.EntityWorkerLogic(input.Text)
	output, err := json.Marshal(struct {
		Organization string `json:"organization"`
	}{Organization: organization})
	return output, confidence, err
}
