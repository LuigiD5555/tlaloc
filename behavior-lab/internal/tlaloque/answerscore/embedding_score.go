package answerscore

import (
	"math"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// EmbeddingScoreWorkerID identifies the resident micro-model worker that
// scores answers by embedding cosine similarity instead of a full chat
// completion. It is registered as an HTTP_JSON tlaloque.HTTPWorker pointed
// at cmd/tlaloc-embedding-scorer — the answerscore package never talks to
// that process directly, it only needs the descriptor and the scoring math,
// both of which the service binary reuses to stay a single implementation.
const EmbeddingScoreWorkerID = "answerscore-embedding-similarity"

// EmbeddingScoreDescriptor is the single source of truth for the embedding
// worker's capability descriptor, used both when registering it as an
// HTTPWorker here and, informationally, by the service binary's /health
// endpoint.
func EmbeddingScoreDescriptor(embeddingModel string) tlaloque.CapabilityDescriptor {
	tags := []string{"embedding-similarity", "lm-studio", "resident"}
	if embeddingModel != "" {
		tags = append(tags, embeddingModel)
	}
	return tlaloque.CapabilityDescriptor{
		ID:             EmbeddingScoreWorkerID,
		Capability:     Capability,
		Scope:          tlaloque.ScopeGeneral,
		Engine:         tlaloque.EngineModel,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Deterministic:  false,
		ParameterCount: 22_700_000, // all-MiniLM-L6-v2
		MaxConcurrency: 1,
		Tags:           tags,
	}
}

// minPassageLength filters out fragments too short to carry real topical
// content (stray punctuation, list markers) when splitting page content.
const minPassageLength = 15

// SplitIntoPassages splits page content into short, sentence-sized passages.
// Comparing an answer against the single best-matching passage — instead of
// against the whole page at once — avoids the length-asymmetry problem where
// a short answer's embedding gets diluted against a long page's embedding
// even when the answer is clearly supported by one part of it.
func SplitIntoPassages(pageContent string) []string {
	var passages []string
	for _, sentence := range strings.Split(pageContent, ".") {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) >= minPassageLength {
			passages = append(passages, sentence)
		}
	}
	return passages
}

// Calibrated against real runs against LM Studio's
// text-embedding-all-minilm-l6-v2-embedding, comparing model answers to
// their single best-matching page passage. Best-passage matching fixed the
// earlier length-asymmetry problem (whole-page comparison scored genuinely
// correct answers 0.03-0.39); real answers now land in a 0.31-0.60 spread.
//
// Known limitation, not fixed by these thresholds: this small model rewards
// lexical overlap more than paraphrase understanding — in one real run, a
// raw keyword dump ("swarm, system, autonomous, emergent...") scored 0.60
// while a well-formed paraphrase of the same page content scored 0.31. Treat
// this worker as a cheap lexical-similarity screener, not a strong semantic
// judge; that is exactly why it sits before, not instead of, the
// chat-completion SemanticModelWorker in ScoreAnswer's fallback order.
const (
	embeddingHighThreshold     = 0.55
	embeddingModerateThreshold = 0.4
)

// ScoreByBestPassageSimilarity scores an answer against the page passage it
// most closely matches by embedding cosine similarity. It never fails on its
// own; callers are responsible for handling embedding errors (network,
// dimension mismatch, no passages) before reaching here.
func ScoreByBestPassageSimilarity(answerVec []float64, passageVecs [][]float64) ScoreOutput {
	best := 0.0
	for _, passageVec := range passageVecs {
		if sim := cosineSimilarity(answerVec, passageVec); sim > best {
			best = sim
		}
	}

	score := best
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	notes := "Embedding similarity below threshold against every page passage"
	if score >= embeddingHighThreshold {
		notes = "Answer closely matches at least one page passage by embedding similarity"
	} else if score >= embeddingModerateThreshold {
		notes = "Answer moderately matches at least one page passage by embedding similarity"
	}

	return ScoreOutput{Score: score, Confidence: 0.75, Notes: notes}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
