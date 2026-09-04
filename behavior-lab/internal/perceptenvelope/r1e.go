package perceptenvelope

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
)

// R1-E VISUAL DEPENDENCE / SHORTCUT CONTROLS.
//
// R1-E asks a different causal question from R1-D: does an apparently
// successful visual answer actually depend on the correct visual operand?
// Every base is exercised under three interventions that share byte-
// identical model-facing text:
//
//	E0_NO_IMAGE      — same instruction, no visual operand at all
//	E1_WRONG_IMAGE   — same instruction, a plausible viewport from a
//	                   different eligible base whose visible value differs
//	E2_CORRECT_IMAGE — the correct visual operand (the frozen R1-D crop)
//
// The R1-D association instruction names no label text and no expected
// value, so the textual task is literally identical across E0/E1/E2 and
// across every base; the only thing that varies is the attached image.

// R1ESeed is the frozen R1-E pairing seed (shared project seed).
const R1ESeed = Seed

const (
	r1eDatasetSchema = "tlaloc.parrot-perceptual-envelope-r1.r1e-dataset.r1"
	r1eWrongSchema   = "tlaloc.parrot-perceptual-envelope-r1.r1e-wrong-image-map.r1"
)

// R1E conditions, in frozen execution order.
var R1EConditions = []string{"E0_NO_IMAGE", "E1_WRONG_IMAGE", "E2_CORRECT_IMAGE"}

// R1ECapabilitySpec is one capability exercised under the interventions.
type R1ECapabilitySpec struct {
	Capability  string `json:"capability"`
	Opcode      string `json:"opcode"`
	Instruction string `json:"instruction"`
	Cue         string `json:"cue"` // LABEL | VALUE
	Role        string `json:"role"`
}

// R1ECapabilities is the frozen capability set. READ_ASSOCIATED_NUMBER is
// the primary target (R1-D produced 22/22 and we must verify it is visually
// grounded). EXTRACT_NUMBER is a positive calibration control: it needs no
// association, only an atomic read of the cued number, so it should be
// strongly visually dependent — if it is not, the whole method is suspect.
var R1ECapabilities = []R1ECapabilitySpec{
	{"READ_ASSOCIATED_NUMBER", R1DAssocOpcode, R1DAssocInstruction, "LABEL", "PRIMARY"},
	{string(FrozenOpcode), string(FrozenOpcode), FrozenInstruction, "VALUE", "POSITIVE_CALIBRATION_CONTROL"},
}

// R1EWrongPair is one frozen base -> wrong-image mapping.
type R1EWrongPair struct {
	BaseID          string `json:"base_id"`
	WrongBaseID     string `json:"wrong_base_id"`
	BaseValue       string `json:"base_value"`
	WrongValue      string `json:"wrong_visible_value"`
	DigitLenMatched bool   `json:"digit_length_matched"`
	RankKey         string `json:"rank_key"`
}

// R1EWrongMap is the frozen deterministic wrong-image pairing.
type R1EWrongMap struct {
	Schema       string         `json:"schema"`
	ExperimentID string         `json:"experiment_id"`
	Seed         string         `json:"seed"`
	Rule         string         `json:"rule"`
	Pairs        []R1EWrongPair `json:"pairs"`
}

// R1EDataset is the frozen R1-E stimulus definition.
type R1EDataset struct {
	Schema                      string              `json:"schema"`
	ExperimentID                string              `json:"experiment_id"`
	Seed                        string              `json:"seed"`
	InterventionReuseOfR1DBases bool                `json:"INTERVENTION_REUSE_OF_R1D_BASES"`
	Note                        string              `json:"note"`
	Conditions                  []string            `json:"conditions"`
	Capabilities                []R1ECapabilitySpec `json:"capabilities"`
	Bases                       []R1DBase           `json:"bases"`
	WrongMap                    R1EWrongMap         `json:"wrong_image_map"`
}

