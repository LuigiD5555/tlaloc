package parrotlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CompetenceEnvelope is the primary Phase P deliverable (SPEC §40).
type CompetenceEnvelope struct {
	Model                        string       `json:"model"`
	ExperimentID                 string       `json:"experiment_id"`
	SafeInstructionDepth         int          `json:"safe_instruction_depth"` // = contract
	SafeInstructionDepthContract int          `json:"safe_instruction_depth_contract"`
	SafeInstructionDepthSemantic int          `json:"safe_instruction_depth_semantic"`
	InstructionCliff             *CliffResult `json:"instruction_cliff,omitempty"`
	Strong                       []string     `json:"strong"`
	Usable                       []string     `json:"usable"`
	Fragile                      []string     `json:"fragile"`
	Weak                         []string     `json:"weak"`
	Unusable                     []string     `json:"unusable"`
	SevereInterference           [][2]string  `json:"severe_interference"`
	ExternalizeFirst             []string     `json:"externalize_first"`
	MissingStages                []string     `json:"missing_stages,omitempty"`
}

// BuildEnvelope assembles the competence envelope from whatever results/*.json
// exist. Missing stages are reported, not fatal, so a partial envelope can be
// inspected mid-campaign.
func BuildEnvelope(exp *Experiment) (CompetenceEnvelope, error) {
	envelope := CompetenceEnvelope{
		Model:              exp.Model.ID,
		ExperimentID:       exp.Manifest.ExperimentID,
		Strong:             []string{},
		Usable:             []string{},
		Fragile:            []string{},
		Weak:               []string{},
		Unusable:           []string{},
		SevereInterference: [][2]string{},
		ExternalizeFirst:   []string{},
	}

	cliff, err := readStageResult(exp, StageInstructionCliff)
	if err == nil && cliff.Cliff != nil {
		envelope.SafeInstructionDepth = cliff.Cliff.MaxSafeOpsContract
		envelope.SafeInstructionDepthContract = cliff.Cliff.MaxSafeOpsContract
		envelope.SafeInstructionDepthSemantic = cliff.Cliff.MaxSafeOpsSemantic
		envelope.InstructionCliff = cliff.Cliff
	} else {
		envelope.MissingStages = append(envelope.MissingStages, StageInstructionCliff)
	}

	interferenceScore := map[string]float64{}
	singles, err := readStageResult(exp, StageSingles)
	if err == nil {
		names := make([]string, 0, len(singles.Capability))
		for name := range singles.Capability {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			verdict := singles.Capability[name]
			switch verdict.Class {
			case "STRONG":
				envelope.Strong = append(envelope.Strong, name)
			case "USABLE":
				envelope.Usable = append(envelope.Usable, name)
			case "FRAGILE":
				envelope.Fragile = append(envelope.Fragile, name)
			case "WEAK":
				envelope.Weak = append(envelope.Weak, name)
			case "UNUSABLE":
				envelope.Unusable = append(envelope.Unusable, name)
			}
			if verdict.ExternalizeCandidate {
				interferenceScore[name] += 1 - verdict.Accuracy
			}
		}
	} else {
		envelope.MissingStages = append(envelope.MissingStages, StageSingles)
	}

	interference, err := readStageResult(exp, StageInterference)
	if err == nil {
		for _, pair := range interference.Interference {
			if pair.Category == "SEVERE" {
				envelope.SevereInterference = append(envelope.SevereInterference, pair.Pair)
				interferenceScore[pair.Pair[0]] += -pair.PairInterference
				interferenceScore[pair.Pair[1]] += -pair.PairInterference
			}
		}
	} else {
		envelope.MissingStages = append(envelope.MissingStages, StageInterference)
	}

	ranked := make([]string, 0, len(interferenceScore))
	for name := range interferenceScore {
		ranked = append(ranked, name)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if interferenceScore[ranked[i]] != interferenceScore[ranked[j]] {
			return interferenceScore[ranked[i]] > interferenceScore[ranked[j]]
		}
		return ranked[i] < ranked[j]
	})
	envelope.ExternalizeFirst = ranked

	return envelope, nil
}

func readStageResult(exp *Experiment, stage string) (StageResult, error) {
	raw, err := os.ReadFile(filepath.Join(exp.Root, "results", stage+".json"))
	if err != nil {
		return StageResult{}, err
	}
	var result StageResult
	return result, json.Unmarshal(raw, &result)
}

// WriteEnvelope persists the envelope JSON and a human REPORT.md.
func WriteEnvelope(exp *Experiment, envelope CompetenceEnvelope) (string, error) {
	dir := filepath.Join(exp.Root, "results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "PARROT_COMPETENCE_ENVELOPE_R0.json")
	raw, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, os.WriteFile(filepath.Join(exp.Root, "REPORT.md"), []byte(renderReport(envelope)), 0o644)
}

func renderReport(envelope CompetenceEnvelope) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Parrot Competence Envelope — %s\n\n", envelope.ExperimentID)
	fmt.Fprintf(&builder, "Model: `%s`\n\n", envelope.Model)
	if len(envelope.MissingStages) > 0 {
		fmt.Fprintf(&builder, "> Partial: missing stages %s\n\n", strings.Join(envelope.MissingStages, ", "))
	}
	fmt.Fprintf(&builder, "## Safe instruction depth\n\n`PARROT_MAX_SAFE_OPS_CONTRACT = %d`\n`PARROT_MAX_SAFE_OPS_SEMANTIC = %d`\n\n",
		envelope.SafeInstructionDepthContract, envelope.SafeInstructionDepthSemantic)
	if envelope.SafeInstructionDepthSemantic > envelope.SafeInstructionDepthContract {
		builder.WriteString("Semantic > contract: Parrot keeps the answer right further than it keeps the form right — externalise format/control before decomposing tasks.\n\n")
	}
	if envelope.InstructionCliff != nil && envelope.InstructionCliff.Detected {
		fmt.Fprintf(&builder, "Cliff detected at OP%d.\n\n", envelope.InstructionCliff.Level)
	}
	section := func(title string, items []string) {
		fmt.Fprintf(&builder, "## %s\n\n", title)
		if len(items) == 0 {
			builder.WriteString("_none_\n\n")
			return
		}
		for _, item := range items {
			fmt.Fprintf(&builder, "- %s\n", item)
		}
		builder.WriteString("\n")
	}
	section("STRONG", envelope.Strong)
	section("USABLE", envelope.Usable)
	section("FRAGILE", envelope.Fragile)
	section("WEAK", envelope.Weak)
	section("UNUSABLE", envelope.Unusable)
	fmt.Fprintf(&builder, "## Severe interference pairs\n\n")
	if len(envelope.SevereInterference) == 0 {
		builder.WriteString("_none_\n\n")
	}
	for _, pair := range envelope.SevereInterference {
		fmt.Fprintf(&builder, "- %s + %s\n", pair[0], pair[1])
	}
	builder.WriteString("\n")
	section("Externalize first (ranked)", envelope.ExternalizeFirst)
	return builder.String()
}
