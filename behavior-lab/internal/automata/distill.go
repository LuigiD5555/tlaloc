package automata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func Distill(trace ActionTrace) (Result, error) {
	if trace.Schema != TraceSchema {
		return Result{}, fmt.Errorf("unsupported trace schema %q", trace.Schema)
	}
	if strings.TrimSpace(trace.ID) == "" {
		return Result{}, fmt.Errorf("trace id is required")
	}
	if len(trace.Steps) == 0 {
		return Result{}, fmt.Errorf("trace has no steps")
	}

	steps := append([]TraceStep(nil), trace.Steps...)
	for i := range steps {
		steps[i].Tlaloque = strings.TrimSpace(steps[i].Tlaloque)
		steps[i].FromState = strings.TrimSpace(steps[i].FromState)
		steps[i].ToState = strings.TrimSpace(steps[i].ToState)
		if steps[i].Step < 0 || steps[i].Tlaloque == "" || steps[i].FromState == "" || steps[i].ToState == "" {
			return Result{}, fmt.Errorf("invalid trace step at index %d", i)
		}
		steps[i].Requires = canonicalPredicates(steps[i].Requires)
		steps[i].EmitsTo = canonicalStrings(steps[i].EmitsTo)
	}
	sort.SliceStable(steps, func(i, j int) bool {
		if steps[i].Step != steps[j].Step { return steps[i].Step < steps[j].Step }
		if steps[i].Tlaloque != steps[j].Tlaloque { return steps[i].Tlaloque < steps[j].Tlaloque }
		if steps[i].FromState != steps[j].FromState { return steps[i].FromState < steps[j].FromState }
		return steps[i].ToState < steps[j].ToState
	})

	initial := map[string]string{}
	kinds := map[string]string{}
	neighbors := map[string]map[string]bool{}
	ensureCell := func(id, kind string) {
		id = strings.TrimSpace(id)
		if id == "" { return }
		if _, ok := kinds[id]; !ok { kinds[id] = kind }
		if neighbors[id] == nil { neighbors[id] = map[string]bool{} }
		if _, ok := initial[id]; !ok { initial[id] = "UNKNOWN" }
	}

	ruleByKey := map[string]Rule{}
	edgeByKey := map[string]Edge{}
	for _, s := range steps {
		ensureCell(s.Tlaloque, "TLALOQUE_DISTILLED_CELL")
		if initial[s.Tlaloque] == "UNKNOWN" { initial[s.Tlaloque] = s.FromState }
		for _, p := range s.Requires {
			ensureCell(p.CellID, "DECLARED_DEPENDENCY")
			if p.CellID != s.Tlaloque {
				neighbors[s.Tlaloque][p.CellID] = true
				neighbors[p.CellID][s.Tlaloque] = true
				key := p.CellID + "\x00" + s.Tlaloque + "\x00DEPENDENCY"
				edgeByKey[key] = Edge{From: p.CellID, To: s.Tlaloque, Kind: "DEPENDENCY"}
			}
		}
		for _, dst := range s.EmitsTo {
			ensureCell(dst, "DECLARED_RECEIVER")
			if dst != s.Tlaloque {
				neighbors[s.Tlaloque][dst] = true
				neighbors[dst][s.Tlaloque] = true
				key := s.Tlaloque + "\x00" + dst + "\x00EMISSION"
				edgeByKey[key] = Edge{From: s.Tlaloque, To: dst, Kind: "EMISSION"}
			}
		}

		keyBytes, _ := json.Marshal(struct {
			Target string      `json:"target"`
			From   string      `json:"from"`
			To     string      `json:"to"`
			Req    []Predicate `json:"requires"`
		}{s.Tlaloque, s.FromState, s.ToState, s.Requires})
		digest := sha256.Sum256(keyBytes)
		key := string(keyBytes)
		if _, ok := ruleByKey[key]; !ok {
			ruleByKey[key] = Rule{ID: "r-" + hex.EncodeToString(digest[:6]), TargetCell: s.Tlaloque, FromState: s.FromState, ToState: s.ToState, Requires: append([]Predicate(nil), s.Requires...)}
		}
	}

	cellIDs := make([]string, 0, len(kinds))
	for id := range kinds { cellIDs = append(cellIDs, id) }
	sort.Strings(cellIDs)
	cells := make([]Cell, 0, len(cellIDs))
	for _, id := range cellIDs {
		ns := make([]string, 0, len(neighbors[id]))
		for n := range neighbors[id] { ns = append(ns, n) }
		sort.Strings(ns)
		cells = append(cells, Cell{ID: id, Kind: kinds[id], InitialState: initial[id], Neighbors: ns})
	}

	rules := make([]Rule, 0, len(ruleByKey))
	for _, r := range ruleByKey { rules = append(rules, r) }
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	edges := make([]Edge, 0, len(edgeByKey))
	for _, e := range edgeByKey { edges = append(edges, e) }
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From { return edges[i].From < edges[j].From }
		if edges[i].To != edges[j].To { return edges[i].To < edges[j].To }
		return edges[i].Kind < edges[j].Kind
	})

	canonicalTrace := ActionTrace{Schema: TraceSchema, ID: trace.ID, Steps: steps}
	traceBytes, _ := json.Marshal(canonicalTrace)
	traceDigest := sha256.Sum256(traceBytes)
	ratio := 0.0
	if len(rules) > 0 { ratio = float64(len(steps)) / float64(len(rules)) }

	return Result{
		Schema: "tlaloc.automaton-distillation-result.r0",
		Automaton: AutomatonIR{Schema: AutomatonSchema, ID: trace.ID + "-distilled", Cells: cells, Rules: rules, Edges: edges, SourceTraceSHA256: hex.EncodeToString(traceDigest[:])},
		Metrics: Metrics{TraceSteps: len(steps), UniqueCells: len(cells), UniqueRules: len(rules), RepeatedTransitionsRemoved: len(steps)-len(rules), DistillationRatio: ratio},
	}, nil
}

func canonicalPredicates(in []Predicate) []Predicate {
	out := make([]Predicate, 0, len(in))
	seen := map[string]bool{}
	for _, p := range in {
		p.CellID = strings.TrimSpace(p.CellID); p.State = strings.TrimSpace(p.State)
		if p.CellID == "" || p.State == "" { continue }
		key := p.CellID + "\x00" + p.State
		if seen[key] { continue }
		seen[key] = true; out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { if out[i].CellID != out[j].CellID { return out[i].CellID < out[j].CellID }; return out[i].State < out[j].State })
	return out
}

func canonicalStrings(in []string) []string {
	seen := map[string]bool{}; out := make([]string, 0, len(in))
	for _, s := range in { s = strings.TrimSpace(s); if s != "" && !seen[s] { seen[s]=true; out=append(out,s) } }
	sort.Strings(out); return out
}
