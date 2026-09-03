package exocortex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// ParrotTlaloqueID is the CapabilityDescriptor.ID this Tlaloque registers
// under, per opcode (see NewParrotTlaloques). Every invocation carries
// exactly one Micro-ISA opcode (E0.2, E3): the raw workflow never bypasses
// the ModelAdapter to reach the model directly.
const ParrotTlaloqueIDPrefix = "parrot-tlaloque"

// ParrotInput is the Parrot Tlaloque's input contract. It is deliberately
// tiny (E0.8, E3B): one crop, one opcode's operand shape, nothing else.
type ParrotInput struct {
	ImagePath   string   `json:"image_path"`
	VisualField string   `json:"visual_field,omitempty"` // TIGHT_CROP | FULL_PAGE
	CharCount   int      `json:"char_count,omitempty"`
	ChoiceWidth int      `json:"choice_width,omitempty"`
	Choices     []string `json:"choices,omitempty"` // SELECT_ONE task choice labels (never the answer)
}

// ParrotOutput is the raw, unverified model output. It is always written
// to the Blackboard as an Observation, never a Fact (E0.6, E0.12).
type ParrotOutput struct {
	Opcode           string `json:"opcode"`
	Text             string `json:"text"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
}

// ParrotEndpoint is the transport configuration for one Parrot deployment.
// It is supplied by the caller (the T0 runner), never invented at runtime.
type ParrotEndpoint struct {
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
}

// ParrotTlaloque wraps one opcode of one Parrot deployment. Every Execute
// call goes through ModelAdapter.Adapt first: a request outside the
// CapabilityProfile's measured envelope is rejected with
// CAPABILITY_CONTRACT_VIOLATION and never reaches the model (E3).
type ParrotTlaloque struct {
	Opcode   string
	Profile  CapabilityProfile
	Endpoint ParrotEndpoint
}

// NewParrotTlaloques builds one ParrotTlaloque per opcode the profile
// deploys (DEPLOY or DEPLOY_WITH_CONSTRAINTS only — an EXTERNALIZE/
// DO_NOT_DEPLOY opcode gets no worker at all, so the CapabilityRouter can
// never accidentally select Parrot for it even without a veto).
func NewParrotTlaloques(profile CapabilityProfile, endpoint ParrotEndpoint) []ParrotTlaloque {
	var out []ParrotTlaloque
	for _, entry := range profile.Capabilities {
		if entry.DeploymentRecommendation != DeploymentDeploy && entry.DeploymentRecommendation != DeploymentDeployConstrained {
			continue
		}
		if _, err := FixedInstruction(entry.Opcode); err != nil {
			continue // opcode has no fixed one-op instruction; not Parrot-eligible
		}
		out = append(out, ParrotTlaloque{Opcode: entry.Opcode, Profile: profile, Endpoint: endpoint})
	}
	return out
}

func (p ParrotTlaloque) id() string {
	return ParrotTlaloqueIDPrefix + ":" + p.Profile.ExecutorID + ":" + p.Opcode
}

func (p ParrotTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: p.id(), Capability: p.Opcode, Engine: tlaloque.EngineModel,
		InputSchema: "exocortex.parrot-input.r0", OutputSchema: "exocortex.parrot-output.r0",
		Deterministic: false, ParameterCount: 1_600_000_000, MaxConcurrency: 1,
	}.Normalize()
	return d
}

func (p ParrotTlaloque) Execute(ctx context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in ParrotInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot tlaloque: decode input: %w", err)
	}
	adapter := ModelAdapter{Profile: p.Profile}
	plan, err := adapter.Adapt(p.Opcode, Operand{VisualField: in.VisualField, CharCount: in.CharCount, ChoiceWidth: in.ChoiceWidth, Choices: in.Choices})
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	image, err := os.ReadFile(in.ImagePath)
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot tlaloque: read image: %w", err)
	}
	client := target.OpenAICompat{BaseURL: p.Endpoint.BaseURL, Model: p.Endpoint.Model, Temperature: p.Endpoint.Temperature, MaxTokens: p.Endpoint.MaxTokens}
	result, err := client.CompletePerception(ctx, target.PerceptionInput{SystemPrompt: "", Question: plan.Instruction, Image: image, MediaType: "image/png"})
	if err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("parrot tlaloque: %w", err)
	}
	out := ParrotOutput{Opcode: p.Opcode, Text: result.Content, PromptTokens: result.PromptTokensReported, CompletionTokens: result.CompletionTokensReported}
	body, err := json.Marshal(out)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	// The raw text is what downstream Normalize/Verify Tlaloques consume;
	// it is written as an Observation only (E0.6) — status is UNVERIFIED
	// by construction until a Verify Tlaloque promotes it.
	obsValue, _ := json.Marshal(out.Text)
	return tlaloque.CapabilityResponse{
		WorkerID: p.id(), Output: body, Confidence: 1,
		Observations: []blackboard.Observation{{
			Key: req.NodeID, Value: obsValue, References: []string{in.ImagePath},
			Provenance: map[string]string{"source": p.id(), "opcode": p.Opcode, "model": p.Endpoint.Model, "visual_field": plan.VisualField},
		}},
	}, nil
}
