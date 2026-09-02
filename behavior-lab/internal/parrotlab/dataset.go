package parrotlab

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Expected is the answer key for one case. `Value` is the single canonical
// correct answer; `Aliases` are other spellings of that same answer
// (P-1 fix #1 — never the answer universe, which is Case.Choices).
type Expected struct {
	Value     string   `json:"value,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
	Number    *float64 `json:"number,omitempty"`
	Tolerance float64  `json:"tolerance,omitempty"`
	Abstain   bool     `json:"abstain,omitempty"`
}

// Case is one evaluation item. The same struct serves every stage; `Stage`
// selects scoring and aggregation. Only `Instruction` (optionally prefixed
// by `BlackboardHint`) and `ImagePath` ever reach the model — never
// `Expected`, `TaskFamily`, `RequiredFacts` or `GroundTruthAddresses`.
type Case struct {
	CaseID       string   `json:"case_id"`
	Stage        string   `json:"stage"`
	Capabilities []string `json:"capabilities"`
	Operations   int      `json:"operations,omitempty"`
	BaseID       string   `json:"base_id,omitempty"`
	Sentinel     bool     `json:"sentinel,omitempty"`
	TaskFamily   string   `json:"task_family"`
	// Choices is the universe of valid answers for a "choice" case
	// (P-1 fix #1): an answer outside it is an unsupported assertion, not
	// merely wrong. Never conflated with the correct answer.
	Choices []string `json:"choices,omitempty"`
	// AddedPrimitive is the operation this instruction adds versus the
	// previous depth (instruction_cliff only, P-1 fix #4) — lets
	// aggregation separate a depth effect from a single-capability effect.
	AddedPrimitive string `json:"added_primitive,omitempty"`
	// PriorContract is true when an earlier step in this depth's pipeline
	// imposes an output form (e.g. JSON) that the scored final step does
	// not — a known contamination path aggregation must be able to isolate.
	PriorContract  bool     `json:"prior_contract,omitempty"`
	Instruction    string   `json:"instruction"`
	ImagePath      string   `json:"image_path,omitempty"`
	BlackboardHint string   `json:"blackboard_hint,omitempty"`
	HintCondition  string   `json:"hint_condition,omitempty"`
	Expected       Expected `json:"expected"`

	PageRefs             []int    `json:"page_refs,omitempty"`
	RequiredFacts        []string `json:"required_facts,omitempty"`
	GroundTruthAddresses []string `json:"ground_truth_addresses,omitempty"`

	// end_to_end only. Variant is "text" or "image": the text variant
	// prepends EvidenceText to the user turn; the image variant sends the
	// rendered page via ImagePath. EvidenceCID / SourceMethod are provenance
	// carried into the run record, never sent to the model.
	Variant      string `json:"variant,omitempty"`
	EvidenceText string `json:"evidence_text,omitempty"`
	EvidenceCID  string `json:"evidence_cid,omitempty"`
	SourceMethod string `json:"source_method,omitempty"`

	// sourceDir is the directory the case was loaded from, used to resolve
	// ImagePath. Not serialised.
	sourceDir string
}

var taskFamilies = map[string]bool{
	"choice": true, "exact": true, "entity": true, "numeric": true, "abstain": true,
}

var hintConditions = map[string]bool{
	"correct": true, "none": true, "incorrect": true, "random": true,
}

// LoadCases reads every *.jsonl under path (a file or a directory) into a
// flat slice, recording each case's source directory.
func LoadCases(path string) ([]Case, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var files []string
	if info.IsDir() {
		files, _ = filepath.Glob(filepath.Join(path, "*.jsonl"))
		if len(files) == 0 {
			return nil, fmt.Errorf("%s: no *.jsonl files", path)
		}
	} else {
		files = []string{path}
	}

	var cases []Case
	seen := map[string]string{}
	for _, file := range files {
		fileCases, err := readCaseFile(file)
		if err != nil {
			return nil, err
		}
		for _, item := range fileCases {
			if prev, dup := seen[item.CaseID]; dup {
				return nil, fmt.Errorf("duplicate case_id %q in %s and %s", item.CaseID, prev, file)
			}
			seen[item.CaseID] = file
			cases = append(cases, item)
		}
	}
	return cases, nil
}

func readCaseFile(file string) ([]Case, error) {
	handle, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	dir := filepath.Dir(file)
	var cases []Case
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "//") {
			continue
		}
		var item Case
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&item); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", file, line, err)
		}
		item.sourceDir = dir
		cases = append(cases, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return cases, nil
}

// ImageBytes reads the case image, or (nil, nil) when the case is text-only.
func (item Case) ImageBytes() ([]byte, error) {
	if item.ImagePath == "" {
		return nil, nil
	}
	return os.ReadFile(filepath.Join(item.sourceDir, filepath.FromSlash(item.ImagePath)))
}

// UserText is exactly what is sent as the model's user turn.
func (item Case) UserText() string {
	if item.BlackboardHint != "" && item.HintCondition != "none" {
		return "Context notes:\n" + item.BlackboardHint + "\n\nTask:\n" + item.Instruction
	}
	if item.Variant == "text" && item.EvidenceText != "" {
		return "Page content:\n" + item.EvidenceText + "\n\nQuestion:\n" + item.Instruction
	}
	return item.Instruction
}

// Validate checks structural and per-stage invariants (SCHEMA.md).
func Validate(cases []Case) []error {
	var problems []error
	add := func(format string, args ...any) { problems = append(problems, fmt.Errorf(format, args...)) }

	singlesByCap := map[string]bool{}
	for _, item := range cases {
		if item.Stage == StageSingles && len(item.Capabilities) == 1 {
			singlesByCap[item.Capabilities[0]] = true
		}
	}

	for _, item := range cases {
		id := item.CaseID
		if id == "" {
			add("a case has an empty case_id")
			continue
		}
		if !taskFamilies[item.TaskFamily] {
			add("%s: unknown task_family %q", id, item.TaskFamily)
		}
		if strings.TrimSpace(item.Instruction) == "" {
			add("%s: empty instruction", id)
		}
		for _, capability := range item.Capabilities {
			if !containsString(Capabilities, capability) {
				add("%s: unknown capability %q", id, capability)
			}
		}
		if item.TaskFamily == "numeric" && item.Expected.Number == nil && !item.Expected.Abstain {
			add("%s: numeric case without expected.number", id)
		}
		if item.TaskFamily != "numeric" && item.TaskFamily != "abstain" && item.Expected.Value == "" {
			add("%s: %s case without expected.value", id, item.TaskFamily)
		}
		if item.TaskFamily == "choice" {
			if len(item.Choices) < 2 {
				add("%s: choice case needs choices with >= 2 options", id)
			} else if !containsString(normaliseAll(item.Choices), normaliseAnswer(item.Expected.Value)) {
				add("%s: expected.value %q is not one of choices", id, item.Expected.Value)
			}
		}
		if len(item.Choices) > 0 && item.TaskFamily != "choice" {
			add("%s: choices given but task_family is %q, not \"choice\"", id, item.TaskFamily)
		}

		switch item.Stage {
		case StageEndToEnd:
			if len(item.PageRefs) == 0 {
				add("%s: end_to_end case without page_refs", id)
			}
			if item.Variant != "text" && item.Variant != "image" {
				add("%s: end_to_end case needs variant text|image, got %q", id, item.Variant)
			}
			if item.Variant == "text" && strings.TrimSpace(item.EvidenceText) == "" {
				add("%s: end_to_end text variant without evidence_text", id)
			}
			if item.Variant == "image" && item.ImagePath == "" {
				add("%s: end_to_end image variant without image_path", id)
			}
			if item.BaseID == "" {
				add("%s: end_to_end case without base_id (pairs text+image)", id)
			}
		case StageInstructionCliff:
			if item.BaseID == "" {
				add("%s: instruction_cliff case without base_id", id)
			}
			if item.Operations < 1 || item.Operations > 5 {
				add("%s: operations %d out of range 1..5", id, item.Operations)
			}
		case StageSingles:
			if len(item.Capabilities) != 1 {
				add("%s: singles case must name exactly one capability", id)
			}
		case StageInterference:
			if len(item.Capabilities) != 2 {
				add("%s: interference case must name exactly two capabilities", id)
			}
		case StageCoalitions:
			if item.BaseID == "" {
				add("%s: coalition case without base_id (K1/K2/K3)", id)
			}
		case StageBlackboard:
			if !hintConditions[item.HintCondition] {
				add("%s: blackboard case with invalid hint_condition %q", id, item.HintCondition)
			}
			if item.HintCondition != "none" && strings.TrimSpace(item.BlackboardHint) == "" {
				add("%s: blackboard case %q without blackboard_hint", id, item.HintCondition)
			}
		default:
			add("%s: unknown stage %q", id, item.Stage)
		}
	}

	// Instruction-cliff base groups must be complete OP1..OP5.
	cliffLevels := map[string]map[int]bool{}
	for _, item := range cases {
		if item.Stage != StageInstructionCliff {
			continue
		}
		if cliffLevels[item.BaseID] == nil {
			cliffLevels[item.BaseID] = map[int]bool{}
		}
		cliffLevels[item.BaseID][item.Operations] = true
	}
	for base, levels := range cliffLevels {
		for level := 1; level <= 5; level++ {
			if !levels[level] {
				add("instruction_cliff base %q is missing OP%d", base, level)
			}
		}
	}

	return problems
}

// FreezeValidate adds the freeze-only checks on top of Validate: no
// placeholder "example-" case ids may survive into a frozen experiment.
func FreezeValidate(cases []Case) []error {
	problems := Validate(cases)
	for _, item := range cases {
		if strings.HasPrefix(item.CaseID, "example-") {
			problems = append(problems, fmt.Errorf("%s: example case_id must be replaced before freeze", item.CaseID))
		}
	}
	return problems
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func normaliseAll(values []string) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = normaliseAnswer(value)
	}
	return out
}
