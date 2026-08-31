package target

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestRepetitionGuardDetectsNumericProgression(t *testing.T) {
	lines := []string{"After B becomes ACTIVE:"}
	for i := 1; i <= 16; i++ {
		lines = append(lines, fmt.Sprintf("%d. The T%d checkpoints are updated to reflect the new value of B.", i, i+7))
	}
	guard := RepetitionGuard{MinRepeatedLines: 16}
	err := guard.Check(strings.Join(lines, "\n"))
	degeneration, ok := AsGenerationDegeneration(err)
	if !ok {
		t.Fatalf("expected GenerationDegenerationError, got %v", err)
	}
	if degeneration.Guard != GenerationGuardRepetitionR0 {
		t.Fatalf("guard=%q", degeneration.Guard)
	}
	if degeneration.Partial == "" {
		t.Fatal("expected partial output to be preserved")
	}
}

func TestRepetitionGuardAllowsNormalAnswer(t *testing.T) {
	answer := "A starts ACTIVE.\nB starts IDLE.\nC starts IDLE.\nThen the declared rules are applied synchronously."
	if err := (RepetitionGuard{MinRepeatedLines: 4}).Check(answer); err != nil {
		t.Fatalf("unexpected guard error: %v", err)
	}
}

func TestGenerationPolicyContext(t *testing.T) {
	guard := RepetitionGuard{MinRepeatedLines: 16}
	ctx := WithGenerationPolicy(context.Background(), GenerationPolicy{MaxTokens: 512, Guard: guard})
	policy := GenerationPolicyFromContext(ctx)
	if policy.MaxTokens != 512 {
		t.Fatalf("max tokens=%d", policy.MaxTokens)
	}
	if policy.Guard == nil || policy.Guard.Name() != GenerationGuardRepetitionR0 {
		t.Fatalf("guard=%v", policy.Guard)
	}
}
