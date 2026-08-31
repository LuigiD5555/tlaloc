package target

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	GenerationGuardNone         = "none"
	GenerationGuardRepetitionR0 = "repetition-r0"
	GenerationFailureClass      = "MODEL_OUTPUT_DEGENERATION"
)

type GenerationGuard interface {
	Name() string
	Check(accumulated string) error
}

type GenerationPolicy struct {
	MaxTokens int
	Guard     GenerationGuard
}

type generationPolicyContextKey struct{}

func WithGenerationPolicy(ctx context.Context, policy GenerationPolicy) context.Context {
	return context.WithValue(ctx, generationPolicyContextKey{}, policy)
}

func GenerationPolicyFromContext(ctx context.Context) GenerationPolicy {
	if ctx == nil {
		return GenerationPolicy{}
	}
	policy, _ := ctx.Value(generationPolicyContextKey{}).(GenerationPolicy)
	return policy
}

type GenerationDegenerationError struct {
	Guard   string
	Reason  string
	Partial string
}

func (e *GenerationDegenerationError) Error() string {
	if e == nil {
		return GenerationFailureClass
	}
	return fmt.Sprintf("%s guard=%s: %s", GenerationFailureClass, e.Guard, e.Reason)
}

func AsGenerationDegeneration(err error) (*GenerationDegenerationError, bool) {
	var target *GenerationDegenerationError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

type RepetitionGuard struct {
	MinRepeatedLines int
}

func (g RepetitionGuard) Name() string { return GenerationGuardRepetitionR0 }

func (g RepetitionGuard) Check(accumulated string) error {
	min := g.MinRepeatedLines
	if min <= 0 {
		min = 16
	}
	lines := nonEmptyLines(accumulated)
	if len(lines) < min {
		return nil
	}
	start := len(lines) - min
	shape := repetitionShape(lines[start])
	if shape == "" {
		return nil
	}
	for _, line := range lines[start+1:] {
		if repetitionShape(line) != shape {
			return nil
		}
	}
	return &GenerationDegenerationError{
		Guard:   g.Name(),
		Reason:  fmt.Sprintf("%d consecutive near-identical lines differ only by numeric progression", min),
		Partial: accumulated,
	}
}

func ResolveGenerationGuard(name string) (GenerationGuard, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", GenerationGuardNone:
		return nil, nil
	case GenerationGuardRepetitionR0:
		return RepetitionGuard{MinRepeatedLines: 16}, nil
	default:
		return nil, fmt.Errorf("unknown generation guard %q", name)
	}
}

var digitRun = regexp.MustCompile(`\d+`)
var whitespaceRun = regexp.MustCompile(`\s+`)

func repetitionShape(line string) string {
	line = strings.ToLower(strings.TrimSpace(line))
	line = digitRun.ReplaceAllString(line, "#")
	line = whitespaceRun.ReplaceAllString(line, " ")
	return line
}

func nonEmptyLines(text string) []string {
	raw := strings.Split(text, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
