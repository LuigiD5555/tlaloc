package answerscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func mustRequest(t *testing.T, in ScoreInput) tlaloque.CapabilityRequest {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal ScoreInput: %v", err)
	}
	return tlaloque.CapabilityRequest{Input: raw}
}

func decodeOutput(t *testing.T, resp tlaloque.CapabilityResponse) ScoreOutput {
	t.Helper()
	var out ScoreOutput
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("unmarshal ScoreOutput: %v", err)
	}
	return out
}

// fixedScoreWorker is a test double standing in for a real embedding or
// model worker, so ordering/fallback behavior can be tested without network.
type fixedScoreWorker struct {
	id   string
	out  ScoreOutput
	fail bool
}

func (w fixedScoreWorker) Descriptor() tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{ID: w.id, Capability: Capability, Engine: tlaloque.EngineModel, InputSchema: inputSchema, OutputSchema: outputSchema}
}

func (w fixedScoreWorker) Execute(_ context.Context, _ tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	if w.fail {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("fixedScoreWorker %q: simulated failure", w.id)
	}
	raw, _ := json.Marshal(w.out)
	return tlaloque.CapabilityResponse{WorkerID: w.id, Output: raw}, nil
}

// Positive case: an answer that clearly reuses the page's vocabulary scores high.
func TestKeywordOverlapWorker_PositiveCase(t *testing.T) {
	pageContent := "El agente distribuido coordina la formación del enjambre mediante señales locales entre vecinos."
	answer := "El agente coordina la formación del enjambre usando señales locales entre vecinos."

	resp, err := KeywordOverlapWorker{}.Execute(context.Background(), mustRequest(t, ScoreInput{
		Question: "¿Cómo coordina el agente la formación?", ModelAnswer: answer, PageContent: pageContent,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeOutput(t, resp)
	if out.Score < 0.7 {
		t.Errorf("expected high score for well-matched answer, got %.2f (notes=%q)", out.Score, out.Notes)
	}
	if resp.WorkerID != KeywordOverlapWorkerID {
		t.Errorf("expected worker id %q, got %q", KeywordOverlapWorkerID, resp.WorkerID)
	}
}

// Non-applicable case: empty page content yields a neutral score with an explanatory note.
func TestKeywordOverlapWorker_NonApplicableCase(t *testing.T) {
	resp, err := KeywordOverlapWorker{}.Execute(context.Background(), mustRequest(t, ScoreInput{
		Question: "¿Qué dice la página?", ModelAnswer: "Cualquier respuesta.", PageContent: "",
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeOutput(t, resp)
	if out.Score != 0.5 {
		t.Errorf("expected neutral score 0.5 for empty page content, got %.2f", out.Score)
	}
	if out.Notes == "" {
		t.Error("expected an explanatory note for the non-applicable case")
	}
}

// False-positive boundary: an answer far shorter than the page content, even
// with matching keywords, must not score as a full match.
func TestKeywordOverlapWorker_LengthBoundary(t *testing.T) {
	pageContent := strings.Repeat("órbita satélite telemetría propulsión combustible trayectoria ", 20)
	shortAnswer := "órbita satélite."

	resp, err := KeywordOverlapWorker{}.Execute(context.Background(), mustRequest(t, ScoreInput{
		Question: "Describe el contenido en detalle", ModelAnswer: shortAnswer, PageContent: pageContent,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeOutput(t, resp)
	if out.Score >= 0.9 {
		t.Errorf("expected the length penalty to keep score below 0.9 for a disproportionately short answer, got %.2f", out.Score)
	}
}

// Fallback interaction: when the semantic model worker cannot be reached,
// ScoreAnswer must fall back to the deterministic worker and report that
// honestly in workerID rather than implying the model judged the answer.
func TestScoreAnswer_FallsBackWhenModelWorkerFails(t *testing.T) {
	registry := tlaloque.NewRegistry()
	unreachable := SemanticModelWorker{Client: target.OpenAICompat{
		Model:   "lfm2-vl-1.6b",
		BaseURL: "http://127.0.0.1:1/unreachable", // reserved, non-listening port
	}}
	if err := registry.Register(unreachable); err != nil {
		t.Fatalf("register model worker: %v", err)
	}
	if err := registry.Register(KeywordOverlapWorker{}); err != nil {
		t.Fatalf("register keyword worker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel() // an already-expired context forces the model call to fail fast

	out, workerID, err := ScoreAnswer(ctx, registry, ScoreInput{
		Question: "¿Qué dice la página?", ModelAnswer: "respuesta", PageContent: "contenido de referencia",
	})
	if err != nil {
		t.Fatalf("ScoreAnswer: %v", err)
	}
	if workerID != KeywordOverlapWorkerID {
		t.Errorf("expected fallback to report worker id %q, got %q", KeywordOverlapWorkerID, workerID)
	}
	if out.Notes == "" && out.Score == 0 && out.KeywordsTotal == 0 {
		t.Error("expected the deterministic fallback to still produce a real score")
	}
}

func TestKeywordOverlap_ExtractedKeywordsMeetMinLength(t *testing.T) {
	tests := []struct {
		text   string
		minLen int
	}{
		{text: "The quick brown fox jumps over the lazy dog", minLen: 4},
		{text: "el gato y el perro corren rápidamente", minLen: 4},
		{text: "a b c d e", minLen: 4},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := extractKeywords(tt.text)
			for keyword := range got {
				if len(keyword) <= tt.minLen {
					t.Errorf("keyword %q is too short (min %d)", keyword, tt.minLen)
				}
			}
		})
	}
}

func TestEstimateConfidence(t *testing.T) {
	tests := []struct {
		response string
		minConf  float64
		maxConf  float64
	}{
		{response: "Definitivamente, la respuesta es correcta", minConf: 0.85, maxConf: 0.95},
		{response: "Probablemente la respuesta es", minConf: 0.55, maxConf: 0.65},
		{response: "No estoy seguro de la respuesta", minConf: 0.35, maxConf: 0.45},
	}

	for _, tt := range tests {
		t.Run(tt.response, func(t *testing.T) {
			got := estimateConfidence(tt.response)
			if got < tt.minConf || got > tt.maxConf {
				t.Errorf("got confidence=%f, want [%f, %f]", got, tt.minConf, tt.maxConf)
			}
		})
	}
}

func TestScoreByKeywordOverlap(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		content  string
		minScore float64
		maxScore float64
	}{
		{
			name:     "perfect match",
			answer:   "Swarms are distributed systems with multiple agents communicating together",
			content:  "Swarms are distributed systems composed of multiple agents that communicate",
			minScore: 0.7,
			maxScore: 1.0,
		},
		{
			name:     "partial match",
			answer:   "The document talks about systems",
			content:  "Swarms are distributed systems with algorithms and learning mechanisms",
			minScore: 0.4,
			maxScore: 0.8,
		},
		{
			name:     "no match",
			answer:   "The capital of France is Paris",
			content:  "Swarms are distributed systems with multiple agents",
			minScore: 0.0,
			maxScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreByKeywordOverlap(tt.answer, tt.content, 0.8)
			if got.Score < tt.minScore || got.Score > tt.maxScore {
				t.Errorf("got score=%f, want [%f, %f]", got.Score, tt.minScore, tt.maxScore)
			}
			if got.KeywordsTotal == 0 {
				t.Logf("warning: no keywords found in content for test %q", tt.name)
			}
		})
	}
}

// ScoreAnswer must prefer the chat-based semantic judge over the cheaper
// embedding worker when both are registered and the semantic judge succeeds
// — a real run showed the embedding worker rewards lexical keyword overlap
// over genuine paraphrase understanding, so it is the secondary signal.
func TestScoreAnswer_PrefersSemanticModelOverEmbeddingWorker(t *testing.T) {
	registry := tlaloque.NewRegistry()
	if err := registry.Register(fixedScoreWorker{id: EmbeddingScoreWorkerID, out: ScoreOutput{Score: 0.95, Notes: "should not be used"}}); err != nil {
		t.Fatalf("register embedding worker: %v", err)
	}
	if err := registry.Register(fixedScoreWorker{id: SemanticModelWorkerID, out: ScoreOutput{Score: 0.6, Notes: "model judged"}}); err != nil {
		t.Fatalf("register model worker: %v", err)
	}
	if err := registry.Register(KeywordOverlapWorker{}); err != nil {
		t.Fatalf("register keyword worker: %v", err)
	}

	out, workerID, err := ScoreAnswer(context.Background(), registry, ScoreInput{
		Question: "¿?", ModelAnswer: "respuesta", PageContent: "contenido",
	})
	if err != nil {
		t.Fatalf("ScoreAnswer: %v", err)
	}
	if workerID != SemanticModelWorkerID {
		t.Errorf("expected the semantic model worker to be preferred, got %q", workerID)
	}
	if out.Score != 0.6 {
		t.Errorf("expected the semantic model worker's score to be used, got %f", out.Score)
	}
}

// When the semantic model judge is registered but fails, ScoreAnswer must
// fall through to the embedding worker (a cheaper secondary signal) rather
// than skipping straight to the deterministic worker.
func TestScoreAnswer_FallsPastFailedSemanticModelToEmbedding(t *testing.T) {
	registry := tlaloque.NewRegistry()
	if err := registry.Register(fixedScoreWorker{id: SemanticModelWorkerID, fail: true}); err != nil {
		t.Fatalf("register model worker: %v", err)
	}
	if err := registry.Register(fixedScoreWorker{id: EmbeddingScoreWorkerID, out: ScoreOutput{Score: 0.4, Notes: "embedding fallback"}}); err != nil {
		t.Fatalf("register embedding worker: %v", err)
	}
	if err := registry.Register(KeywordOverlapWorker{}); err != nil {
		t.Fatalf("register keyword worker: %v", err)
	}

	out, workerID, err := ScoreAnswer(context.Background(), registry, ScoreInput{
		Question: "¿?", ModelAnswer: "respuesta", PageContent: "contenido",
	})
	if err != nil {
		t.Fatalf("ScoreAnswer: %v", err)
	}
	if workerID != EmbeddingScoreWorkerID {
		t.Errorf("expected fallthrough to the embedding worker, got %q", workerID)
	}
	if out.Score != 0.4 {
		t.Errorf("expected the embedding worker's score to be used, got %f", out.Score)
	}
}

// When both the semantic model judge and the embedding worker are
// unreachable, ScoreAnswer must fall all the way through to the mandatory
// deterministic keyword-overlap worker.
func TestScoreAnswer_FallsPastUnreachableEmbeddingWorker(t *testing.T) {
	registry := tlaloque.NewRegistry()
	if err := registry.Register(fixedScoreWorker{id: SemanticModelWorkerID, fail: true}); err != nil {
		t.Fatalf("register model worker: %v", err)
	}
	unreachableEmbedding := tlaloque.HTTPWorker{
		Desc:     EmbeddingScoreDescriptor(""),
		Endpoint: "http://127.0.0.1:1/unreachable",
		Timeout:  2 * time.Second,
	}
	if err := registry.Register(unreachableEmbedding); err != nil {
		t.Fatalf("register embedding worker: %v", err)
	}
	if err := registry.Register(KeywordOverlapWorker{}); err != nil {
		t.Fatalf("register keyword worker: %v", err)
	}

	_, workerID, err := ScoreAnswer(context.Background(), registry, ScoreInput{
		Question: "¿?", ModelAnswer: "respuesta", PageContent: "contenido",
	})
	if err != nil {
		t.Fatalf("ScoreAnswer: %v", err)
	}
	if workerID != KeywordOverlapWorkerID {
		t.Errorf("expected fallthrough all the way to the keyword-overlap worker, got %q", workerID)
	}
}
