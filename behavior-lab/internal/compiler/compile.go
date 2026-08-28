package compiler

import (
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/spec"
)

func BuildIR(s spec.BehaviorSpec, target string) (PromptIR, error) {
	if err := spec.Validate(s); err != nil {
		return PromptIR{}, err
	}
	ir := PromptIR{Version: "promptir-0.2", Target: target}
	identity := s.Identity
	if strings.TrimSpace(identity) == "" {
		identity = "Execute the active formal behavior contract exactly."
	}
	ir.Sections = append(ir.Sections, Section{ID: "identity", Priority: 100, Text: fmt.Sprintf("TLALOC CONTRACT %s: %s", s.ID, identity)})
	if len(s.StateKinds) > 0 {
		kinds := make([]string, 0, len(s.StateKinds))
		for _, k := range s.StateKinds { kinds = append(kinds, string(k)) }
		ir.Sections = append(ir.Sections, Section{ID: "state-kinds", Priority: 90, Text: "STATE KINDS: " + strings.Join(kinds, ", ")})
	}
	if len(s.Operations) > 0 {
		ops := make([]string, 0, len(s.Operations))
		for _, op := range s.Operations { ops = append(ops, string(op)) }
		ir.Sections = append(ir.Sections, Section{ID: "operations", Priority: 89, Text: "OPERATIONS: " + strings.Join(ops, ", ")})
	}
	for _, rule := range s.Rules {
		priority := rule.Priority
		if priority == 0 { priority = 90 }
		ir.Sections = append(ir.Sections, Section{ID: "rule:" + rule.Code, Priority: priority, Text: fmt.Sprintf("RULE %s: %s", rule.Code, rule.Description)})
	}
	for _, inv := range s.Invariants {
		ir.Sections = append(ir.Sections, Section{ID: "inv:" + string(inv.Code), Priority: 80 + inv.Severity, Text: fmt.Sprintf("INVARIANT %s: %s", inv.Code, inv.Description)})
	}
	output := "OUTPUT FORMAT: " + s.Output.Format
	if strings.TrimSpace(s.Output.Schema) != "" { output = s.Output.Schema }
	ir.Sections = append(ir.Sections, Section{ID: "output", Priority: 100, Text: output})
	return ir, nil
}

func Render(ir PromptIR) string {
	sections := append([]Section(nil), ir.Sections...)
	sort.SliceStable(sections, func(i, j int) bool { if sections[i].Priority == sections[j].Priority { return sections[i].ID < sections[j].ID }; return sections[i].Priority > sections[j].Priority })
	var b strings.Builder
	b.WriteString("TLALOC BEHAVIOR COMPILED PROMPT\n")
	b.WriteString("PromptIR: " + ir.Version + "\n")
	if ir.Target != "" { b.WriteString("Target: " + ir.Target + "\n") }
	b.WriteString("\n")
	for _, s := range sections { b.WriteString("[" + s.ID + "]\n" + s.Text + "\n\n") }
	return b.String()
}

func ApplyPatch(ir PromptIR, patch Section) PromptIR {
	for i := range ir.Sections { if ir.Sections[i].ID == patch.ID { ir.Sections[i] = patch; return ir } }
	ir.Sections = append(ir.Sections, patch)
	return ir
}
