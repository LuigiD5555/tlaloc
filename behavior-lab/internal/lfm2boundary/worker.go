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
	QuestionID       string `json:"question_id"`
	Responsibility   string `json:"responsibility"`
	ImagePath        string `json:"image_path"`
	Text             string `json:"text"`
	Normalized       string `json:"normalized"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
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
	case "Q0", "Q6", "Q8": return "rosetta", true
	case "Q1", "Q2": return "cells", true
	case "Q3", "Q4": return "transitions", true
	case "Q5", "Q7": return "timeline", true
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

func roleAllows(role, qid string) bool {
	want, ok := ResponsibilityForQuestion(qid)
	return ok && strings.EqualFold(strings.TrimSpace(role), want)
}

func RunVisualSpecialist(ctx context.Context, role string, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var task VisualTask
	if err := json.Unmarshal(req.Input, &task); err != nil { return tlaloque.CapabilityResponse{}, fmt.Errorf("decode visual task: %w", err) }
	qid := QuestionIDFromNode(req.NodeID)
	if !roleAllows(role, qid) { return tlaloque.CapabilityResponse{}, fmt.Errorf("responsibility %s cannot answer %s", role, qid) }
	q, ok := question(qid); if !ok { return tlaloque.CapabilityResponse{}, fmt.Errorf("unknown benchmark question %q", qid) }
	imagePath := task.ImagePath
	if strings.EqualFold(task.Condition, "BLACKBOARD_CROPPED") {
		cropID := cropForQuestion(qid)
		imagePath = task.Crops[cropID]
		if strings.TrimSpace(imagePath) == "" { return tlaloque.CapabilityResponse{}, fmt.Errorf("missing declared crop %s for %s", cropID, qid) }
	}
	img, err := os.ReadFile(imagePath); if err != nil { return tlaloque.CapabilityResponse{}, err }
	system := ""
	switch strings.ToUpper(strings.TrimSpace(task.Condition)) {
	case "NATIVE_PNG_ONLY":
		system = ""
	case "R4_ASSISTED":
		system = task.R4Prompt
		if strings.TrimSpace(system) == "" { return tlaloque.CapabilityResponse{}, fmt.Errorf("R4_ASSISTED requires Master Prompt R4") }
	case "BLACKBOARD_CROPPED":
		system = "Inspect only the supplied Origami visual crop for your assigned responsibility. Report only visually supported evidence. Do not use OCR, source code, ground truth, a payload decoder, or an exact decoder. If an exact hidden value cannot be verified from the permitted semantic view, answer UNKNOWN."
	default:
		return tlaloque.CapabilityResponse{}, fmt.Errorf("unsupported condition %q", task.Condition)
	}
	client := target.OpenAICompat{BaseURL:task.Endpoint, Model:task.Model, Temperature:0, MaxTokens:task.MaxTokens}
	result, err := client.CompletePerception(ctx, target.PerceptionInput{SystemPrompt:system, Question:q, Image:img, MediaType:"image/png"})
	if err != nil { return tlaloque.CapabilityResponse{}, err }
	norm := NormalizeEvidence(result.Content)
	out := VisualOutput{QuestionID:qid, Responsibility:role, ImagePath:imagePath, Text:result.Content, Normalized:norm, PromptTokens:result.PromptTokensReported, CompletionTokens:result.CompletionTokensReported}
	body, _ := json.Marshal(out)
	value, _ := json.Marshal(norm)
	return tlaloque.CapabilityResponse{Output:body, Confidence:1, Observations:[]blackboard.Observation{{Key:qid, Value:value, Confidence:1, References:[]string{imagePath}, Provenance:map[string]string{"condition":task.Condition,"responsibility":role,"source":"lfm2-vl-1.6b-visual"}}}}, nil
}

func RunConsolidator(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	if req.Blackboard == nil { return tlaloque.CapabilityResponse{}, fmt.Errorf("consolidator requires blackboard snapshot") }
	out := ConsolidatedOutput{Responses:map[string]string{}, Consensus:map[string]string{}}
	observations := []blackboard.Observation{}
	for _, q := range temporalbench.CanonicalQuestions() {
		c, err := blackboard.Consolidate(*req.Blackboard, q.ID, nil); if err != nil { return tlaloque.CapabilityResponse{}, err }
		out.Consensus[q.ID] = c.Status
		answer := "UNKNOWN"
		if c.Status == blackboard.ConsensusConfirmed && len(c.Value) > 0 {
			if err := json.Unmarshal(c.Value, &answer); err != nil { return tlaloque.CapabilityResponse{}, err }
		}
		out.Responses[q.ID] = answer
		value, _ := json.Marshal(map[string]any{"status":c.Status,"answer":answer,"votes":c.Votes,"required_votes":c.RequiredVotes,"observation_ids":c.ObservationIDs})
		observations = append(observations, blackboard.Observation{Key:"decision."+q.ID, Value:value, Confidence:1, References:c.ObservationIDs, Provenance:map[string]string{"source":"deterministic-quorum"}})
	}
	body, _ := json.Marshal(out)
	return tlaloque.CapabilityResponse{Output:body, Confidence:1, Observations:observations}, nil
}
