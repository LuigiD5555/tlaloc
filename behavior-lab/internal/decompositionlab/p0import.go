package decompositionlab

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BLOCKER 1 (T0 protocol section, "C0 IS NOT A NEW EXOCORTEX EXECUTION"):
// C0 is the historical frozen P0 direct-Parrot baseline. It must be
// IMPORTED from the frozen P0 experiment, never re-run through the
// Exocortex ModelAdapter (which correctly rejects a FULL_PAGE operand for
// a tight-crop profile — valid runtime behaviour, invalid as a P0
// baseline). This file is the deterministic import path. It makes zero new
// model calls.

// P0Outcome is one frozen P0 IMAGE-variant per-base result, imported as-is.
type P0Outcome struct {
	BaseID               string `json:"base_id"`
	CaseID               string `json:"case_id"`
	Category             string `json:"category"`
	TaskFamily           string `json:"task_family"`
	Attempted            bool   `json:"attempted"`
	OriginalOutput       string `json:"original_output"`
	ContractSuccess      bool   `json:"contract_success"`
	SemanticCorrect      bool   `json:"semantic_correct"`
	Abstained            bool   `json:"abstained"`
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	FormatFailure        bool   `json:"format_failure"`
	LatencyMS            int64  `json:"latency_ms"`
	ExpectedAnswer       string `json:"expected_answer"`
}

// P0BaselineProvenance records exactly which frozen artifacts C0 was
// imported from, with their hashes (T0 protocol: "for every paired P0
// record preserve ... source provenance, artifact hash").
type P0BaselineProvenance struct {
	ExperimentDir     string `json:"experiment_dir"`
	ExperimentID      string `json:"experiment_id"`
	DatasetPath       string `json:"dataset_path"`
	DatasetSHA256     string `json:"dataset_sha256"`
	DatasetFreezeHash string `json:"dataset_freeze_hash"`
	RunsPath          string `json:"runs_path"`
	RunsSHA256        string `json:"runs_sha256"`
	ProvenancePath    string `json:"provenance_path"`
	ProvenanceSHA256  string `json:"provenance_sha256"`
	ImageRecords      int    `json:"image_records"`
}

// P0ProvenanceRecord is one frozen P0 per-base provenance entry (exported
// so `prepare` can build corrected P0Records from it).
type P0ProvenanceRecord struct {
	BaseID         string   `json:"base_id"`
	Category       string   `json:"category"`
	TaskFamily     string   `json:"task_family"`
	Question       string   `json:"question"`
	ExpectedAnswer string   `json:"expected_answer"`
	PageRefs       []int    `json:"page_refs"`
	GTAddresses    []string `json:"ground_truth_addresses"`
	RequiredFacts  []string `json:"required_facts"`
}

type p0ProvenanceRecord = P0ProvenanceRecord

// LoadP0Provenance reads the frozen P0 provenance file (an array of records).
func LoadP0Provenance(experimentDir string) ([]P0ProvenanceRecord, string, error) {
	path := filepath.Join(experimentDir, "datasets", "end-to-end.provenance.json")
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(body)
	var records []P0ProvenanceRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, "", fmt.Errorf("decode P0 provenance: %w", err)
	}
	return records, hex.EncodeToString(sum[:]), nil
}

type p0RunRecord struct {
	CaseID     string `json:"case_id"`
	BaseID     string `json:"base_id"`
	Variant    string `json:"variant"`
	TaskFamily string `json:"task_family"`
	Actual     struct {
		Raw string `json:"raw"`
	} `json:"actual"`
	Score struct {
		SemanticCorrect      bool `json:"semantic_correct"`
		FormatValid          bool `json:"format_valid"`
		ContractSuccess      bool `json:"contract_success"`
		Abstained            bool `json:"abstained"`
		UnsupportedAssertion bool `json:"unsupported_assertion"`
	} `json:"score"`
	Resources struct {
		WallMS int64 `json:"wall_ms"`
	} `json:"resources"`
}

