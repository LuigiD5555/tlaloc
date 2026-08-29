package profiles

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"

	"tlaloc.local/behaviorlab/internal/compiler"
	"tlaloc.local/behaviorlab/internal/evaluate"
	"tlaloc.local/behaviorlab/internal/spec"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const CoherentID = "origami.quantum-inspired.r0"
const RelationalID = "origami.relational-core.r0"

//go:embed upstream/origami-relational-core-r0.json
var relationalFixtureData []byte

type Profile struct {
	ID      string
	Version string
	Cases   []tlaloque.Case
	Compare tlaloque.CompareFunc
	Agents  []tlaloque.Tlaloque
}
type Registry struct{ profiles map[string]Profile }
type fixtureCase struct {
	ID       string            `json:"id"`
	Input    map[string]string `json:"input"`
	Expected relationalResult  `json:"expected"`
}
type fixtureCampaign struct {
	Schema           string        `json:"schema"`
	Contract         string        `json:"contract"`
	SourceExperiment string        `json:"source_experiment"`
	Cases            []fixtureCase `json:"cases"`
}

func Builtin() Registry {
	registry := Registry{profiles: map[string]Profile{}}
	registry.register(coherentProfile())
	registry.register(relationalProfile())
	return registry
}
func (registry Registry) Lookup(id, version string) (Profile, error) {
	profile, ok := registry.profiles[id]
	if !ok {
		return Profile{}, fmt.Errorf("UNSUPPORTED profile %q", id)
	}
	if profile.Version != version {
		return Profile{}, fmt.Errorf("UNSUPPORTED profile version %q for %s; expected %s", version, id, profile.Version)
	}
	return profile, nil
}
func (registry Registry) IDs() []string {
	ids := make([]string, 0, len(registry.profiles))
	for id := range registry.profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func (registry Registry) register(profile Profile) {
	if _, exists := registry.profiles[profile.ID]; exists {
		panic("duplicate profile: " + profile.ID)
	}
	registry.profiles[profile.ID] = profile
}

func coherentProfile() Profile {
	return Profile{ID: CoherentID, Version: "0.1.0", Cases: []tlaloque.Case{
		{ID: "transform-no-collapse", User: `Initial state is SUPERPOSED with A=(0.7071067811865475,0), B=(0.7071067811865475,0). Apply TRANSFORM A->D and B->E, both multiplier (1,0). Return every required state field.`, ExpectedRaw: `{"kind":"superposed","branches":[{"label":"D","real":0.7071067811865475,"imag":0},{"label":"E","real":0.7071067811865475,"imag":0}],"members":[],"observed":"","unknown":false,"semantic":"PRESENT","notes":[]}`},
		{ID: "cancellation", User: `INTERFERE two contributions to C: +0.5 and -0.5. Preserve cancellation cause and return every required state field.`, ExpectedRaw: `{"kind":"superposed","branches":[],"members":[],"observed":"","unknown":false,"semantic":"CANCELLED","notes":["exact complex-amplitude cancellation"]}`},
		{ID: "coupled", User: `Create COUPLED members A,B with joint branches 00=(0.7071067811865475,0) and 11=(0.7071067811865475,0). Return every required state field.`, ExpectedRaw: `{"kind":"coupled","branches":[{"label":"00","real":0.7071067811865475,"imag":0},{"label":"11","real":0.7071067811865475,"imag":0}],"members":["A","B"],"observed":"","unknown":false,"semantic":"PRESENT","notes":[]}`},
	}, Agents: tlaloque.DefaultTlaloque()}
}

type relationalResult struct {
	Contract string            `json:"contract"`
	Outcome  string            `json:"outcome"`
	State    map[string]string `json:"state"`
	Trace    []relationalTrace `json:"trace"`
	Steps    *int              `json:"steps"`
	Evidence []string          `json:"evidence"`
	Errors   []string          `json:"errors"`
}
type relationalTrace struct {
	Step    int      `json:"step"`
	Applied []string `json:"applied"`
}

func decodeRelational(raw string) (relationalResult, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var result relationalResult
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return result, fmt.Errorf("multiple JSON values")
	}
	if result.Contract == "" || result.Outcome == "" || result.State == nil || result.Trace == nil || result.Steps == nil || result.Evidence == nil || result.Errors == nil {
		return result, fmt.Errorf("contract, outcome, state, trace, steps, evidence and errors are required")
	}
	return result, nil
}
func compareRelational(expectedRaw, actualRaw string) evaluate.Result {
	expected, err := decodeRelational(expectedRaw)
	if err != nil {
		return evaluate.Result{Findings: []evaluate.Finding{{Code: spec.StructuredOutputRequired, Message: "invalid fixture: " + err.Error()}}}
	}
	actual, err := decodeRelational(actualRaw)
	if err != nil {
		return evaluate.Result{Findings: []evaluate.Finding{{Code: spec.StructuredOutputRequired, Message: err.Error()}}}
	}
	var findings []evaluate.Finding
	if actual.Contract != expected.Contract {
		findings = append(findings, evaluate.Finding{Code: spec.ContractMismatch, Message: "contract differs from upstream fixture"})
	}
	if actual.Outcome != expected.Outcome {
		findings = append(findings, evaluate.Finding{Code: spec.OutcomeClassification, Message: "terminal outcome differs from upstream fixture"})
	}
	if !reflect.DeepEqual(actual.State, expected.State) {
		findings = append(findings, evaluate.Finding{Code: spec.RelationPreservation, Message: "terminal state differs from upstream fixture"})
	}
	if *actual.Steps != *expected.Steps {
		findings = append(findings, evaluate.Finding{Code: spec.BudgetTermination, Message: "step count differs from upstream fixture"})
	}
	if !reflect.DeepEqual(actual.Trace, expected.Trace) || !reflect.DeepEqual(actual.Evidence, expected.Evidence) || !reflect.DeepEqual(actual.Errors, expected.Errors) {
		findings = append(findings, evaluate.Finding{Code: spec.TraceRequired, Message: "trace or causal evidence differs from upstream fixture"})
	}
	score := 1.0 - float64(len(findings))*.25
	if score < 0 {
		score = 0
	}
	return evaluate.Result{Pass: len(findings) == 0, Score: score, Findings: findings}
}