func rankKeyR1E(baseID, wrongID string) string {
	sum := sha256.Sum256([]byte(R1ESeed + "|" + baseID + "|" + wrongID))
	return hex.EncodeToString(sum[:])
}

// EligibleR1DBases returns the eligible bases in frozen order.
func EligibleR1DBases(alloc R1DAllocation) []R1DBase {
	var out []R1DBase
	for _, b := range alloc.Bases {
		if b.Eligible {
			out = append(out, b)
		}
	}
	return out
}

// BuildR1EWrongMap applies the frozen deterministic pairing rule (protocol
// §5): rank the other eligible bases by sha256(seed || base_id ||
// wrong_id) ascending and take the first candidate whose visible value
// differs from the base value and whose line text does not accidentally
// contain the base value; prefer an equal digit-length candidate. No model
// output is read.
func BuildR1EWrongMap(elig []R1DBase) (R1EWrongMap, error) {
	wm := R1EWrongMap{
		Schema: r1eWrongSchema, ExperimentID: ExperimentID, Seed: R1ESeed,
		Rule: "rank other eligible bases by sha256(seed||base_id||wrong_id) asc; " +
			"first with wrong.value != base.value AND base.value not a substring of wrong.line_text; " +
			"prefer equal digit-length (recorded), else first valid",
	}
	for _, base := range elig {
		type cand struct {
			id  string
			key string
			b   R1DBase
		}
		var cands []cand
		for _, w := range elig {
			if w.BaseID == base.BaseID || w.Value == base.Value {
				continue
			}
			if strings.Contains(w.LineText, base.Value) {
				continue
			}
			cands = append(cands, cand{w.BaseID, rankKeyR1E(base.BaseID, w.BaseID), w})
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].key < cands[j].key })
		if len(cands) == 0 {
			return wm, fmt.Errorf("%s: no compatible wrong-image candidate", base.BaseID)
		}
		pick, matched := 0, false
		for i, c := range cands {
			if len(c.b.Value) == len(base.Value) {
				pick, matched = i, true
				break
			}
		}
		wm.Pairs = append(wm.Pairs, R1EWrongPair{
			BaseID: base.BaseID, WrongBaseID: cands[pick].id,
			BaseValue: base.Value, WrongValue: cands[pick].b.Value,
			DigitLenMatched: matched, RankKey: cands[pick].key,
		})
	}
	return wm, nil
}

// R1EWrongMapFP is a deterministic fingerprint of a wrong-image map (used
// by the doctor to prove determinism / poison-invariance).
func R1EWrongMapFP(wm R1EWrongMap) string {
	var sb strings.Builder
	for _, p := range wm.Pairs {
		fmt.Fprintf(&sb, "%s->%s(%s|%s);", p.BaseID, p.WrongBaseID, p.BaseValue, p.WrongValue)
	}
	return sb.String()
}

// BuildR1EDataset assembles the frozen dataset from the eligible R1-D bases.
func BuildR1EDataset(elig []R1DBase) (R1EDataset, error) {
	wm, err := BuildR1EWrongMap(elig)
	if err != nil {
		return R1EDataset{}, err
	}
	return R1EDataset{
		Schema: r1eDatasetSchema, ExperimentID: ExperimentID, Seed: R1ESeed,
		InterventionReuseOfR1DBases: true,
		Note: "R1-E reuses the frozen R1-D eligible bases as INTERVENTION bases. " +
			"R1-E asks a different causal question (does changing/removing the image change the answer?); " +
			"the per-condition accuracies are NOT a fresh independent accuracy estimate for the capability.",
		Conditions: R1EConditions, Capabilities: R1ECapabilities,
		Bases: elig, WrongMap: wm,
	}, nil
}

