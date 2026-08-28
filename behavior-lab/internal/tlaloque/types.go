package tlaloque

import (
	"context"

	"tlaloc.local/behaviorlab/internal/compiler"
	"tlaloc.local/behaviorlab/internal/evaluate"
)

// Case is one behavior-training case evaluated against reference semantics.
type Case struct {
	ID          string
	User        string
	ExpectedRaw string
}

// Model is the target model interface used by the Behavior Lab.
type Model interface {
	Complete(ctx context.Context, systemPrompt, user string) (string, error)
}

// Tlaloque is a bounded specialist under Tlaloc. It may diagnose a structured
// finding and propose a compiled-artifact patch, but it has no promotion authority.
type Tlaloque interface {
	Name() string
	Propose(findings []evaluate.Finding) []compiler.Section
}

type Generation struct {
	Index   int
	Score   float64
	Passed  int
	Failed  int
	Patches []compiler.Section
	Prompt  string
}
