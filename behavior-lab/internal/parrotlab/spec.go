// Package parrotlab is the campaign layer for the Parrot Capability Lab
// (Tlaloc Phase P). It loads a frozen experiment directory, runs its stages
// against a small multimodal model through internal/target, scores the
// answers, and aggregates the results into a competence envelope.
//
// It deliberately owns no model transport, no scoring model and no
// representation: transport is internal/target, statistics are
// internal/tlaloque/calibration. Datasets are either deterministically
// generated (instruction_cliff) or authored against datasets/SCHEMA.md.
package parrotlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Stage identifiers, in mandatory execution order (SPEC §3).
const (
	StageEndToEnd         = "end_to_end"
	StageInstructionCliff = "instruction_cliff"
	StageSingles          = "singles"
	StageInterference     = "interference"
	StageCoalitions       = "coalitions"
	StageBlackboard       = "blackboard"
)

// StageOrder is the fixed order stages must be executed and reported in.
var StageOrder = []string{
	StageEndToEnd, StageInstructionCliff, StageSingles,
	StageInterference, StageCoalitions, StageBlackboard,
}

// Capabilities measured in R0 (SPEC §15).
var Capabilities = []string{
	"VISUAL_IDENTIFY", "VISUAL_LOCATE", "READ_SHORT_TEXT", "EXTRACT_ENTITY",
	"EXTRACT_NUMBER", "CLASSIFY_SIMPLE", "COMPARE_SIMPLE", "FOLLOW_REFERENCE",
	"SELECT_ACTION", "USE_BLACKBOARD_HINT", "ANSWER_FROM_EVIDENCE",
}

// ModelConfig is the frozen model configuration (MODEL.json).
type ModelConfig struct {
	ID                 string  `json:"id"`
	Alias              string  `json:"alias"`
	Endpoint           string  `json:"endpoint"`
	Compatibility      string  `json:"compatibility"`
	Temperature        float64 `json:"temperature"`
	TopP               float64 `json:"top_p"`
	MaxTokens          int     `json:"max_tokens"`
	Seed               *int    `json:"seed"`
	ContextSize        *int    `json:"context_size"`
	ImagePreprocessing string  `json:"image_preprocessing"`
	ModelFileHash      *string `json:"model_file_hash"`
	Runtime            string  `json:"runtime"`
	RuntimeVersion     *string `json:"runtime_version"`
	Notes              string  `json:"notes"`

	// Additive P2 instrument-identity fields (parrot-microisa-r0). Legacy
	// experiments omit them and load with zero values.
	Quantization            string            `json:"quantization,omitempty"`
	ModelFilePaths          []string          `json:"model_file_paths,omitempty"`
	ModelFileHashes         map[string]string `json:"model_file_hashes,omitempty"`
	ModelFileHashesMeasured bool              `json:"model_file_hashes_measured,omitempty"`
	RuntimeVersionMeasured  bool              `json:"runtime_version_measured,omitempty"`
}

// Prompt is the single frozen R0 template (PROMPT.txt): an id line followed
// by a "SYSTEM:" marker and the system prompt body.
type Prompt struct {
	ID     string
	System string
	Raw    string
}

// Experiment is a loaded experiment directory.
type Experiment struct {
	Root     string
	Manifest Manifest
	Model    ModelConfig
	Prompt   Prompt
}

// Manifest mirrors MANIFEST.json (only the fields the runner needs).
type Manifest struct {
	ExperimentID string `json:"experiment_id"`
	Stages       []struct {
		ID      string `json:"id"`
		Dataset string `json:"dataset"`
		Levels  []int  `json:"levels,omitempty"`
		Status  string `json:"status"`
	} `json:"stages"`
	Thresholds Thresholds `json:"thresholds"`
}

// Thresholds are the frozen decision cut-offs (SPEC §13, §20, §24).
type Thresholds struct {
	InstructionCliffDropPP float64              `json:"instruction_cliff_drop_pp"`
	CapabilityClass        CapabilityClassCuts  `json:"capability_class"`
	PairInterference       PairInterferenceCuts `json:"pair_interference"`
}

// CapabilityClassCuts are the CI-bound cut-offs for STRONG/USABLE/WEAK/UNUSABLE.
type CapabilityClassCuts struct {
	StrongLowerCI   float64 `json:"strong_lower_ci"`
	UsableLowerCI   float64 `json:"usable_lower_ci"`
	WeakUpperCI     float64 `json:"weak_upper_ci"`
	UnusableUpperCI float64 `json:"unusable_upper_ci"`
}

// PairInterferenceCuts are the upper bounds of the NEUTRAL/MILD/MODERATE bands
// (anything below Moderate is SEVERE).
type PairInterferenceCuts struct {
	Neutral  float64 `json:"neutral"`
	Mild     float64 `json:"mild"`
	Moderate float64 `json:"moderate"`
}

// Load reads MANIFEST.json, MODEL.json and PROMPT.txt from root.
func Load(root string) (*Experiment, error) {
	exp := &Experiment{Root: root}
	if err := readJSON(filepath.Join(root, "MANIFEST.json"), &exp.Manifest); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(root, "MODEL.json"), &exp.Model); err != nil {
		return nil, err
	}
	prompt, err := loadPrompt(filepath.Join(root, "PROMPT.txt"))
	if err != nil {
		return nil, err
	}
	exp.Prompt = prompt
	return exp, nil
}

func loadPrompt(path string) (Prompt, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Prompt{}, err
	}
	text := string(raw)
	marker := "SYSTEM:"
	index := strings.Index(text, marker)
	if index < 0 {
		return Prompt{}, fmt.Errorf("%s: missing %q marker", path, marker)
	}
	id := strings.TrimSpace(text[:index])
	if id == "" {
		return Prompt{}, fmt.Errorf("%s: missing prompt id line before %q", path, marker)
	}
	system := strings.TrimSpace(text[index+len(marker):])
	if system == "" {
		return Prompt{}, fmt.Errorf("%s: empty system prompt", path)
	}
	return Prompt{ID: id, System: system, Raw: text}, nil
}

// PromptHash is the SHA256 of the raw PROMPT.txt bytes.
func (exp *Experiment) PromptHash() string { return hashString(exp.Prompt.Raw) }

// StageDataset returns the dataset path (or directory, for singles) for a
// stage id, resolved against the experiment root.
func (exp *Experiment) StageDataset(stage string) (string, error) {
	for _, s := range exp.Manifest.Stages {
		if s.ID == stage {
			return filepath.Join(exp.Root, filepath.FromSlash(s.Dataset)), nil
		}
	}
	return "", fmt.Errorf("unknown stage %q", stage)
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hashString(text string) string { return hashBytes([]byte(text)) }