// R1ERecord is one intervention result.
type R1ERecord struct {
	Capability           string `json:"capability"`
	BaseID               string `json:"base_id"`
	Condition            string `json:"condition"`
	Opcode               string `json:"opcode"`
	Instruction          string `json:"instruction"`
	Page                 int    `json:"page"`
	Label                string `json:"label"`
	TaskGold             string `json:"task_gold"` // Y
	HasImage             bool   `json:"has_image"`
	WrongBaseID          string `json:"wrong_base_id,omitempty"`
	WrongVisibleValue    string `json:"wrong_visible_value,omitempty"` // Y2
	RawText              string `json:"raw_text"`
	GotValue             string `json:"got_value"`
	TaskGoldCorrect      bool   `json:"task_gold_correct"`
	ImageConsistent      bool   `json:"image_consistent"` // E1 only
	ContractSuccess      bool   `json:"contract_success"`
	Abstained            bool   `json:"abstained"`
	UnsupportedAssertion bool   `json:"unsupported_assertion"`
	FormatFailure        bool   `json:"format_failure"`
	FailureClass         string `json:"failure_class"`
	LatencyMS            int64  `json:"latency_ms"`
	PromptTokens         int    `json:"prompt_tokens"`
	CompletionTok        int    `json:"completion_tokens"`
	CropPath             string `json:"crop_path,omitempty"`
	Error                string `json:"error,omitempty"`
}

func writeRawR1E(dir string, rec R1ERecord) {
	body, _ := json.MarshalIndent(rec, "", "  ")
	name := strings.ToLower(rec.Capability + "_" + rec.BaseID + "_" + rec.Condition)
	_ = os.WriteFile(filepath.Join(dir, name+".json"), body, 0o644)
}

// r1eCuedViewport renders one base's frozen R1-D viewport with the cue on
// the requested token. This reuses the frozen R1-D renderer unchanged, so
// the E2 correct-image crop is byte-identical to the R1-D D0L (LABEL) or
// D0V (VALUE) crop for that base.
func r1eCuedViewport(prov parrotlab.PageProvider, base R1DBase, cue string) (*image.RGBA, R1DGeometry, error) {
	geo, err := DeriveR1DGeometry(base)
	if err != nil {
		return nil, geo, err
	}
	pagePNG, err := prov.RenderPNG(base.Page)
	if err != nil {
		return nil, geo, fmt.Errorf("render page %d: %w", base.Page, err)
	}
	vp, err := BuildR1DViewport(pagePNG, base, geo)
	if err != nil {
		return nil, geo, err
	}
	img := cloneRGBA(vp)
	if cue == "VALUE" {
		drawR1DCue(img, geo.ValueBBoxCanvas)
	} else {
		drawR1DCue(img, geo.LabelBBoxCanvas)
	}
	return img, geo, nil
}

func r1eImageConsistent(raw, wrongValue, taskGold string) bool {
	got, ok := parseFamilyValue(FamMultiDigit, raw)
	if !ok || got == "" {
		return false
	}
	w, _ := parseFamilyValue(FamMultiDigit, wrongValue)
	g, _ := parseFamilyValue(FamMultiDigit, taskGold)
	return got == w && w != g
}

