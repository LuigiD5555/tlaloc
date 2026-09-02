package parrotlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FreezeState is the experiment's freeze ledger (FREEZE.json). The prompt
// and model are frozen once, globally; each stage dataset is frozen
// separately and never rewritten, so P3/P4 datasets can still be authored
// using what P1/P2 revealed without ever un-freezing an earlier stage
// (P-1 fix #6).
type FreezeState struct {
	Global GlobalFreeze           `json:"global"`
	Stages map[string]StageFreeze `json:"stages"`
}

type GlobalFreeze struct {
	Frozen           bool   `json:"frozen"`
	PromptFileHash   string `json:"prompt_file_hash"`
	SystemPromptHash string `json:"system_prompt_hash"`
	ModelHash        string `json:"model_hash"`
	FrozenAt         string `json:"frozen_at"`
}

type StageFreeze struct {
	DatasetHashes map[string]string `json:"dataset_hashes"`
	Cases         int               `json:"cases"`
	FrozenAt      string            `json:"frozen_at"`
}

const freezeFile = "FREEZE.json"

// LoadFreezeState reads FREEZE.json, returning a zero state when absent.
func LoadFreezeState(exp *Experiment) (FreezeState, error) {
	state := FreezeState{Stages: map[string]StageFreeze{}}
	raw, err := os.ReadFile(filepath.Join(exp.Root, freezeFile))
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, err
	}
	if state.Stages == nil {
		state.Stages = map[string]StageFreeze{}
	}
	return state, nil
}

func (state FreezeState) write(exp *Experiment) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(exp.Root, freezeFile), append(raw, '\n'), 0o644)
}

// FreezeGlobal locks the prompt and model configuration. It refuses to
// re-freeze; changing either after this is a deliberate new experiment.
func FreezeGlobal(exp *Experiment) (GlobalFreeze, error) {
	state, err := LoadFreezeState(exp)
	if err != nil {
		return GlobalFreeze{}, err
	}
	if state.Global.Frozen {
		return state.Global, fmt.Errorf("global config already frozen at %s", state.Global.FrozenAt)
	}
	modelRaw, err := os.ReadFile(filepath.Join(exp.Root, "MODEL.json"))
	if err != nil {
		return GlobalFreeze{}, err
	}
	state.Global = GlobalFreeze{
		Frozen:           true,
		PromptFileHash:   exp.PromptHash(),
		SystemPromptHash: hashString(exp.Prompt.System),
		ModelHash:        hashBytes(modelRaw),
		FrozenAt:         time.Now().UTC().Format(time.RFC3339),
	}
	if err := os.WriteFile(filepath.Join(exp.Root, "PROMPT.sha256"),
		[]byte(state.Global.PromptFileHash+"  PROMPT.txt\n"), 0o644); err != nil {
		return GlobalFreeze{}, err
	}
	return state.Global, state.write(exp)
}

// FreezeStage locks one stage's dataset. Requires the global config to be
// frozen first, and refuses if the stage is already frozen.
func FreezeStage(exp *Experiment, stage string) (StageFreeze, error) {
	state, err := LoadFreezeState(exp)
	if err != nil {
		return StageFreeze{}, err
	}
	if !state.Global.Frozen {
		return StageFreeze{}, fmt.Errorf("freeze the global config first (freeze --scope global)")
	}
	if existing, done := state.Stages[stage]; done {
		return existing, fmt.Errorf("stage %q already frozen at %s (never rewritten)", stage, existing.FrozenAt)
	}
	if !containsString(StageOrder, stage) {
		return StageFreeze{}, fmt.Errorf("unknown stage %q", stage)
	}

	datasetPath, err := exp.StageDataset(stage)
	if err != nil {
		return StageFreeze{}, err
	}
	cases, err := LoadCases(datasetPath)
	if err != nil {
		return StageFreeze{}, fmt.Errorf("stage %q: %w", stage, err)
	}
	problems := FreezeValidate(cases)
	if stage == StageEndToEnd {
		problems = append(problems, ValidateEndToEnd(cases)...)
		if _, green, auditErr := WriteP0Audit(exp); auditErr != nil {
			return StageFreeze{}, auditErr
		} else if !green {
			return StageFreeze{}, fmt.Errorf("P0 audit gate is not green — see datasets/P0_AUDIT.md (need 30 balanced questions, all rows PASS, human review complete)")
		}
	}
	if len(problems) > 0 {
		messages := make([]string, 0, len(problems))
		for _, problem := range problems {
			messages = append(messages, problem.Error())
		}
		return StageFreeze{}, fmt.Errorf("stage %q invalid:\n  %s", stage, strings.Join(messages, "\n  "))
	}

	hashes, err := stageDatasetHashes(exp, datasetPath)
	if err != nil {
		return StageFreeze{}, err
	}
	frozen := StageFreeze{
		DatasetHashes: hashes,
		Cases:         len(cases),
		FrozenAt:      time.Now().UTC().Format(time.RFC3339),
	}
	state.Stages[stage] = frozen
	return frozen, state.write(exp)
}

func stageDatasetHashes(exp *Experiment, datasetPath string) (map[string]string, error) {
	info, err := os.Stat(datasetPath)
	if err != nil {
		return nil, err
	}
	var files []string
	if info.IsDir() {
		files, _ = filepath.Glob(filepath.Join(datasetPath, "*.jsonl"))
	} else {
		files = []string{datasetPath}
	}
	sort.Strings(files)
	hashes := map[string]string{}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(exp.Root, file)
		hashes[filepath.ToSlash(rel)] = hashBytes(raw)
	}
	return hashes, nil
}

// requireStageFrozen is the run-time gate: the global config and this
// stage's dataset must be frozen, and the dataset on disk must still match
// the recorded hashes.
func requireStageFrozen(exp *Experiment, stage string) error {
	state, err := LoadFreezeState(exp)
	if err != nil {
		return err
	}
	if !state.Global.Frozen {
		return fmt.Errorf("global config not frozen; run `freeze --scope global` (or pass --allow-unfrozen for a smoke run)")
	}
	frozen, done := state.Stages[stage]
	if !done {
		return fmt.Errorf("stage %q not frozen; run `freeze --scope stage --stage %s`", stage, stage)
	}
	datasetPath, err := exp.StageDataset(stage)
	if err != nil {
		return err
	}
	current, err := stageDatasetHashes(exp, datasetPath)
	if err != nil {
		return err
	}
	for path, want := range frozen.DatasetHashes {
		if current[path] != want {
			return fmt.Errorf("stage %q dataset %s changed after freeze (%s != %s)", stage, path, current[path], want)
		}
	}
	if len(current) != len(frozen.DatasetHashes) {
		return fmt.Errorf("stage %q dataset file set changed after freeze", stage)
	}
	return nil
}
