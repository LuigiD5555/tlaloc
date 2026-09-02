package answerscore

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// preferredWorkerOrder lists optional SCORE_ANSWER_RELEVANCE candidates,
// most trustworthy first. SemanticModelWorker (a real chat-completion judge)
// goes before EmbeddingScoreWorker: a real run showed the embedding worker
// rewards lexical keyword overlap over genuine paraphrase understanding (see
// the calibration note on embeddingHighThreshold in embedding_score.go), so
// it is kept only as a cheap secondary signal, not the primary judge.
// KeywordOverlapWorkerID is not in this list: it is the mandatory final
// fallback, tried separately below.
var preferredWorkerOrder = []string{SemanticModelWorkerID, EmbeddingScoreWorkerID}

// ScoreAnswer selects the SCORE_ANSWER_RELEVANCE worker via the registry,
// trying each optional candidate in preferredWorkerOrder (only those that
// are actually registered) before falling back to the mandatory
// deterministic keyword-overlap worker. A candidate that errors (LM Studio
// or the embedding service unreachable, malformed response, timeout) is
// skipped in favor of the next one — the caller must never be told a model
// judged an answer when it was really a cheaper or deterministic fallback;
// workerID always reflects whichever worker actually produced the score.
func ScoreAnswer(ctx context.Context, registry *tlaloque.Registry, in ScoreInput) (out ScoreOutput, workerID string, err error) {
	inputRaw, err := json.Marshal(in)
	if err != nil {
		return ScoreOutput{}, "", err
	}
	req := tlaloque.CapabilityRequest{Input: inputRaw}

	for _, candidateID := range preferredWorkerOrder {
		worker, ok := registry.Get(candidateID)
		if !ok {
			continue
		}
		resp, execErr := worker.Execute(ctx, req)
		if execErr != nil {
			continue
		}
		var scored ScoreOutput
		if err := json.Unmarshal(resp.Output, &scored); err != nil {
			continue
		}
		return scored, resp.WorkerID, nil
	}

	fallbackWorker, ok := registry.Get(KeywordOverlapWorkerID)
	if !ok {
		return ScoreOutput{}, "", fmt.Errorf("answerscore: no %q worker registered", KeywordOverlapWorkerID)
	}
	resp, err := fallbackWorker.Execute(ctx, req)
	if err != nil {
		return ScoreOutput{}, "", err
	}
	var scored ScoreOutput
	if err := json.Unmarshal(resp.Output, &scored); err != nil {
		return ScoreOutput{}, "", err
	}
	return scored, resp.WorkerID, nil
}
