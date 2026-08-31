package realcampaign

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tlaloc.local/behaviorlab/internal/target"
)

const parentProfile = "origami.temporal-carrier.r0.profile-1"

func Normalize(spec Spec) (Spec, error) {
	if spec.Schema != "" && spec.Schema != SpecSchema {
		return Spec{}, fmt.Errorf("unexpected schema %q", spec.Schema)
	}
	spec.Schema = SpecSchema
	spec.Phase = strings.ToUpper(strings.TrimSpace(spec.Phase))
	if spec.Phase == "" {
		spec.Phase = PhaseSmoke
	}
	if spec.Phase != PhaseSmoke && spec.Phase != PhaseEvidence {
		return Spec{}, fmt.Errorf("unsupported phase %q", spec.Phase)
	}
	if strings.TrimSpace(spec.CampaignID) == "" {
		return Spec{}, fmt.Errorf("campaign_id is required")
	}
	if strings.TrimSpace(spec.Endpoint) == "" {
		spec.Endpoint = "http://127.0.0.1:1234/v1"
	}
	spec.Endpoint = strings.TrimRight(spec.Endpoint, "/")
	if strings.TrimSpace(spec.Compatibility) == "" {
		spec.Compatibility = target.CompatibilityLMStudio
	}
	strategy, err := target.ResolveMultimodalCompatibility(spec.Compatibility)
	if err != nil {
		return Spec{}, err
	}
	spec.Compatibility = strategy.Name()
	spec.TransportCondition = NormalizeTransportCondition(spec.TransportCondition, spec.Compatibility)
	if spec.MaxOutputTokens <= 0 {
		spec.MaxOutputTokens = 512
	}
	if strings.TrimSpace(spec.GenerationGuard) == "" {
		spec.GenerationGuard = target.GenerationGuardRepetitionR0
	}
	guard, err := target.ResolveGenerationGuard(spec.GenerationGuard)
	if err != nil {
		return Spec{}, err
	}
	if guard == nil {
		spec.GenerationGuard = target.GenerationGuardNone
	} else {
		spec.GenerationGuard = guard.Name()
	}
	if strings.TrimSpace(spec.InteropMemoryRoot) == "" {
		spec.InteropMemoryRoot = DefaultInteropMemoryRoot()
	}
	for name, value := range map[string]string{
		"program": spec.Program,
		"temporal_carrier": spec.TemporalCarrier,
		"candidate_builder": spec.CandidateBuilder,
		"output_dir": spec.OutputDir,
	} {
		if strings.TrimSpace(value) == "" {
			return Spec{}, fmt.Errorf("%s is required", name)
		}
	}
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = 180
	}
	if spec.TransportRetries < 0 {
		spec.TransportRetries = 0
	}
	if spec.Phase == PhaseSmoke {
		if spec.TrialsPerModel <= 0 {
			spec.TrialsPerModel = 1
		}
		if spec.CandidatesPerGen <= 0 {
			spec.CandidatesPerGen = 1
		}
		if spec.MaxGenerations <= 0 {
			spec.MaxGenerations = 1
		}
		if len(spec.Conditions) == 0 {
			spec.Conditions = []string{"NATIVE_PNG_ONLY"}
		}
	} else {
		if spec.TrialsPerModel <= 0 {
			spec.TrialsPerModel = 3
		}
		if spec.TrialsPerModel < 3 {
			return Spec{}, fmt.Errorf("EVIDENCE requires trials_per_model >= 3")
		}
		if spec.CandidatesPerGen <= 0 {
			spec.CandidatesPerGen = 2
		}
		if spec.MaxGenerations <= 0 {
			spec.MaxGenerations = 3
		}
		if len(spec.Conditions) == 0 {
			spec.Conditions = []string{"NATIVE_PNG_ONLY"}
			if spec.MasterPrompt != "" {
				spec.Conditions = append(spec.Conditions, "R4_ASSISTED")
			}
		}
	}
	if spec.Model != "" {
		if err := validateRealModelID(spec.Model); err != nil {
			return Spec{}, err
		}
	}
	if err := validateCanonicalSignalChain(spec.Program); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

type signalChainProgram struct {
	Schema string `json:"schema"`
	ID     string `json:"id"`
	Automaton struct {
		Schema string `json:"schema"`
		ID     string `json:"id"`
		Cells  []struct {
			ID           string   `json:"id"`
			Kind         string   `json:"kind"`
			InitialState string   `json:"initial_state"`
			Neighbors    []string `json:"neighbors"`
		} `json:"cells"`
		Rules []struct {
			ID         string `json:"id"`
			TargetCell string `json:"target_cell"`
			FromState  string `json:"from_state"`
			ToState    string `json:"to_state"`
			Requires   []struct {
				CellID string `json:"cell_id"`
				State  string `json:"state"`
			} `json:"requires"`
		} `json:"rules"`
	} `json:"automaton"`
	MaxSteps        int `json:"max_steps"`
	CheckpointEvery int `json:"checkpoint_every"`
}

func validateCanonicalSignalChain(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var p signalChainProgram
	if err := json.Unmarshal(body, &p); err != nil {
		return fmt.Errorf("program JSON: %w", err)
	}
	if p.Schema != "origami.temporal-program.r0" || p.ID != "signal-chain-r0" || p.Automaton.Schema != "origami.automaton.r0" || p.Automaton.ID != "signal-chain" || p.MaxSteps != 8 || p.CheckpointEvery != 2 {
		return fmt.Errorf("program is not the canonical signal-chain benchmark fixture")
	}
	cells := map[string]string{}
	for _, c := range p.Automaton.Cells {
		cells[c.ID] = c.InitialState
	}
	if len(cells) != 3 || cells["A"] != "ACTIVE" || cells["B"] != "IDLE" || cells["C"] != "IDLE" {
		return fmt.Errorf("program signal-chain cells do not match benchmark ground truth")
	}
	type transition struct{ from, to, fromState, toState string }
	required := map[transition]bool{
		{"A", "B", "IDLE", "ACTIVE"}: false,
		{"B", "A", "ACTIVE", "DONE"}: false,
		{"B", "C", "IDLE", "ACTIVE"}: false,
		{"C", "B", "ACTIVE", "DONE"}: false,
	}
	for _, r := range p.Automaton.Rules {
		if len(r.Requires) == 0 {
			continue
		}
		key := transition{r.Requires[0].CellID, r.TargetCell, r.FromState, r.ToState}
		if _, ok := required[key]; ok {
			required[key] = true
		}
	}
	for k, ok := range required {
		if !ok {
			return fmt.Errorf("program signal-chain transition missing: %s -> %s (%s -> %s)", k.from, k.to, k.fromState, k.toState)
		}
	}
	return nil
}

func validateRealModelID(id string) error {
	u := strings.ToUpper(strings.TrimSpace(id))
	if u == "" {
		return nil
	}
	if strings.HasPrefix(u, "SYNTHETIC") || strings.Contains(u, "REPLACE_WITH") || strings.Contains(u, "PLACEHOLDER") {
		return fmt.Errorf("model id %q is not admissible for Real VLM Campaign R0", id)
	}
	return nil
}
