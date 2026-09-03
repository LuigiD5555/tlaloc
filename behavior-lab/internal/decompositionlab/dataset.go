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

// P0Record is one frozen P0 image-variant question, referenced by hash
// from the frozen P0 benchmark (section 16-17). ExpectedAnswer is used
// only for post-hoc scoring — the T0 runner must never place it on any
// path an executor (Parrot included) can read.
type P0Record struct {
	BaseID          string             `json:"base_id"`
	Question        string             `json:"question"`
	ExpectedAnswer  string             `json:"expected_answer"`
	Category        string             `json:"category"`
	DocID           string             `json:"doc_id"`
	Page            int                `json:"page"`
	PageImagePath   string             `json:"page_image_path"`
	PageWidth       float64            `json:"page_width"`
	PageHeight      float64            `json:"page_height"`
	EvidenceAddress string             `json:"evidence_address"`
	EvidenceBBox    *canonicaldoc.BBox `json:"evidence_bbox,omitempty"`

	// Opcode is the single Micro-ISA opcode Parrot executes for this record
	// (P1: one op per invocation). OperandCharCount/OperandChoiceWidth are
	// curator-declared properties of what the region actually shows (not
	// derived from ExpectedAnswer), used only for the ModelAdapter's
	// contract check; zero means "no declared bound for this record".
	Opcode             string `json:"opcode"`
	OperandCharCount   int    `json:"operand_char_count,omitempty"`
	OperandChoiceWidth int    `json:"operand_choice_width,omitempty"`
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
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q missing evidence_address (required for the T0-A oracle condition)", path, r.BaseID)
		}
		if _, err := exocortex.NormalizeOpcode(r.Opcode); err != nil {
			return Dataset{}, "", fmt.Errorf("T0 dataset %s: record %q: %w", path, r.BaseID, err)
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
