package exocortex

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/parrotpresent"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// ParrotTlaloqueR1IDPrefix is the CapabilityDescriptor.ID prefix for the
// R1-aware Parrot Tlaloque, one worker per R1-covered opcode.
const ParrotTlaloqueR1IDPrefix = "parrot-tlaloque-r1"

// perceptionCompleter is the one model call the Parrot Tlaloque makes. It
// is an interface so tests can inject a fake endpoint.
type perceptionCompleter interface {
	CompletePerception(context.Context, target.PerceptionInput) (target.PerceptionResult, error)
}

// ParrotR1Region is the operand geometry this Tlaloque consumes. It is the
// documented output shape of the LOCATE_REGION capability (page + bbox +
// page dimensions) — evidence, not a preprocessing recipe.
type ParrotR1Region struct {
	Page       int                `json:"page"`
	BBox       *canonicaldoc.BBox `json:"bbox"`
	PageWidth  float64            `json:"page_width"`
	PageHeight float64            `json:"page_height"`
}

// ParrotR1Input is the Parrot Tlaloque's input contract: one opcode, one
// page image, the located operand region. Nothing executor-specific.
type ParrotR1Input struct {
	Opcode                   string          `json:"opcode"`
	PageImagePath            string          `json:"page_image_path"`
	Region                   *ParrotR1Region `json:"region"`
	CompetingNumericOperands int             `json:"competing_numeric_operands,omitempty"`
}

// ParrotR1Output is the raw, unverified result plus the full pre-inference
// decision trace.
type ParrotR1Output struct {
	Opcode                string           `json:"opcode"`
	Text                  string           `json:"text"`
	PromptTokens          int              `json:"prompt_tokens"`
	CompletionTokens      int              `json:"completion_tokens"`
	ModelCalls            int              `json:"model_calls"`
	SubmittedLineHeightPx int              `json:"submitted_line_height_px"`
	PreparedImageSHA256   string           `json:"prepared_image_sha256,omitempty"`
	ProfileVersion        string           `json:"profile_version"`
	ProfileHash           string           `json:"profile_hash"`
	AdapterDecision       *AdaptDecisionR1 `json:"adapter_decision"`
}

// ParrotTlaloqueR1 wraps one opcode of the LFM2-VL 1.6B executor under
// CapabilityProfile R1. Every Execute call runs AdapterR1.Prepare against
// the frozen profile FIRST: a request outside the measured envelope is
// rejected here (zero model calls). At most one cognitive model call is
// ever made.
type ParrotTlaloqueR1 struct {
	opcode      string
	profile     CapabilityProfileR1
	instruction string
	endpoint    ParrotEndpoint
	workDir     string
	client      perceptionCompleter
}

// NewParrotTlaloqueR1 loads and hash-validates the frozen CapabilityProfile
// R1 at profilePath, then builds one Tlaloque per R1-covered opcode that
// T1 requires (EXTRACT_NUMBER). expectedHash is the hash the caller
// demands (full or a prefix); a mismatch is a hard error. Returns the
// workers plus the profile's id and hash.
func NewParrotTlaloqueR1(profilePath, expectedHash, workDir string, endpoint ParrotEndpoint) ([]tlaloque.CapabilityWorker, string, string, error) {
	profile, err := LoadCapabilityProfileR1(profilePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("load profile: %w", err)
	}
	computed, err := ComputeProfileR1Hash(profile)
	if err != nil {
		return nil, "", "", err
	}
	if profile.ProfileHash == "" || computed != profile.ProfileHash {
		return nil, "", "", fmt.Errorf("profile hash is not self-consistent: embedded %q, computed %q", profile.ProfileHash, computed)
	}
	if want := strings.TrimSpace(expectedHash); want != "" && !strings.HasPrefix(profile.ProfileHash, want) {
		return nil, "", "", fmt.Errorf("profile hash %q does not match expected %q", profile.ProfileHash, want)
	}
	if strings.TrimSpace(workDir) == "" {
		workDir = os.TempDir()
	}

	instruction, err := FixedInstruction(OpExtractNumber)
	if err != nil {
		return nil, "", "", err
	}
	client := target.OpenAICompat{
		BaseURL: endpoint.BaseURL, Model: endpoint.Model,
		Temperature: endpoint.Temperature, MaxTokens: endpoint.MaxTokens,
	}
	worker := &ParrotTlaloqueR1{
		opcode: OpExtractNumber, profile: profile, instruction: instruction,
		endpoint: endpoint, workDir: workDir, client: client,
	}
	return []tlaloque.CapabilityWorker{worker}, profile.ProfileID, profile.ProfileHash, nil
}

