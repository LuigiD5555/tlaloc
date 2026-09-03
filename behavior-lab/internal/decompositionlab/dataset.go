package decompositionlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/exocortex"
)

// DatasetSchemaR0 identifies the T0 primary dataset file: the 30 frozen P0
// IMAGE variants (section 16), referenced by hash from P0, never copy-
// edited or regenerated here.
const DatasetSchemaR0 = "tlaloc.exocortex-t0.p0-image-dataset.r0"

// RequiredRecordCount is the primary paired denominator (section 16): T0-A
// and T0-B compare exactly the same 30 paired image questions.
const RequiredRecordCount = 30

// Category is the case taxonomy section 18 reports by.
const (
	CategoryLocate    = "locate"
	CategoryEntity    = "entity"
	CategoryFactual   = "factual"
	CategoryNumeric   = "numeric"
	CategorySynthesis = "synthesis"
)

var validCategories = map[string]bool{
	CategoryLocate: true, CategoryEntity: true, CategoryFactual: true,
	CategoryNumeric: true, CategorySynthesis: true,
}

// EvidenceRef is one evidence operand a bounded recipe may cite (BLOCKER 3:
// "one EvidenceAddress is not general enough"). Frozen P0 addresses are
// page-level, so BBox is usually nil here and the oracle Region Tlaloque
// derives geometry at execution time by locating TextSpan among the store's
// page blocks — TextSpan is a layout locator, never the semantic answer.
type EvidenceRef struct {
	ID       string             `json:"id"`
	DocID    string             `json:"doc_id"`
	Page     int                `json:"page"`
	Address  string             `json:"address,omitempty"`
	TextSpan string             `json:"text_span,omitempty"`
	BBox     *canonicaldoc.BBox `json:"bbox,omitempty"`
}

// AtomicStep is one element of a fixed, bounded recipe (BLOCKER 2:
// "Question != Atomic Operand"). The goal/question stays in Tlaloc; Parrot
// only ever receives one AtomicStep's opcode and minimum operand. There is
// no planner — recipes are compiled by the deterministic eligibility audit
// from a fixed rule table (E0.13/E0.14).
type AtomicStep struct {
	ID            string   `json:"id"`
	Opcode        string   `json:"opcode"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"` // ids into P0Record.EvidenceRefs
	OutputKey     string   `json:"output_key"`
	Deterministic bool     `json:"deterministic"` // true => a deterministic Tlaloque, not Parrot
	ChoiceWidth   int      `json:"choice_width,omitempty"`
}

// P0Record is one frozen P0 image-variant question, referenced by hash
// from the frozen P0 benchmark (section 16-17). ExpectedAnswer is used
// only for post-hoc scoring — the T0 runner must never place it on any
// path an executor (Parrot included) can read.
type P0Record struct {
	BaseID          string             `json:"base_id"`
	Goal            string             `json:"goal,omitempty"` // the broad P0 goal; stays in Tlaloc (BLOCKER 2)
	Question        string             `json:"question"`
	ExpectedAnswer  string             `json:"expected_answer"`
	Category        string             `json:"category"`
	TaskFamily      string             `json:"task_family,omitempty"`
	DocID           string             `json:"doc_id"`
	Page            int                `json:"page"`
	PageImagePath   string             `json:"page_image_path"`
	PageWidth       float64            `json:"page_width"`
	PageHeight      float64            `json:"page_height"`
	EvidenceAddress string             `json:"evidence_address"`
	EvidenceBBox    *canonicaldoc.BBox `json:"evidence_bbox,omitempty"`

	// EvidenceRefs / Recipe are the corrected representation (BLOCKER 2/3).
	// They are optional on load for backward compatibility with single-op
	// hand-authored fixtures; `prepare` always populates them.
	EvidenceRefs []EvidenceRef `json:"evidence_refs,omitempty"`
	Recipe       []AtomicStep  `json:"recipe,omitempty"`

	// Opcode is the single Micro-ISA opcode Parrot executes for this record
	// (P1: one op per invocation). OperandCharCount/OperandChoiceWidth are
	// curator-declared properties of what the region actually shows (not
	// derived from ExpectedAnswer), used only for the ModelAdapter's
	// contract check; zero means "no declared bound for this record".
	//
	// Opcode is a legacy single-op field kept for hand-authored fixtures.
	// The frozen prepared T0-B dataset leaves it empty and carries the model
	// opcode inside Recipe instead (see ModelStep); prefer ModelStep.
	Opcode             string `json:"opcode"`
	OperandCharCount   int    `json:"operand_char_count,omitempty"`
	OperandChoiceWidth int    `json:"operand_choice_width,omitempty"`

	// Choices carries the frozen P0 task's explicit choice labels for a
	// SELECT_ONE record (the locate/choice family). It is task structure
	// from the frozen P0 benchmark — never evidence text, never the answer —
	// and is presented to Parrot in full so a 2-way selection stays a real
	// selection after oracle localization.
	Choices []string `json:"choices,omitempty"`
}