func sha256Bytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func sha256File(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// ImportP0Baseline reads the frozen P0 experiment directory and returns the
// IMAGE-variant per-base outcomes plus their provenance. It verifies the
// dataset hash against the experiment's own FREEZE.json ledger. It never
// touches ExpectedAnswer beyond carrying it for post-hoc reference.
func ImportP0Baseline(experimentDir string) (map[string]P0Outcome, P0BaselineProvenance, error) {
	datasetPath := filepath.Join(experimentDir, "datasets", "end-to-end.jsonl")
	runsPath := filepath.Join(experimentDir, "runs", "end_to_end", "end_to_end.jsonl")
	provPath := filepath.Join(experimentDir, "datasets", "end-to-end.provenance.json")
	freezePath := filepath.Join(experimentDir, "FREEZE.json")

	datasetHash, err := sha256File(datasetPath)
	if err != nil {
		return nil, P0BaselineProvenance{}, fmt.Errorf("hash P0 dataset: %w", err)
	}
	runsHash, err := sha256File(runsPath)
	if err != nil {
		return nil, P0BaselineProvenance{}, fmt.Errorf("hash P0 runs: %w", err)
	}
	provHash, err := sha256File(provPath)
	if err != nil {
		return nil, P0BaselineProvenance{}, fmt.Errorf("hash P0 provenance: %w", err)
	}

	freezeBody, err := os.ReadFile(freezePath)
	if err != nil {
		return nil, P0BaselineProvenance{}, fmt.Errorf("read P0 freeze ledger: %w", err)
	}
	var freeze struct {
		Stages map[string]struct {
			DatasetHashes map[string]string `json:"dataset_hashes"`
			Cases         int               `json:"cases"`
		} `json:"stages"`
	}
	if err := json.Unmarshal(freezeBody, &freeze); err != nil {
		return nil, P0BaselineProvenance{}, fmt.Errorf("decode P0 freeze ledger: %w", err)
	}
	stage, ok := freeze.Stages["end_to_end"]
	if !ok {
		return nil, P0BaselineProvenance{}, fmt.Errorf("P0 freeze ledger has no end_to_end stage")
	}
	freezeHash := stage.DatasetHashes["datasets/end-to-end.jsonl"]
	if freezeHash == "" {
		return nil, P0BaselineProvenance{}, fmt.Errorf("P0 freeze ledger records no hash for datasets/end-to-end.jsonl")
	}
	if freezeHash != datasetHash {
		return nil, P0BaselineProvenance{}, fmt.Errorf("P0 dataset hash mismatch: on disk %s, frozen %s", datasetHash, freezeHash)
	}

	provBody, err := os.ReadFile(provPath)
	if err != nil {
		return nil, P0BaselineProvenance{}, err
	}
	var provRecords []p0ProvenanceRecord
	if err := json.Unmarshal(provBody, &provRecords); err != nil {
		return nil, P0BaselineProvenance{}, fmt.Errorf("decode P0 provenance: %w", err)
	}
	provByBase := map[string]p0ProvenanceRecord{}
	for _, r := range provRecords {
		provByBase[r.BaseID] = r
	}

	runsFile, err := os.Open(runsPath)
	if err != nil {
		return nil, P0BaselineProvenance{}, err
	}
	defer runsFile.Close()

	outcomes := map[string]P0Outcome{}
	scanner := bufio.NewScanner(runsFile)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rr p0RunRecord
		if err := json.Unmarshal([]byte(line), &rr); err != nil {
			return nil, P0BaselineProvenance{}, fmt.Errorf("decode P0 run record: %w", err)
		}
		if rr.Variant != "image" {
			continue
		}
		prov := provByBase[rr.BaseID]
		outcomes[rr.BaseID] = P0Outcome{
			BaseID:               rr.BaseID,
			CaseID:               rr.CaseID,
			Category:             prov.Category,
			TaskFamily:           firstNonEmpty(rr.TaskFamily, prov.TaskFamily),
			Attempted:            true,
			OriginalOutput:       rr.Actual.Raw,
			ContractSuccess:      rr.Score.ContractSuccess,
			SemanticCorrect:      rr.Score.SemanticCorrect,
			Abstained:            rr.Score.Abstained,
			UnsupportedAssertion: rr.Score.UnsupportedAssertion,
			FormatFailure:        !rr.Score.FormatValid,
			LatencyMS:            rr.Resources.WallMS,
			ExpectedAnswer:       prov.ExpectedAnswer,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, P0BaselineProvenance{}, err
	}
	if len(outcomes) == 0 {
		return nil, P0BaselineProvenance{}, fmt.Errorf("no IMAGE-variant P0 run records found in %s", runsPath)
	}

	prov := P0BaselineProvenance{
		ExperimentDir:     experimentDir,
		ExperimentID:      "parrot-capability-r0",
		DatasetPath:       datasetPath,
		DatasetSHA256:     datasetHash,
		DatasetFreezeHash: freezeHash,
		RunsPath:          runsPath,
		RunsSHA256:        runsHash,
		ProvenancePath:    provPath,
		ProvenanceSHA256:  provHash,
		ImageRecords:      len(outcomes),
	}
	return outcomes, prov, nil
}

// LoadC0Baseline reads a frozen results/C0_P0_BASELINE.json (written by
// `prepare`) back into the base_id-keyed map the runner consumes for C0
// rows. C0 makes zero new model calls; this is the only source for it.
func LoadC0Baseline(path string) (map[string]P0Outcome, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Schema   string      `json:"schema"`
		Outcomes []P0Outcome `json:"outcomes"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode C0 baseline %s: %w", path, err)
	}
	if len(doc.Outcomes) == 0 {
		return nil, fmt.Errorf("C0 baseline %s has no outcomes", path)
	}
	out := make(map[string]P0Outcome, len(doc.Outcomes))
	for _, o := range doc.Outcomes {
		out[o.BaseID] = o
	}
	return out, nil
}

// SortedP0BaseIDs returns the imported base ids in deterministic order.
func SortedP0BaseIDs(outcomes map[string]P0Outcome) []string {
	ids := make([]string, 0, len(outcomes))
	for id := range outcomes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