func (p *ParrotTlaloqueR1) id() string {
	return ParrotTlaloqueR1IDPrefix + ":" + p.profile.Executor.ModelID + ":" + p.opcode
}

func (p *ParrotTlaloqueR1) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: p.id(), Capability: p.opcode, Engine: tlaloque.EngineModel,
		InputSchema: "exocortex.parrot-r1-input.r0", OutputSchema: "exocortex.parrot-r1-output.r0",
		Deterministic: false, ParameterCount: 1_600_000_000, MaxConcurrency: 1,
		Tags: []string{"capability-profile:" + p.profile.ProfileID},
	}.Normalize()
	return d
}

func (p *ParrotTlaloqueR1) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in ParrotR1Input
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot r1 tlaloque: decode input: %w", err)
	}
	opcode := strings.ToUpper(strings.TrimSpace(in.Opcode))
	if opcode == "" {
		opcode = p.opcode
	}
	if opcode != p.opcode {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot r1 tlaloque: this worker serves %s, not %s (one cognitive opcode per call)", p.opcode, opcode)
	}

	// --- observable runtime properties only ---
	hasVisualOperand := in.PageImagePath != "" && in.Region != nil && in.Region.BBox != nil && in.Region.PageWidth > 0 && in.Region.PageHeight > 0
	var lineHeightPx float64
	var imageHeightPx int
	if hasVisualOperand {
		if _, err := os.Stat(in.PageImagePath); err != nil {
			hasVisualOperand = false
		} else if file, err := os.Open(in.PageImagePath); err == nil {
			if cfg, _, err := image.DecodeConfig(file); err == nil {
				imageHeightPx = cfg.Height
				lineHeightPx = (in.Region.BBox.Y2 - in.Region.BBox.Y1) / in.Region.PageHeight * float64(cfg.Height)
			}
			_ = file.Close()
		}
	}
	visualField := "FULL_PAGE"
	if !hasVisualOperand {
		visualField = ""
	}

	adapter := AdapterR1{Profile: p.profile}
	decision, err := adapter.Prepare(AdaptRequestR1{
		Opcode:                   p.opcode,
		HasVisualOperand:         hasVisualOperand,
		LineHeightPx:             lineHeightPx,
		VisualFieldName:          visualField,
		CompetingNumericOperands: in.CompetingNumericOperands,
	})
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot r1 tlaloque: adapter: %w", err)
	}

	out := ParrotR1Output{
		Opcode: p.opcode, ModelCalls: decision.ModelCallCount,
		ProfileVersion: p.profile.ProfileVersion, ProfileHash: p.profile.ProfileHash,
		AdapterDecision: &decision,
	}

	// --- pre-inference rejection: zero model calls ---
	if decision.Rejected {
		out.ModelCalls = 0
		status := "UNKNOWN"
		if strings.EqualFold(decision.FallbackAction, "UNSUPPORTED") {
			status = "UNSUPPORTED"
		}
		return p.respond(req, out, status, 0, decision, "")
	}

	// --- profile-earned preventive presentation BEFORE the call ---
	plan := parrotpresent.Plan{
		CropToLine:     transformApplied(decision, "CROP_TO_OPERAND_LINE"),
		Upscale:        transformApplied(decision, "UPSCALE_TO_PREFERRED"),
		IsolateOperand: transformApplied(decision, "ISOLATE_OPERAND"),
	}
	if value, ok := decision.ResultingWorkingSet["target_line_height_px"].(float64); ok {
		plan.TargetLineHeightPx = int(value)
	}
	// The submitted visual field is the operand line; if the profile did
	// not ask for a crop but we hold only a full page, cropping to the
	// line is still the honest minimum working set.
	if !plan.CropToLine {
		plan.CropToLine = true
	}

	region := parrotpresent.Region{
		Line: parrotpresent.BBox{
			X1: in.Region.BBox.X1, Y1: in.Region.BBox.Y1,
			X2: in.Region.BBox.X2, Y2: in.Region.BBox.Y2,
		},
		PageWidth: in.Region.PageWidth, PageHeight: in.Region.PageHeight,
	}
	outPath := filepath.Join(p.workDir, "parrot-r1-"+sanitize(req.NodeID)+".png")
	prepared, err := parrotpresent.Prepare(in.PageImagePath, region, plan, outPath)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot r1 tlaloque: prepare working set: %w", err)
	}
	out.SubmittedLineHeightPx = prepared.SubmittedLineHeightPx
	out.PreparedImageSHA256 = prepared.SHA256
	_ = imageHeightPx

	// --- exactly one cognitive model call ---
	result, err := p.client.CompletePerception(ctx, target.PerceptionInput{
		Question: p.instruction, Image: prepared.Bytes, MediaType: "image/png",
	})
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot r1 tlaloque: model call: %w", err)
	}
	out.Text = strings.TrimSpace(result.Content)
	out.PromptTokens = result.PromptTokensReported
	out.CompletionTokens = result.CompletionTokensReported
	out.ModelCalls = 1

	return p.respond(req, out, "", 1, decision, prepared.OutputPath)
}

