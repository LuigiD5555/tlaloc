package parrotlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"math/rand"
	"os"
	"path/filepath"
)

// T0-A CONTROLLED EXTERNAL COMPOSITION dataset generator (T0 protocol
// sections 8-14). It produces a NEW, frozen dataset — it never modifies P1.
// Every base stimulus supports the SAME underlying two-stage task across
// all four T0-A conditions (D0/D1/D2/D3):
//
//	task: two labeled numeric operands A and B are shown; which label
//	carries the larger value? (values are always distinct, so the answer
//	is exactly "A" or "B")
//
//	OP1: EXTRACT_NUMBER for operand A     (measured-usable primitive)
//	OP2: EXTRACT_NUMBER for operand B     (measured-usable primitive)
//	join: deterministic COMPARE_NUMBERS   (always external)
//
// This deliberately avoids the P2-A collapsed capabilities
// (VISUAL_LOCATE / FOLLOW_ONE_REFERENCE): T0-A tests composition, not
// visual localization.

const (
	T0ADatasetSchemaR0 = "tlaloc.exocortex-t0a.controlled-composition.r0"
	t0aCanvasW         = 520
	t0aCanvasH         = 320
	t0aCropW           = 460
	t0aCropH           = 120
)

// T0ARecord is one frozen T0-A base stimulus.
type T0ARecord struct {
	ID       string `json:"id"`
	Seed     int64  `json:"seed"`
	ValueA   int    `json:"value_a"`
	ValueB   int    `json:"value_b"`
	Larger   string `json:"larger"` // "A" | "B" — the expected answer
	FullPath string `json:"full_image_path"`
	CropA    string `json:"crop_a_path"`
	CropB    string `json:"crop_b_path"`

	// OracleOperandA is the T0-A generator's own scene truth for operand A.
	// It is consumed ONLY by the D2_ORACLE_EXTERNAL_OP1 / D3 conditions as
	// an explicit upper-bound intervention — it is NOT an OCR/extraction of
	// the rendered pixels and must never be read as evidence that a real
	// deterministic Tlaloque can obtain the operand. It is never exposed to
	// any Parrot call. (JSON tag kept as-is to preserve the frozen dataset
	// sha256; the name says what it really is.)
	OracleOperandA string `json:"deterministic_operand_a"`
}

// T0ADataset is the frozen dataset document.
type T0ADataset struct {
	Schema     string      `json:"schema"`
	Seed       int64       `json:"seed"`
	Count      int         `json:"count"`
	TaskFamily string      `json:"task_family"`
	Records    []T0ARecord `json:"records"`
}

// GenerateT0A writes the T0-A dataset (JSON + rendered images) under
// outDir and returns the dataset plus the sha256 of the written JSON file.
func GenerateT0A(seed int64, count int, outDir string) (T0ADataset, string, error) {
	if count <= 0 {
		count = 40
	}
	imageDir := filepath.Join(outDir, "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return T0ADataset{}, "", err
	}
	source := rand.New(rand.NewSource(seed))
	dataset := T0ADataset{Schema: T0ADatasetSchemaR0, Seed: seed, Count: count, TaskFamily: "num_compare"}

	for index := 0; index < count; index++ {
		valueA := 100 + source.Intn(900)
		valueB := 100 + source.Intn(900)
		for valueB == valueA {
			valueB = 100 + source.Intn(900)
		}
		larger := "A"
		if valueB > valueA {
			larger = "B"
		}
		id := fmt.Sprintf("t0a-%03d", index+1)

		full := newCanvas(t0aCanvasW, t0aCanvasH)
		drawText(full, fmt.Sprintf("A = %d", valueA), image.Point{X: 40, Y: 70}, 5, miInk)
		drawText(full, fmt.Sprintf("B = %d", valueB), image.Point{X: 40, Y: 200}, 5, miInk)

		cropA := newCanvas(t0aCropW, t0aCropH)
		drawText(cropA, fmt.Sprintf("A = %d", valueA), image.Point{X: 30, Y: 40}, 5, miInk)
		cropB := newCanvas(t0aCropW, t0aCropH)
		drawText(cropB, fmt.Sprintf("B = %d", valueB), image.Point{X: 30, Y: 40}, 5, miInk)

		fullRel := filepath.ToSlash(filepath.Join("images", id+"-full.png"))
		cropARel := filepath.ToSlash(filepath.Join("images", id+"-a.png"))
		cropBRel := filepath.ToSlash(filepath.Join("images", id+"-b.png"))
		if err := os.WriteFile(filepath.Join(outDir, filepath.FromSlash(fullRel)), encodePNG(full), 0o644); err != nil {
			return T0ADataset{}, "", err
		}
		if err := os.WriteFile(filepath.Join(outDir, filepath.FromSlash(cropARel)), encodePNG(cropA), 0o644); err != nil {
			return T0ADataset{}, "", err
		}
		if err := os.WriteFile(filepath.Join(outDir, filepath.FromSlash(cropBRel)), encodePNG(cropB), 0o644); err != nil {
			return T0ADataset{}, "", err
		}

		dataset.Records = append(dataset.Records, T0ARecord{
			ID: id, Seed: seed + int64(index), ValueA: valueA, ValueB: valueB, Larger: larger,
			FullPath: fullRel, CropA: cropARel, CropB: cropBRel,
			OracleOperandA: fmt.Sprintf("%d", valueA),
		})
	}

	body, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return T0ADataset{}, "", err
	}
	body = append(body, '\n')
	datasetPath := filepath.Join(outDir, "t0a_dataset.json")
	if err := os.WriteFile(datasetPath, body, 0o644); err != nil {
		return T0ADataset{}, "", err
	}
	sum := sha256.Sum256(body)
	return dataset, hex.EncodeToString(sum[:]), nil
}

// LoadT0ADataset reads and hash-verifies a frozen T0-A dataset file.
func LoadT0ADataset(path string) (T0ADataset, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return T0ADataset{}, "", err
	}
	sum := sha256.Sum256(body)
	var dataset T0ADataset
	if err := json.Unmarshal(body, &dataset); err != nil {
		return T0ADataset{}, "", fmt.Errorf("decode T0-A dataset %s: %w", path, err)
	}
	if dataset.Schema != T0ADatasetSchemaR0 {
		return T0ADataset{}, "", fmt.Errorf("T0-A dataset %s: unexpected schema %q", path, dataset.Schema)
	}
	if len(dataset.Records) != dataset.Count || dataset.Count == 0 {
		return T0ADataset{}, "", fmt.Errorf("T0-A dataset %s: count %d != %d records", path, dataset.Count, len(dataset.Records))
	}
	seen := map[string]bool{}
	for _, r := range dataset.Records {
		if r.ID == "" || seen[r.ID] {
			return T0ADataset{}, "", fmt.Errorf("T0-A dataset %s: missing or duplicate record id", path)
		}
		seen[r.ID] = true
		if r.Larger != "A" && r.Larger != "B" {
			return T0ADataset{}, "", fmt.Errorf("T0-A dataset %s: record %q has invalid larger %q", path, r.ID, r.Larger)
		}
		if (r.ValueA > r.ValueB) != (r.Larger == "A") {
			return T0ADataset{}, "", fmt.Errorf("T0-A dataset %s: record %q larger label disagrees with values", path, r.ID)
		}
	}
	return dataset, hex.EncodeToString(sum[:]), nil
}
