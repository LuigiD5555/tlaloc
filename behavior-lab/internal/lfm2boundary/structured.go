package lfm2boundary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

const (
	RoleRosetta     = "rosetta"
	RoleCells       = "cells"
	RoleTransitions = "transitions"
	RoleTimeline    = "timeline"
)

var SpecialistRoles = []string{RoleRosetta, RoleCells, RoleTransitions, RoleTimeline}

type RosettaObservation struct {
	Box                  string `json:"box"`
	Arrow                string `json:"arrow"`
	Ring                 string `json:"ring"`
	XTime                string `json:"x_time"`
	SemanticFilmNotVideo bool   `json:"semantic_film_not_video"`
}

type ObservedCell struct {
	ID           string `json:"id"`
	InitialState string `json:"initial_state"`
}

type CellsObservation struct {
	Cells []ObservedCell `json:"cells"`
}

type ObservedRequirement struct {
	CellID string `json:"cell_id"`
	State  string `json:"state"`
}

type ObservedRule struct {
	Requires   []ObservedRequirement `json:"requires,omitempty"`
	TargetCell string                `json:"target_cell"`
	FromState  string                `json:"from_state"`
	ToState    string                `json:"to_state"`
}

type TransitionsObservation struct {
	Rules []ObservedRule `json:"rules"`
}

type TimelineObservation struct {
	Checkpoints []string `json:"checkpoints"`
}

func RoleFromNodeID(nodeID string) (string, bool) {
	upper := strings.ToUpper(strings.TrimSpace(nodeID))
	for _, role := range SpecialistRoles {
		prefix := strings.ToUpper(role) + "-"
		if strings.HasPrefix(upper, prefix) || upper == strings.ToUpper(role) {
			return role, true
		}
	}
	qid := QuestionIDFromNode(nodeID)
	return ResponsibilityForQuestion(qid)
}

func RoleCrop(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleRosetta:
		return "BOOT_T1"
	case RoleCells, RoleTransitions:
		return "T2"
	case RoleTimeline:
		return "TIMELINE"
	default:
		return ""
	}
}

func RolePrompt(role string) (string, error) {
	common := "Read only the supplied Origami visual crop. Return exactly one JSON object and no prose. Use only visible evidence. Do not use OCR, source code, ground truth, payload bytes, an exact decoder, or outside knowledge. If a field cannot be supported visually, leave its string empty or its array empty."
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleRosetta:
		return common + ` Schema: {"box":"visible meaning of BOX","arrow":"visible meaning of ARROW","ring":"visible meaning of RING","x_time":"visible meaning of X/TIME","semantic_film_not_video":true_or_false}.`, nil
	case RoleCells:
		return common + ` Schema: {"cells":[{"id":"visible cell id","initial_state":"visible initial state"}]}. Include every clearly visible cell and its initial state, once each.`, nil
	case RoleTransitions:
		return common + ` Schema: {"rules":[{"requires":[{"cell_id":"id","state":"required state"}],"target_cell":"id","from_state":"state before transition","to_state":"state after transition"}]}. Record only declared visible rules; graph adjacency alone is not a rule.`, nil
	case RoleTimeline:
		return common + ` Schema: {"checkpoints":["visible checkpoint label"]}. Preserve visible labels such as T0, T2, T4 when present.`, nil
	default:
		return "", fmt.Errorf("unknown specialist role %q", role)
	}
}

