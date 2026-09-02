package promptgenome

import (
	"fmt"
	"sort"
	"strings"
)

func Compile(req CompileRequest) (Compiled, error) {
	g := req.Genome
	if g.Schema != GenomeSchemaR1 {
		return Compiled{}, fmt.Errorf("unexpected genome schema %q", g.Schema)
	}
	if strings.TrimSpace(g.ID) == "" {
		return Compiled{}, fmt.Errorf("genome id is required")
	}

	relevant := map[string]bool{}
	for _, id := range req.RelevantModules {
		relevant[id] = true
	}
	mods := append([]Module(nil), g.Modules...)
	sort.SliceStable(mods, func(i, j int) bool {
		if mods[i].Priority != mods[j].Priority {
			return mods[i].Priority > mods[j].Priority
		}
		return mods[i].ID < mods[j].ID
	})

	selected := []CompiledModule{}
	omitted := []string{}
	used := 0
	for _, m := range mods {
		if !appliesToModel(m, req.Model) {
			omitted = append(omitted, m.ID)
			continue
		}
		if len(relevant) > 0 && !m.Required && !relevant[m.ID] {
			omitted = append(omitted, m.ID)
			continue
		}
		text := strings.TrimSpace(m.Text)
		mode := "FULL"
		if text == "" {
			omitted = append(omitted, m.ID)
			continue
		}
		need := len(text)
		if used > 0 {
			need += 2
		}
		if req.MaxChars > 0 && used+need > req.MaxChars {
			min := strings.TrimSpace(m.MinText)
			if min != "" {
				need = len(min)
				if used > 0 {
					need += 2
				}
				if used+need <= req.MaxChars {
					text = min
					mode = "MIN"
				} else if m.Required {
					return Compiled{}, fmt.Errorf("required module %s exceeds prompt budget", m.ID)
				} else {
					omitted = append(omitted, m.ID)
					continue
				}
			} else if m.Required {
				return Compiled{}, fmt.Errorf("required module %s exceeds prompt budget", m.ID)
			} else {
				omitted = append(omitted, m.ID)
				continue
			}
		}
		selected = append(selected, CompiledModule{ID: m.ID, Version: m.Version, Text: text, Mode: mode})
		used += len(text)
		if len(selected) > 1 {
			used += 2
		}
	}

	// Dependencies must survive selection.
	selectedSet := map[string]bool{}
	for _, m := range selected {
		selectedSet[m.ID] = true
	}
	byID := map[string]Module{}
	for _, m := range g.Modules {
		byID[m.ID] = m
	}
	for _, cm := range selected {
		for _, dep := range byID[cm.ID].Dependencies {
			if !selectedSet[dep] {
				return Compiled{}, fmt.Errorf("module %s requires omitted dependency %s", cm.ID, dep)
			}
		}
	}

	parts := make([]string, 0, len(selected))
	for _, m := range selected {
		parts = append(parts, m.Text)
	}
	prompt := strings.Join(parts, "\n\n")
	sort.Strings(omitted)
	return Compiled{Schema: CompiledSchemaR1, GenomeID: g.ID, GenomeVersion: g.Version, Model: req.Model, Modules: selected, Prompt: prompt, Chars: len(prompt), Omitted: omitted, Authority: "EXPERIMENTAL_PROMPT_COMPILATION"}, nil
}

func appliesToModel(m Module, model string) bool {
	if len(m.ModelScopes) == 0 {
		return true
	}
	for _, s := range m.ModelScopes {
		if s == "*" || strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(model)) {
			return true
		}
	}
	return false
}