// ModelStep returns the single non-deterministic (model-executed) AtomicStep
// of a record's bounded recipe — the one opcode that reaches Parrot (P1: one
// cognitive op per invocation). It is an explicit integrity error for an
// ELIGIBLE T0-B record to have zero or more than one model step. Records
// with no Recipe fall back to the legacy single Opcode field so hand-
// authored fixtures keep working.
func (r P0Record) ModelStep() (AtomicStep, error) {
	if len(r.Recipe) == 0 {
		if strings.TrimSpace(r.Opcode) == "" {
			return AtomicStep{}, fmt.Errorf("record %q: no recipe and no legacy opcode; cannot determine the model step", r.BaseID)
		}
		op, err := exocortex.NormalizeOpcode(r.Opcode)
		if err != nil {
			return AtomicStep{}, fmt.Errorf("record %q: %w", r.BaseID, err)
		}
		return AtomicStep{ID: "legacy", Opcode: op, OutputKey: "answer"}, nil
	}
	var found []AtomicStep
	for _, step := range r.Recipe {
		if !step.Deterministic {
			found = append(found, step)
		}
	}
	if len(found) == 0 {
		return AtomicStep{}, fmt.Errorf("record %q: recipe has zero model steps; an ELIGIBLE T0-B record must have exactly one", r.BaseID)
	}
	if len(found) > 1 {
		return AtomicStep{}, fmt.Errorf("record %q: recipe has %d model steps; an ELIGIBLE T0-B record must have exactly one (no planner)", r.BaseID, len(found))
	}
	op, err := exocortex.NormalizeOpcode(found[0].Opcode)
	if err != nil {
		return AtomicStep{}, fmt.Errorf("record %q step %q: %w", r.BaseID, found[0].ID, err)
	}
	found[0].Opcode = op
	return found[0], nil
}

// ValidateRecipe checks that a record's bounded recipe never asks a model
// step to carry more than one cognitive opcode and that every referenced
// evidence id resolves. It is applied by `prepare` and by the eligibility
// audit, never by a runtime planner.
func (r P0Record) ValidateRecipe(maxCognitiveOps int) error {
	refIDs := map[string]bool{}
	for _, ref := range r.EvidenceRefs {
		if strings.TrimSpace(ref.ID) == "" {
			return fmt.Errorf("record %q: evidence ref with empty id", r.BaseID)
		}
		if refIDs[ref.ID] {
			return fmt.Errorf("record %q: duplicate evidence ref id %q", r.BaseID, ref.ID)
		}
		refIDs[ref.ID] = true
	}
	stepIDs := map[string]bool{}
	for _, step := range r.Recipe {
		if strings.TrimSpace(step.ID) == "" || stepIDs[step.ID] {
			return fmt.Errorf("record %q: recipe step with missing or duplicate id", r.BaseID)
		}
		stepIDs[step.ID] = true
		if _, err := exocortex.NormalizeOpcode(step.Opcode); err != nil {
			return fmt.Errorf("record %q step %q: %w", r.BaseID, step.ID, err)
		}
		if strings.TrimSpace(step.OutputKey) == "" {
			return fmt.Errorf("record %q step %q: output_key is required", r.BaseID, step.ID)
		}
		for _, id := range step.EvidenceRefs {
			if !refIDs[id] {
				return fmt.Errorf("record %q step %q: unknown evidence ref %q", r.BaseID, step.ID, id)
			}
		}
		// One AtomicStep carries exactly one opcode, so a model step is
		// structurally one cognitive op. A recipe that asked a model step
		// for more than the profile limit could not be represented here;
		// this guard makes the invariant explicit for the audit.
		if !step.Deterministic && maxCognitiveOps < 1 {
			return fmt.Errorf("record %q step %q: profile permits no model cognitive ops", r.BaseID, step.ID)
		}
	}
	return nil
}

