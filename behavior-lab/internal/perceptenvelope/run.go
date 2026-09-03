package perceptenvelope

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/decompositionlab"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
)

// FrozenInstruction is the single atomic EXTRACT_NUMBER instruction used
// for every R1-A and R1-B condition (protocol addendum R1_PROTOCOL_ADDENDUM_00).
// No workflow narration; one cognitive operation.
const FrozenInstruction = "Read the number inside the marked rectangle. Reply with only the number."

// FrozenOpcode is the model-facing cognitive opcode.
const FrozenOpcode = exocortex.OpExtractNumber

var (
	numberLike   = regexp.MustCompile(`^[+-]?[0-9][0-9.,]*%?$`)
	abstainWords = regexp.MustCompile(`(?i)\b(unknown|n/?a|none|cannot|can't|unable|not\s+visible|no\s+number|not\s+sure)\b`)
	digitsOnly   = regexp.MustCompile(`[^0-9]`)
)

// RecordOutcome is one (base, context level) result.
type RecordOutcome struct {
	BaseID      string `json:"base_id"`
	CandidateID string `json:"candidate_id"`
	Stage       string `json:"stage"`
	Level       string `json:"context_level"`
	Page        int    `json:"page"`
	Gold        string `json:"gold"`
	RawText     string `json:"raw_text"`

	SemanticCorrect      bool `json:"semantic_correct"`
	ContractSuccess      bool `json:"contract_success"`
	Abstained            bool `json:"abstained"`
	UnsupportedAssertion bool `json:"unsupported_assertion"`
	FormatFailure        bool `json:"format_failure"`

	FailureClass   string  `json:"failure_class"`
	EditDistance   int     `json:"edit_distance"`
	VisualExposure float64 `json:"visual_exposure_ratio"`
	PixelArea      int     `json:"pixel_area"`
	CropWidth      int     `json:"crop_width"`
	CropHeight     int     `json:"crop_height"`
	LatencyMS      int64   `json:"latency_ms"`
	PromptTokens   int     `json:"prompt_tokens"`
	CompletionToks int     `json:"completion_tokens"`
	CropPath       string  `json:"crop_path"`
	Error          string  `json:"error,omitempty"`
}

// RunConfig configures an R1-A execution.
type RunConfig struct {
	StoreDir    string
	PDFPath     string // optional override; empty -> store's own source object
	Endpoint    string
	Model       string
	Temperature float64
	MaxTokens   int
	RunDir      string // experiments/parrot-perceptual-envelope-r1/runs/<id>
}