func relationalProfile() Profile {
	cases := loadRelationalCases()
	agents := []tlaloque.Tlaloque{
		tlaloque.NewGuard("contract_guard", spec.ContractMismatch, compiler.Section{ID: "repair:relational-contract", Priority: 125, Text: "HARD REPAIR: Emit contract origami.relational-core.r0 exactly; never substitute another Origami profile."}),
		tlaloque.NewGuard("relation_guard", spec.RelationPreservation, compiler.Section{ID: "repair:relational-state", Priority: 122, Text: "HARD REPAIR: Apply all declared relations from the prior-round state simultaneously and preserve every semantic value exactly."}),
		tlaloque.NewGuard("outcome_guard", spec.OutcomeClassification, compiler.Section{ID: "repair:relational-outcome", Priority: 121, Text: "HARD REPAIR: Classify only FIXED_POINT, CYCLE, CONTRADICTION, or BUDGET_EXHAUSTED using the declared finite rules."}),
		tlaloque.NewGuard("budget_guard", spec.BudgetTermination, compiler.Section{ID: "repair:relational-budget", Priority: 120, Text: "HARD REPAIR: Stop at the declared budget and report the exact number of completed rounds."}),
		tlaloque.NewGuard("trace_guard", spec.TraceRequired, compiler.Section{ID: "repair:relational-trace", Priority: 123, Text: "HARD REPAIR: Return the ordered applied relation/rule IDs and causal evidence exactly for every completed round."}),
		tlaloque.NewGuard("relational_output_guard", spec.StructuredOutputRequired, compiler.Section{ID: "repair:relational-json", Priority: 126, Text: "HARD REPAIR: Return exactly one JSON object with contract, outcome, state, trace, steps, evidence, and errors; no extra fields or prose."}),
	}
	return Profile{ID: RelationalID, Version: "0.1.0", Compare: compareRelational, Agents: agents, Cases: cases}
}

func loadRelationalCases() []tlaloque.Case {
	var campaign fixtureCampaign
	if err := json.Unmarshal(relationalFixtureData, &campaign); err != nil {
		panic("invalid embedded Origami fixture: " + err.Error())
	}
	if campaign.Contract != RelationalID {
		panic("embedded Origami fixture contract mismatch")
	}
	cases := make([]tlaloque.Case, 0, len(campaign.Cases))
	for _, fixture := range campaign.Cases {
		input, _ := json.Marshal(fixture.Input)
		expected, _ := json.Marshal(fixture.Expected)
		cases = append(cases, tlaloque.Case{ID: fixture.ID, User: "Execute the declared origami.relational-core.r0 EXP-001 rules for initial state " + string(input) + " and return the strict result.", ExpectedRaw: string(expected)})
	}
	return cases
}
