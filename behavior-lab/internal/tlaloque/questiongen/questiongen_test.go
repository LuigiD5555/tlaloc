package questiongen

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func mustRequest(t *testing.T, in GenerateInput) tlaloque.CapabilityRequest {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal GenerateInput: %v", err)
	}
	return tlaloque.CapabilityRequest{Input: raw}
}

func decodeOutput(t *testing.T, resp tlaloque.CapabilityResponse) GenerateOutput {
	t.Helper()
	var out GenerateOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal GenerateOutput: %v", err)
	}
	return out
}

// Positive case: with enough sentences, the template worker adds keyword-derived
// questions on top of the 3 fixed base questions.
func TestTemplateWorker_PositiveCase(t *testing.T) {
	pageContent := "El enjambre coordina agentes distribuidos. La telemetría revela patrones emergentes. Otros datos siguen aquí."

	resp, err := TemplateWorker{}.Execute(context.Background(), mustRequest(t, GenerateInput{
		PageContent: pageContent, PageNumber: 12,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeOutput(t, resp)
	if len(out.Questions) <= 3 {
		t.Errorf("expected keyword-derived questions beyond the 3 base templates, got %d: %v", len(out.Questions), out.Questions)
	}
	if resp.WorkerID != TemplateWorkerID {
		t.Errorf("expected worker id %q, got %q", TemplateWorkerID, resp.WorkerID)
	}
}

// Non-applicable case: empty page content still yields the 3 base questions
// without panicking in extractKeyword.
func TestTemplateWorker_NonApplicableCase(t *testing.T) {
	resp, err := TemplateWorker{}.Execute(context.Background(), mustRequest(t, GenerateInput{
		PageContent: "", PageNumber: 3,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeOutput(t, resp)
	if len(out.Questions) != 3 {
		t.Errorf("expected exactly the 3 base questions for empty content, got %d: %v", len(out.Questions), out.Questions)
	}
}

// Boundary: content with 2 or fewer sentences must not trigger keyword-derived
// questions (mirrors the `len(sentences) > 2` guard moved from validation.go).
func TestTemplateWorker_SentenceCountBoundary(t *testing.T) {
	resp, err := TemplateWorker{}.Execute(context.Background(), mustRequest(t, GenerateInput{
		PageContent: "Una sola oración sin punto final adicional", PageNumber: 5,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeOutput(t, resp)
	if len(out.Questions) != 3 {
		t.Errorf("expected exactly the 3 base questions when sentence count is at/below the boundary, got %d: %v", len(out.Questions), out.Questions)
	}
}

// Fallback interaction: when the semantic model worker cannot be reached,
// GenerateQuestions must fall back to the deterministic worker and report
// that honestly in workerID rather than implying the model generated them.
func TestGenerateQuestions_FallsBackWhenModelWorkerFails(t *testing.T) {
	registry := tlaloque.NewRegistry()
	unreachable := SemanticModelWorker{Client: target.OpenAICompat{
		Model:   "lfm2-vl-1.6b",
		BaseURL: "http://127.0.0.1:1/unreachable", // reserved, non-listening port
	}}
	if err := registry.Register(unreachable); err != nil {
		t.Fatalf("register model worker: %v", err)
	}
	if err := registry.Register(TemplateWorker{}); err != nil {
		t.Fatalf("register template worker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel() // an already-expired context forces the model call to fail fast

	out, workerID, err := GenerateQuestions(ctx, registry, GenerateInput{
		PageContent: "Contenido de referencia con más de dos oraciones. Segunda oración. Tercera oración.", PageNumber: 9,
	})
	if err != nil {
		t.Fatalf("GenerateQuestions: %v", err)
	}
	if workerID != TemplateWorkerID {
		t.Errorf("expected fallback to report worker id %q, got %q", TemplateWorkerID, workerID)
	}
	if len(out.Questions) == 0 {
		t.Error("expected the deterministic fallback to still produce real questions")
	}
}

// The model worker must reject a response with too few parsed questions
// rather than silently accepting a poor list.
func TestSemanticModelWorker_RejectsTooFewQuestions(t *testing.T) {
	got := parseQuestions("Q: única pregunta encontrada")
	if len(got) >= minModelQuestions {
		t.Fatalf("test setup invalid: expected fewer than %d parsed questions, got %d", minModelQuestions, len(got))
	}
}