// RunContextEnvelope executes R1-A: every base x every context level.
// Deterministic order; one model call per record; raw output preserved.
func RunContextEnvelope(ctx context.Context, cfg RunConfig, alloc Allocation) ([]RecordOutcome, error) {
	provider, err := parrotlab.NewPDFMemoryProvider(cfg.StoreDir, cfg.PDFPath)
	if err != nil {
		return nil, fmt.Errorf("page provider: %w", err)
	}
	// target.OpenAICompat expects an OpenAI-style base ending in /v1; the
	// experiment endpoint is the bare host.
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	client := target.OpenAICompat{BaseURL: baseURL, Model: cfg.Model, Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens}

	pageDir := filepath.Join(cfg.RunDir, "pages")
	cropDir := filepath.Join(cfg.RunDir, "crops")
	rawDir := filepath.Join(cfg.RunDir, "raw")
	for _, d := range []string{pageDir, cropDir, rawDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	var out []RecordOutcome
	for _, base := range alloc.Bases {
		cand := base.Candidate
		cuedPath := filepath.Join(pageDir, fmt.Sprintf("%s_cued.png", base.BaseID))
		if _, statErr := os.Stat(cuedPath); statErr != nil {
			pagePNG, rerr := provider.RenderPNG(cand.Page)
			if rerr != nil {
				return nil, fmt.Errorf("render page %d for %s: %w", cand.Page, base.BaseID, rerr)
			}
			cued, _, cerr := RenderCuedPage(pagePNG, cand)
			if cerr != nil {
				return nil, fmt.Errorf("cue page for %s: %w", base.BaseID, cerr)
			}
			if werr := os.WriteFile(cuedPath, cued, 0o644); werr != nil {
				return nil, werr
			}
		}

		for _, level := range AllContextLevels {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			default:
			}
			rec := RecordOutcome{
				BaseID: base.BaseID, CandidateID: cand.CandidateID, Stage: alloc.Stage,
				Level: string(level), Page: cand.Page, Gold: cand.NormalizedTarget,
			}
			cropPath := filepath.Join(cropDir, fmt.Sprintf("%s_%s.png", base.BaseID, strings.ToLower(string(level))))
			exposure, cerr := WriteContextVariant(cuedPath, cropPath, cand, level)
			if cerr != nil {
				rec.Error = cerr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			rec.VisualExposure = exposure
			rec.CropPath = cropPath
			if w, h, derr := pageDimsFromPNG(cropPath); derr == nil {
				rec.CropWidth, rec.CropHeight, rec.PixelArea = w, h, w*h
			}

			img, rerr := os.ReadFile(cropPath)
			if rerr != nil {
				rec.Error = rerr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			start := time.Now()
			result, ierr := client.CompletePerception(ctx, target.PerceptionInput{
				Question: FrozenInstruction, Image: img, MediaType: "image/png",
			})
			rec.LatencyMS = time.Since(start).Milliseconds()
			if ierr != nil {
				rec.Error = ierr.Error()
				out = append(out, rec)
				writeRaw(rawDir, rec)
				continue
			}
			rec.RawText = result.Content
			rec.PromptTokens = result.PromptTokensReported
			rec.CompletionToks = result.CompletionTokensReported
			scoreRecord(&rec, cand.NormalizedTarget)
			out = append(out, rec)
			writeRaw(rawDir, rec)
		}
	}
	return out, nil
}

func writeRaw(rawDir string, rec RecordOutcome) {
	body, _ := json.MarshalIndent(rec, "", "  ")
	_ = os.WriteFile(filepath.Join(rawDir, fmt.Sprintf("%s_%s.json", rec.BaseID, strings.ToLower(rec.Level))), body, 0o644)
}

// scoreRecord fills the correctness / contract / failure-taxonomy fields
// from the raw model text and the frozen gold.
func scoreRecord(rec *RecordOutcome, gold string) {
	raw := strings.TrimSpace(rec.RawText)
	rec.SemanticCorrect = decompositionlab.ScoreSemantic(FrozenOpcode, raw, gold)

	compact := strings.TrimSpace(strings.Trim(raw, ".,:;"))
	switch {
	case raw == "":
		rec.Abstained = true
		rec.FailureClass = "ABSTAIN"
	case abstainWords.MatchString(raw) && digitsOnly.ReplaceAllString(raw, "") == "":
		rec.Abstained = true
		rec.FailureClass = "ABSTAIN"
	case numberLike.MatchString(compact):
		rec.ContractSuccess = true
	case len(strings.Fields(raw)) > 3:
		rec.UnsupportedAssertion = true
		rec.FailureClass = "COMMENTARY_CONTAMINATION"
	default:
		rec.FormatFailure = true
		rec.FailureClass = "OTHER"
	}
	if rec.SemanticCorrect {
		rec.ContractSuccess = true
		rec.FailureClass = ""
		rec.EditDistance = 0
		return
	}
	got := digitsOnly.ReplaceAllString(raw, "")
	want := digitsOnly.ReplaceAllString(gold, "")
	rec.EditDistance = levenshtein(got, want)
	if rec.FailureClass == "" || rec.FailureClass == "OTHER" {
		rec.FailureClass = classifyDigits(got, want)
	}
}

func classifyDigits(got, want string) string {
	switch {
	case got == "":
		return "ABSTAIN"
	case got == want:
		return "SEPARATOR_OR_FORMAT_ONLY"
	case len(got) == len(want):
		return "DIGIT_SUBSTITUTION"
	case len(got) < len(want) && strings.HasSuffix(want, got):
		return "PREFIX_TRUNCATION"
	case len(got) < len(want) && strings.HasPrefix(want, got):
		return "SUFFIX_TRUNCATION"
	case len(got) < len(want):
		return "DIGIT_DELETION"
	case len(got) > len(want) && strings.Contains(got, want):
		return "DIGIT_INSERTION"
	default:
		return "WRONG_NUMBER"
	}
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
