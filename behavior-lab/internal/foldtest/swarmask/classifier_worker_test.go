package swarmask

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/questionclass"
)

func TestQuestionClassifierWorker_Types(t *testing.T) {
	tests := []struct {
		name          string
		question      string
		wantType      string
		wantConfident bool
	}{
		{name: "definition english", question: "What is a swarm?", wantType: QuestionTypeDefinition, wantConfident: true},
		{name: "definition spanish", question: "¿Qué es un enjambre?", wantType: QuestionTypeDefinition, wantConfident: true},
		{name: "comparison", question: "What is the relationship between swarms and agents?", wantType: QuestionTypeComparison, wantConfident: true},
		{name: "process", question: "How do swarms coordinate?", wantType: QuestionTypeProcess, wantConfident: true},
		{name: "factual detail", question: "What happened with swarm research in 2019?", wantType: QuestionTypeFactualDetail, wantConfident: true},
		{name: "general fallback", question: "Tell me about swarms.", wantType: QuestionTypeGeneral, wantConfident: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := QuestionClassifierWorker{}.Execute(context.Background(), mustAskRequest(t, AskInput{Question: tt.question}))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			var out QuestionTypeOutput
			if err := json.Unmarshal(resp.Output, &out); err != nil {
				t.Fatalf("unmarshal output: %v", err)
			}
			if out.Type != tt.wantType {
				t.Errorf("question %q: got type %q, want %q", tt.question, out.Type, tt.wantType)
			}

			// Unlike scout/entities, the classifier always emits an
			// observation — even GENERAL is informative, not a fabrication.
			if len(resp.Observations) != 1 {
				t.Fatalf("expected exactly 1 observation, got %d", len(resp.Observations))
			}
			confidence := resp.Observations[0].Confidence
			if tt.wantConfident && confidence < 0.5 {
				t.Errorf("expected a confident classification (>=0.5), got %f", confidence)
			}
			if !tt.wantConfident && confidence >= 0.5 {
				t.Errorf("expected a low-confidence GENERAL fallback (<0.5), got %f", confidence)
			}
			// With no ModelRegistry the verdict must come from the rules.
			if method := resp.Observations[0].Provenance["method"]; method != "rule-based" {
				t.Errorf("expected rule-based provenance, got %q", method)
			}
		})
	}
}

func classifierModelRegistry(t *testing.T, workerID, verdict string, confidence float64) *tlaloque.Registry {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outputRaw, _ := json.Marshal(questionclass.Output{Type: verdict})
		json.NewEncoder(w).Encode(tlaloque.CapabilityResponse{WorkerID: workerID, Output: outputRaw, Confidence: confidence})
	}))
	t.Cleanup(server.Close)
	return questionclass.NewRegistry(server.URL)
}

// When a confident trained model is wired, its verdict wins and the
// provenance says so honestly.
func TestQuestionClassifierWorker_UsesTrainedModel(t *testing.T) {
	worker := QuestionClassifierWorker{
		ModelRegistry: classifierModelRegistry(t, questionclass.WorkerID, QuestionTypeProcess, 0.94),
	}
	// A question the rules would call DEFINITION ("what is ...").
	resp, err := worker.Execute(context.Background(), mustAskRequest(t, AskInput{Question: "What is the process that produces flocking?"}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out QuestionTypeOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != QuestionTypeProcess {
		t.Errorf("got %q, want the model verdict %q", out.Type, QuestionTypeProcess)
	}
	if method := resp.Observations[0].Provenance["method"]; method != "charcnn-model" {
		t.Errorf("expected charcnn-model provenance, got %q", method)
	}
}

// When the model is unreachable (or unconfident), the worker falls back to
// the rules and reports that honestly rather than failing the node.
func TestQuestionClassifierWorker_FallsBackWhenModelUnreachable(t *testing.T) {
	worker := QuestionClassifierWorker{ModelRegistry: questionclass.NewRegistry("http://127.0.0.1:0")}
	resp, err := worker.Execute(context.Background(), mustAskRequest(t, AskInput{Question: "What is a swarm?"}))
	if err != nil {
		t.Fatalf("Execute must not fail when the model is unreachable: %v", err)
	}
	var out QuestionTypeOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != QuestionTypeDefinition {
		t.Errorf("expected the rule-based DEFINITION verdict, got %q", out.Type)
	}
	if method := resp.Observations[0].Provenance["method"]; method != "rule-based" {
		t.Errorf("expected rule-based provenance on fallback, got %q", method)
	}
}
