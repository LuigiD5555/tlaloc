package lfm2boundary

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/temporalbench"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const VisualCapability = "LFM2_VISUAL_READ"
const ConsolidateCapability = "CONSOLIDATE_BLACKBOARD"

type VisualTask struct {
	Endpoint  string            `json:"endpoint"`
	Model     string            `json:"model"`
	ImagePath string            `json:"image_path"`
	Crops     map[string]string `json:"crops,omitempty"`
	Condition string            `json:"condition"`
	R4Prompt  string            `json:"r4_prompt,omitempty"`
	MaxTokens int               `json:"max_tokens,omitempty"`
}

type VisualOutput struct {
	QuestionID       string          `json:"question_id,omitempty"`
	Responsibility   string          `json:"responsibility"`
	ImagePath        string          `json:"image_path"`
	Text             string          `json:"text"`
	Normalized       string          `json:"normalized,omitempty"`
	Structured       json.RawMessage `json:"structured,omitempty"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
}

type ConsolidatedOutput struct {
	Responses map[string]string `json:"responses"`
	Consensus map[string]string `json:"consensus"`
}

func QuestionIDFromNode(nodeID string) string {
	id := strings.ToUpper(strings.TrimSpace(nodeID))
	if i := strings.IndexByte(id, '-'); i >= 0 { id = id[:i] }
	return id
}

func ResponsibilityForQuestion(qid string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(qid)) {
	case "Q0", "Q6", "Q8": return RoleRosetta, true
	case "Q1", "Q2": return RoleCells, true
	case "Q3", "Q4": return RoleTransitions, true
	case "Q5", "Q7": return RoleTimeline, true
	default: return "", false
	}
}

func cropForQuestion(qid string) string {
	switch strings.ToUpper(strings.TrimSpace(qid)) {
	case "Q0", "Q6": return "BOOT_T1"
	case "Q1", "Q2", "Q3", "Q4": return "T2"
	case "Q5": return "TIMELINE"
	case "Q7", "Q8": return "SEMANTIC_FULL"
	default: return ""
	}
}

func question(id string) (string, bool) {
	for _, q := range temporalbench.CanonicalQuestions() {
		if q.ID == strings.ToUpper(strings.TrimSpace(id)) { return q.Text, true }
	}
	return "", false
}

func NormalizeEvidence(s string) string {
	set := map[string]bool{}
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 { return }
		set[strings.ToUpper(b.String())] = true
		b.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' { b.WriteRune(unicode.ToUpper(r)) } else { flush() }
	}
	flush()
	terms := make([]string, 0, len(set)); for k := range set { terms = append(terms, k) }; sort.Strings(terms)
	return strings.Join(terms, " ")
}

func RunVisualSpecialist(ctx context.Context, role string, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var task VisualTask
	if err := json.Unmarshal(req.Input, &task); err != nil { return tlaloque.CapabilityResponse{}, fmt.Errorf("decode visual task: %w", err) }
	condition := strings.ToUpper(strings.TrimSpace(task.Condition))
	resolvedRole, ok := RoleFromNodeID(req.NodeID)
	if !ok || !strings.EqualFold(resolvedRole, role) { return tlaloque.CapabilityResponse{}, fmt.Errorf("responsibility %s cannot serve node %s", role, req.NodeID) }

	imagePath := task.ImagePath
	prompt := ""
	system := ""
	qid := ""
	observationKey := ""
	if condition == "BLACKBOARD_CROPPED" {
		cropID := RoleCrop(role)
		imagePath = task.Crops[cropID]
		if strings.TrimSpace(imagePath) == "" { return tlaloque.CapabilityResponse{}, fmt.Errorf("missing declared crop %s for %s", cropID, role) }
		var err error
		prompt, err = RolePrompt(role); if err != nil { return tlaloque.CapabilityResponse{}, err }
		system = "You are one bounded visual specialist in Tlaloc. Follow the requested JSON schema exactly and do not infer hidden exact payload values."
		observationKey = strings.ToLower(role)
	} else {
		qid = QuestionIDFromNode(req.NodeID)
		wantRole, ok := ResponsibilityForQuestion(qid)
		if !ok || !strings.EqualFold(wantRole, role) { return tlaloque.CapabilityResponse{}, fmt.Errorf("responsibility %s cannot answer %s", role, qid) }
		var found bool
		prompt, found = question(qid); if !found { return tlaloque.CapabilityResponse{}, fmt.Errorf("unknown benchmark question %q", qid) }
		observationKey = qid
		switch condition {
		case "NATIVE_PNG_ONLY":
			system = ""
		case "R4_ASSISTED":
			system = task.R4Prompt
			if strings.TrimSpace(system) == "" { return tlaloque.CapabilityResponse{}, fmt.Errorf("R4_ASSISTED requires Master Prompt R4") }
		default:
			return tlaloque.CapabilityResponse{}, fmt.Errorf("unsupported condition %q", task.Condition)
		}
	}

	img, err := os.ReadFile(imagePath); if err != nil { return tlaloque.CapabilityResponse{}, err }
	client := target.OpenAICompat{BaseURL:task.Endpoint, Model:task.Model, Temperature:0, MaxTokens:task.MaxTokens}
	result, err := client.CompletePerception(ctx, target.PerceptionInput{SystemPrompt:system, Question:prompt, Image:img, MediaType:"image/png"})
	if err != nil { return tlaloque.CapabilityResponse{}, err }
	out := VisualOutput{QuestionID:qid, Responsibility:role, ImagePath:imagePath, Text:result.Content, PromptTokens:result.PromptTokensReported, CompletionTokens:result.CompletionTokensReported}
	var observationValue json.RawMessage
	if condition == "BLACKBOARD_CROPPED" {
		out.Structured = StructuredObservation(role, result.Content)
		observationValue = out.Structured
	} else {
		out.Normalized = NormalizeEvidence(result.Content)
		observationValue, _ = json.Marshal(out.Normalized)
	}
	body, _ := json.Marshal(out)
	return tlaloque.CapabilityResponse{Output:body, Confidence:1, Observations:[]blackboard.Observation{{Key:observationKey, Value:observationValue, Confidence:1, References:[]string{imagePath}, Provenance:map[string]string{"condition":task.Condition,"responsibility":role,"source":"lfm2-vl-1.6b-visual"}}}}, nil
}

func RunConsolidator(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	if req.Blackboard == nil { return tlaloque.CapabilityResponse{}, fmt.Errorf("consolidator requires blackboard snapshot") }
	var task VisualTask
	if err := json.Unmarshal(req.Input, &task); err != nil { return tlaloque.CapabilityResponse{}, fmt.Errorf("decode consolidator task: %w", err) }
	var out ConsolidatedOutput
	var err error
	if strings.EqualFold(task.Condition, "BLACKBOARD_CROPPED") {
		out, err = SynthesizeBlackboardResponses(*req.Blackboard)
	} else {
		out = ConsolidatedOutput{Responses:map[string]string{}, Consensus:map[string]string{}}
		for _, q := range temporalbench.CanonicalQuestions() {
			c, conErr := blackboard.Consolidate(*req.Blackboard, q.ID, nil); if conErr != nil { return tlaloque.CapabilityResponse{}, conErr }
			out.Consensus[q.ID] = c.Status
			answer := "UNKNOWN"
			if c.Status == blackboard.ConsensusConfirmed && len(c.Value) > 0 { if unmarshalErr := json.Unmarshal(c.Value, &answer); unmarshalErr != nil { return tlaloque.CapabilityResponse{}, unmarshalErr } }
			out.Responses[q.ID] = answer
		}
	}
	if err != nil { return tlaloque.CapabilityResponse{}, err }
	observations := make([]blackboard.Observation,0,9)
	for _, q := range temporalbench.CanonicalQuestions() {
		answer := out.Responses[q.ID]; if answer == "" { answer = "UNKNOWN" }
		status := out.Consensus[q.ID]
		if status == "" {
			status = blackboard.ConsensusUnknown
			if answer != "UNKNOWN" { status = blackboard.ConsensusConfirmed }
		}
		value, _ := json.Marshal(map[string]any{"status":status,"answer":answer})
		observations = append(observations, blackboard.Observation{Key:"decision."+q.ID, Value:value, Confidence:1, Provenance:map[string]string{"source":"deterministic-quorum-and-simulator"}})
	}
	body, _ := json.Marshal(out)
	return tlaloque.CapabilityResponse{Output:body, Confidence:1, Observations:observations}, nil
}