func (p *ParrotTlaloqueR1) respond(req tlaloque.CapabilityRequest, out ParrotR1Output, status string, modelCalls int, decision AdaptDecisionR1, preparedPath string) (tlaloque.CapabilityResponse, error) {
	body, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	// The blackboard observation value is the raw extracted text (for a
	// rejection: empty). It is an OBSERVATION only — a Verify Tlaloque
	// promotes it, never this one.
	obsValue, _ := json.Marshal(out.Text)
	provenance := map[string]string{
		"source":                   p.id(),
		"opcode":                   p.opcode,
		"model":                    p.endpoint.Model,
		"profile_version":          p.profile.ProfileVersion,
		"profile_hash":             p.profile.ProfileHash,
		"adapter_rules_applied":    strings.Join(decision.RulesApplied, ","),
		"deterministic_transforms": fmt.Sprintf("%d", len(decision.Transformations)),
		"submitted_line_height_px": fmt.Sprintf("%d", out.SubmittedLineHeightPx),
		"status":                   status,
	}
	references := []string{}
	if preparedPath != "" {
		references = append(references, preparedPath)
	}
	usage := &tlaloque.WorkerUsage{
		TokensIn: out.PromptTokens, TokensOut: out.CompletionTokens, UpstreamCalls: modelCalls,
	}
	return tlaloque.CapabilityResponse{
		WorkerID: p.id(), Output: body, Confidence: 1, Usage: usage,
		Observations: []blackboard.Observation{{
			Key: req.NodeID, Value: obsValue, References: references, Provenance: provenance,
		}},
	}, nil
}

func transformApplied(decision AdaptDecisionR1, kind string) bool {
	for _, transform := range decision.Transformations {
		if transform.Kind == kind {
			return true
		}
	}
	return false
}

func sanitize(value string) string {
	return strings.Map(func(r rune) rune {
		if r == ':' || r == '/' || r == ' ' || r == '\\' {
			return '_'
		}
		return r
	}, value)
}