func extractJSONObject(text string) (json.RawMessage, bool) {
	text = strings.TrimSpace(text)
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end < start {
		return nil, false
	}
	raw := json.RawMessage(text[start : end+1])
	if !json.Valid(raw) {
		return nil, false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return canonical, true
}

// StructuredObservation preserves malformed model output as a JSON string.
// That means every replica still reaches the blackboard, while the role
// contract later turns any malformed/out-of-contract evidence into UNKNOWN.
func StructuredObservation(role, content string) json.RawMessage {
	if raw, ok := extractJSONObject(content); ok {
		return raw
	}
	body, _ := json.Marshal(strings.TrimSpace(content))
	return body
}

func roleContract(role string) blackboard.ValueContract {
	return func(raw json.RawMessage) bool {
		switch role {
		case RoleRosetta:
			var v RosettaObservation
			if err := strictUnmarshal(raw, &v); err != nil { return false }
			return strings.TrimSpace(v.Box) != "" && strings.TrimSpace(v.Arrow) != "" && strings.TrimSpace(v.Ring) != "" && strings.TrimSpace(v.XTime) != ""
		case RoleCells:
			var v CellsObservation
			if err := strictUnmarshal(raw, &v); err != nil || len(v.Cells) == 0 { return false }
			seen := map[string]bool{}
			for _, c := range v.Cells {
				id := norm(c.ID); state := norm(c.InitialState)
				if id == "" || state == "" || seen[id] { return false }
				seen[id] = true
			}
			return true
		case RoleTransitions:
			var v TransitionsObservation
			if err := strictUnmarshal(raw, &v); err != nil || len(v.Rules) == 0 { return false }
			for _, r := range v.Rules {
				if norm(r.TargetCell) == "" || norm(r.FromState) == "" || norm(r.ToState) == "" { return false }
				for _, req := range r.Requires { if norm(req.CellID) == "" || norm(req.State) == "" { return false } }
			}
			return true
		case RoleTimeline:
			var v TimelineObservation
			if err := strictUnmarshal(raw, &v); err != nil || len(v.Checkpoints) == 0 { return false }
			for _, cp := range v.Checkpoints { if norm(cp) == "" { return false } }
			return true
		default:
			return false
		}
	}
}

func strictUnmarshal(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil { return err }
	var extra any
	if err := dec.Decode(&extra); err == nil { return fmt.Errorf("extra JSON value") }
	return nil
}

func norm(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func decodeRole[T any](raw json.RawMessage) (T, error) {
	var out T
	if err := strictUnmarshal(raw, &out); err != nil { return out, err }
	return out, nil
}

// SimulateObservedRules executes only the observed rules, synchronously. Every
// rule in a step is evaluated against the same pre-step snapshot. Conflicting
// simultaneous writes, unknown cells, or a non-converging rule set return an
// error instead of inventing a state.
func SimulateObservedRules(cells CellsObservation, transitions TransitionsObservation) (map[string]string, error) {
	state := map[string]string{}
	for _, c := range cells.Cells {
		id := norm(c.ID); st := norm(c.InitialState)
		if id == "" || st == "" { return nil, fmt.Errorf("invalid cell observation") }
		if _, exists := state[id]; exists { return nil, fmt.Errorf("duplicate cell %s", id) }
		state[id] = st
	}
	if len(state) == 0 { return nil, fmt.Errorf("no observed cells") }
	for step := 0; step < 64; step++ {
		writes := map[string]string{}
		for _, rule := range transitions.Rules {
			target := norm(rule.TargetCell); from := norm(rule.FromState); to := norm(rule.ToState)
			current, ok := state[target]
			if !ok { return nil, fmt.Errorf("rule targets unknown cell %s", target) }
			if current != from { continue }
			applies := true
			for _, req := range rule.Requires {
				actual, ok := state[norm(req.CellID)]
				if !ok { return nil, fmt.Errorf("rule requires unknown cell %s", norm(req.CellID)) }
				if actual != norm(req.State) { applies = false; break }
			}
			if !applies || current == to { continue }
			if previous, exists := writes[target]; exists && previous != to {
				return nil, fmt.Errorf("conflicting synchronous writes for %s: %s vs %s", target, previous, to)
			}
			writes[target] = to
		}
		if len(writes) == 0 { return state, nil }
		for id, next := range writes { state[id] = next }
	}
	return nil, fmt.Errorf("observed rule set did not converge within 64 synchronous steps")
}

func formatStateMap(states map[string]string) string {
	ids := make([]string, 0, len(states)); for id := range states { ids = append(ids, id) }; sort.Strings(ids)
	parts := make([]string, 0, len(ids)); for _, id := range ids { parts = append(parts, id+" "+states[id]) }
	return strings.Join(parts, "; ")
}

func formatRequirements(reqs []ObservedRequirement) string {
	parts := make([]string,0,len(reqs)); for _, r := range reqs { parts=append(parts,norm(r.CellID)+" "+norm(r.State)) }; sort.Strings(parts); return strings.Join(parts, " AND ")
}

// SynthesizeBlackboardResponses converts confirmed structured visual evidence
// into the nine existing benchmark responses. Q7 is computed, not generated;
// Q8 is forced to the exactness boundary because this campaign never executes
// an exact decoder.
func SynthesizeBlackboardResponses(snapshot blackboard.Snapshot) (ConsolidatedOutput, error) {
	out := ConsolidatedOutput{Responses: map[string]string{}, Consensus: map[string]string{}}
	roleValues := map[string]json.RawMessage{}
	for _, role := range SpecialistRoles {
		c, err := blackboard.Consolidate(snapshot, role, roleContract(role)); if err != nil { return out, err }
		out.Consensus[role] = c.Status
		if c.Status == blackboard.ConsensusConfirmed { roleValues[role] = c.Value }
	}
	for i:=0;i<9;i++ { out.Responses[fmt.Sprintf("Q%d",i)] = "UNKNOWN" }
	out.Responses["Q8"] = "NOT VERIFIED: NO EXACT DECODER"

	if raw, ok := roleValues[RoleRosetta]; ok {
		v, err := decodeRole[RosettaObservation](raw); if err != nil { return out, err }
		out.Responses["Q0"] = fmt.Sprintf("BOX %s; ARROW %s; RING %s; X TIME %s", norm(v.Box), norm(v.Arrow), norm(v.Ring), norm(v.XTime))
		if v.SemanticFilmNotVideo { out.Responses["Q6"] = "NOT LITERAL VIDEO; TEMPORAL SEMANTIC FILM" }
	}
	var cells CellsObservation; cellsOK := false
	if raw, ok := roleValues[RoleCells]; ok {
		v, err := decodeRole[CellsObservation](raw); if err != nil { return out, err }; cells=v; cellsOK=true
		ids:=[]string{}; for _,c:=range v.Cells { ids=append(ids,norm(c.ID)); if norm(c.ID)=="A" { out.Responses["Q2"]="A "+norm(c.InitialState) } }; sort.Strings(ids); out.Responses["Q1"]=strings.Join(ids," ")
	}
	var transitions TransitionsObservation; transitionsOK := false
	if raw, ok := roleValues[RoleTransitions]; ok {
		v, err := decodeRole[TransitionsObservation](raw); if err != nil { return out, err }; transitions=v; transitionsOK=true
		q3:=[]string{};q4:=[]string{}
		for _,r:=range v.Rules {
			piece:=fmt.Sprintf("%s -> %s %s TO %s",formatRequirements(r.Requires),norm(r.TargetCell),norm(r.FromState),norm(r.ToState))
			if norm(r.TargetCell)=="B"&&norm(r.ToState)=="ACTIVE"{q3=append(q3,piece)}
			for _,req:=range r.Requires{if norm(req.CellID)=="B"&&norm(req.State)=="ACTIVE"{q4=append(q4,piece);break}}
		}
		if len(q3)>0{sort.Strings(q3);out.Responses["Q3"]=strings.Join(q3,"; ")};if len(q4)>0{sort.Strings(q4);out.Responses["Q4"]=strings.Join(q4,"; ")}
	}
	if raw, ok := roleValues[RoleTimeline]; ok {
		v, err := decodeRole[TimelineObservation](raw); if err != nil { return out, err }; cps:=[]string{}; for _,cp:=range v.Checkpoints{cps=append(cps,norm(cp))};sort.Strings(cps);out.Responses["Q5"]=strings.Join(cps," ")
	}
	if cellsOK && transitionsOK {
		states, err := SimulateObservedRules(cells, transitions)
		if err == nil { out.Responses["Q7"] = formatStateMap(states) }
	}
	return out, nil
}
