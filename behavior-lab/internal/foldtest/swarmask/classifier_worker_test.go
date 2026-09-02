package swarmask

import (
	"context"
	"encoding/json"
	"testing"
)

func TestQuestionClassifierWorker_Types(t *testing.T) {
	tests := []struct {
		name         string
		question     string
		wantType     string
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
		})
	}
}
