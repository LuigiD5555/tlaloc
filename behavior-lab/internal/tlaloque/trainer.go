package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/compiler"
	"tlaloc.local/behaviorlab/internal/evaluate"
	"tlaloc.local/behaviorlab/internal/reference"
)

type Trainer struct {
	Model          Model
	Agents         []Tlaloque
	MaxGenerations int
	Compare        CompareFunc
}

func (t Trainer) Train(ctx context.Context, ir compiler.PromptIR, cases []Case) ([]Generation, compiler.PromptIR, error) {
	if t.Model == nil {
		return nil, ir, fmt.Errorf("model is required")
	}
	if len(t.Agents) == 0 {
		t.Agents = DefaultTlaloque()
	}
	if t.MaxGenerations <= 0 {
		t.MaxGenerations = 6
	}
	var history []Generation
	for gen := 0; gen < t.MaxGenerations; gen++ {
		prompt := compiler.Render(ir)
		var findings []evaluate.Finding
		passed := 0
		totalScore := 0.0
		for _, c := range cases {
			raw, err := t.Model.Complete(ctx, prompt, c.User)
			if err != nil {
				return history, ir, fmt.Errorf("case %s target: %w", c.ID, err)
			}
			var result evaluate.Result
			if t.Compare != nil {
				result = t.Compare(c.ExpectedRaw, raw)
			} else {
				var expected reference.State
				if err := json.Unmarshal([]byte(c.ExpectedRaw), &expected); err != nil {
					return history, ir, fmt.Errorf("case %s expected: %w", c.ID, err)
				}
				result = evaluate.Compare(expected, raw)
			}
			totalScore += result.Score
			if result.Pass {
				passed++
			} else {
				findings = append(findings, result.Findings...)
			}
		}
		score := 0.0
		if len(cases) > 0 {
			score = totalScore / float64(len(cases))
		}
		g := Generation{Index: gen, Score: score, Passed: passed, Failed: len(cases) - passed, Prompt: prompt}
		if len(findings) == 0 {
			history = append(history, g)
			return history, ir, nil
		}
		patches := coordinate(t.Agents, findings)
		g.Patches = patches
		history = append(history, g)
		if len(patches) == 0 {
			return history, ir, fmt.Errorf("no Tlaloque could repair remaining findings")
		}
		for _, p := range patches {
			ir = compiler.ApplyPatch(ir, p)
		}
	}
	return history, ir, nil
}

func coordinate(agents []Tlaloque, findings []evaluate.Finding) []compiler.Section {
	seen := map[string]bool{}
	var out []compiler.Section
	for _, w := range agents {
		for _, p := range w.Propose(findings) {
			if !seen[p.ID] {
				seen[p.ID] = true
				out = append(out, p)
			}
		}
	}
	return out
}
