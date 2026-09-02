// Package semanticverify adapts internal/tlaloque/answerscore (the
// SCORE_ANSWER_RELEVANCE Tlaloque: chat judge -> embedding -> deterministic
// keyword overlap, in that trust order) to the verify.SemanticVerifier
// interface, so the Verification Spine's SEMANTIC level is an independent
// specialist checking whether the evidence supports the claim.
package semanticverify

import (
	"context"

	"tlaloc.local/behaviorlab/internal/tlaloque"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
)

// Answerscore is a verify.SemanticVerifier backed by an answerscore
// registry (build it with answerscore.NewRegistry(model, baseURL,
// embeddingServiceURL)).
type Answerscore struct {
	Registry *tlaloque.Registry
	// AgreeThreshold: a relevance score at or above this counts as the
	// specialist agreeing the evidence supports the claim. Zero -> 0.6.
	AgreeThreshold float64
}

// AgreesWith scores how well claim is supported by evidence and reports
// agreement plus the scorer's own confidence. A scoring error is
// propagated so the spine fails closed.
func (verifier Answerscore) AgreesWith(ctx context.Context, claim, evidence string) (bool, float64, error) {
	threshold := verifier.AgreeThreshold
	if threshold <= 0 {
		threshold = 0.6
	}
	out, _, err := answerscore.ScoreAnswer(ctx, verifier.Registry, answerscore.ScoreInput{
		ModelAnswer: claim,
		PageContent: evidence,
	})
	if err != nil {
		return false, 0, err
	}
	return out.Score >= threshold, out.Confidence, nil
}