// RunR1E executes every frozen triplet once (temperature 0). Deterministic
// order: capability, then base, then E0/E1/E2.
func RunR1E(ctx context.Context, cfg RunConfig, ds R1EDataset) ([]R1ERecord, error) {
	prov, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, err
	}
	client := newR1DClient(cfg)
	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	baseOf := map[string]R1DBase{}
	for _, b := range ds.Bases {
		baseOf[b.BaseID] = b
	}
	wrongOf := map[string]R1EWrongPair{}
	for _, p := range ds.WrongMap.Pairs {
		wrongOf[p.BaseID] = p
	}

	var out []R1ERecord
	for _, capSpec := range ds.Capabilities {
		for _, base := range ds.Bases {
			wp, ok := wrongOf[base.BaseID]
			if !ok {
				return nil, fmt.Errorf("%s: no wrong-image pair", base.BaseID)
			}
			wbase := baseOf[wp.WrongBaseID]

			correctImg, _, cerr := r1eCuedViewport(prov, base, capSpec.Cue)
			if cerr != nil {
				return nil, cerr
			}
			wrongImg, _, werr := r1eCuedViewport(prov, wbase, capSpec.Cue)
			if werr != nil {
				return nil, werr
			}

			for _, cond := range ds.Conditions {
				select {
				case <-ctx.Done():
					return out, ctx.Err()
				default:
				}
				rec := R1ERecord{
					Capability: capSpec.Capability, BaseID: base.BaseID, Condition: cond,
					Opcode: capSpec.Opcode, Instruction: capSpec.Instruction,
					Page: base.Page, Label: base.Label, TaskGold: base.Value,
				}
				var raw string
				var promptTok, compTok int
				var latency int64
				var visible []string

				switch cond {
				case "E0_NO_IMAGE":
					rec.HasImage = false
					start := time.Now()
					tr, terr := client.CompleteText(ctx, "", capSpec.Instruction)
					latency = time.Since(start).Milliseconds()
					if terr != nil {
						rec.Error = terr.Error()
						break
					}
					raw, promptTok, compTok = tr.Content, tr.PromptTokensReported, tr.CompletionTokensReported
				case "E1_WRONG_IMAGE":
					rec.HasImage = true
					rec.WrongBaseID = wbase.BaseID
					rec.WrongVisibleValue = wbase.Value
					visible = []string{wbase.Value}
					cropPath := filepath.Join(cropDir, strings.ToLower(capSpec.Capability+"_"+base.BaseID+"_e1_wrong.png"))
					if err := writeRGBAPNG(cropPath, wrongImg); err != nil {
						return nil, err
					}
					rec.CropPath = cropPath
					img, ierr := os.ReadFile(cropPath)
					if ierr != nil {
						rec.Error = ierr.Error()
						break
					}
					start := time.Now()
					res, rerr := client.CompletePerception(ctx, target.PerceptionInput{Question: capSpec.Instruction, Image: img, MediaType: "image/png"})
					latency = time.Since(start).Milliseconds()
					if rerr != nil {
						rec.Error = rerr.Error()
						break
					}
					raw, promptTok, compTok = res.Content, res.PromptTokensReported, res.CompletionTokensReported
				case "E2_CORRECT_IMAGE":
					rec.HasImage = true
					visible = []string{base.Value}
					cropPath := filepath.Join(cropDir, strings.ToLower(capSpec.Capability+"_"+base.BaseID+"_e2_correct.png"))
					if err := writeRGBAPNG(cropPath, correctImg); err != nil {
						return nil, err
					}
					rec.CropPath = cropPath
					img, ierr := os.ReadFile(cropPath)
					if ierr != nil {
						rec.Error = ierr.Error()
						break
					}
					start := time.Now()
					res, rerr := client.CompletePerception(ctx, target.PerceptionInput{Question: capSpec.Instruction, Image: img, MediaType: "image/png"})
					latency = time.Since(start).Milliseconds()
					if rerr != nil {
						rec.Error = rerr.Error()
						break
					}
					raw, promptTok, compTok = res.Content, res.PromptTokensReported, res.CompletionTokensReported
				}

				rec.LatencyMS = latency
				if rec.Error != "" {
					out = append(out, rec)
					writeRawR1E(rawDir, rec)
					continue
				}
				rec.RawText = raw
				rec.PromptTokens = promptTok
				rec.CompletionTok = compTok
				sc := ScoreR1DAssoc(raw, base.Value, visible, nil)
				rec.GotValue = sc.GotValue
				rec.TaskGoldCorrect = sc.ValueCorrect
				rec.ContractSuccess = sc.ContractSuccess
				rec.Abstained = sc.Abstained
				rec.UnsupportedAssertion = sc.UnsupportedAssertion
				rec.FormatFailure = sc.FormatFailure
				rec.FailureClass = sc.FailureClass
				if cond == "E1_WRONG_IMAGE" {
					rec.ImageConsistent = r1eImageConsistent(raw, wbase.Value, base.Value)
				}
				out = append(out, rec)
				writeRawR1E(rawDir, rec)
			}
		}
	}
	return out, nil
}