// Dataset is the loaded, hash-verified T0 primary dataset.
type Dataset struct {
	Schema               string     `json:"schema"`
	SourceBenchmark      string     `json:"source_benchmark"`
	SourceArtifactSHA256 string     `json:"source_artifact_sha256"`
	Records              []P0Record `json:"records"`
}

// LoadDataset reads, hash-verifies and validates the T0 primary dataset. It
// enforces exactly RequiredRecordCount records with unique base_ids, valid
// categories, and no accidental leakage: EvidenceAddress and BBox must
// exist without needing to touch ExpectedAnswer, and this function itself
// never inspects ExpectedAnswer beyond confirming it is non-empty (so a
// downstream scorer has something to score against).
func LoadDataset(path string) (Dataset, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Dataset{}, "", fmt.Errorf("read T0 dataset %s: %w", path, err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])

	var dataset Dataset
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&dataset); err != nil {
		return Dataset{}, "", fmt.Errorf("decode T0 dataset %s: %w", path, err)
	}
	if dataset.Schema != DatasetSchemaR0 {
		return Dataset{}, "", fmt.Errorf("T0 dataset %s: unexpected schema %q, want %q", path, dataset.Schema, DatasetSchemaR0)
	}
	if strings.TrimSpace(dataset.SourceBenchmark) == "" || strings.TrimSpace(dataset.SourceArtifactSHA256) == "" {
		return Dataset{}, "", fmt.Errorf("T0 dataset %s: source_benchmark and source_artifact_sha256 are required (P0 must be referenced by hash)", path)
	}
	if len(dataset.Records) != RequiredRecordCount {
		return Dataset{}, "", fmt.Errorf("T0 dataset %s: has %d records, want exactly %d (section 16 primary denominator)", path, len(dataset.Records), RequiredRecordCount)
	}
	seen := map[string]bool{}
	for i, r := range dataset.Records {
		if strings.TrimSpace(r.BaseID) == "" {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record[%d] missing base_id", path, i)
		}
		if seen[r.BaseID] {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: duplicate base_id %q", path, r.BaseID)
		}
		seen[r.BaseID] = true
		if strings.TrimSpace(r.Question) == "" || strings.TrimSpace(r.ExpectedAnswer) == "" {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q missing question or expected_answer", path, r.BaseID)
		}
		if !validCategories[r.Category] {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q has unknown category %q", path, r.BaseID, r.Category)
		}
		if strings.TrimSpace(r.DocID) == "" || r.Page <= 0 || strings.TrimSpace(r.PageImagePath) == "" {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q missing doc_id/page/page_image_path", path, r.BaseID)
		}
		if strings.TrimSpace(r.EvidenceAddress) == "" {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q missing evidence_address (required for the T0-B oracle condition)", path, r.BaseID)
		}
		// A record is valid with EITHER a single legacy Opcode (hand-authored
		// fixtures) OR a bounded Recipe (BLOCKER 2). A NOT_APPLICABLE_R0
		// record legitimately has neither.
		if strings.TrimSpace(r.Opcode) != "" {
			if _, err := exocortex.NormalizeOpcode(r.Opcode); err != nil {
				return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q: %w", path, r.BaseID, err)
			}
		}
		if err := r.ValidateRecipe(1); err != nil {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: %w", path, err)
		}
	}
	return dataset, hash, nil
}

// CategoryCounts reports how many records fall in each category, for the
// per-category breakdown section 18 asks for.
func (d Dataset) CategoryCounts() map[string]int {
	counts := map[string]int{}
	for _, r := range d.Records {
		counts[r.Category]++
	}
	return counts
}

// SortedBaseIDs returns every record's base_id in deterministic order, used
// to pair records across conditions when aggregating.
func (d Dataset) SortedBaseIDs() []string {
	ids := make([]string, 0, len(d.Records))
	for _, r := range d.Records {
		ids = append(ids, r.BaseID)
	}
	sort.Strings(ids)
	return ids
}
