package lfm2boundary

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const RequiredModel = "lfm2-vl-1.6b"
const RequiredContext = 4096

type ModelInstance struct {
	ID     string `json:"id"`
	Config struct {
		ContextLength int `json:"context_length"`
		Parallel      int `json:"parallel,omitempty"`
	} `json:"config"`
}

type ListedModel struct {
	Type         string `json:"type"`
	Key          string `json:"key"`
	DisplayName  string `json:"display_name"`
	Quantization *struct { Name string `json:"name"` } `json:"quantization"`
	Capabilities struct { Vision bool `json:"vision"` } `json:"capabilities"`
	LoadedInstances []ModelInstance `json:"loaded_instances"`
}

type PreflightResult struct {
	Model          string `json:"model"`
	ModelKey       string `json:"model_key"`
	InstanceID     string `json:"instance_id"`
	ContextLength  int    `json:"context_length"`
	ServerParallel int    `json:"server_parallel,omitempty"`
	Vision         bool   `json:"vision"`
	Quantization   string `json:"quantization,omitempty"`
}

func serverRoot(endpoint string) string {
	s := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	for _, suffix := range []string{"/api/v1", "/v1"} {
		if strings.HasSuffix(s, suffix) { return strings.TrimSuffix(s, suffix) }
	}
	return s
}

// Preflight is deliberately read-only: it never loads, reloads, or changes the
// model. The campaign characterizes the user's already-loaded 4096-token F16
// instance rather than silently changing the experimental condition.
func Preflight(ctx context.Context, endpoint, model string) (PreflightResult, error) {
	if strings.TrimSpace(model) != RequiredModel {
		return PreflightResult{}, fmt.Errorf("model must be exactly %q, got %q", RequiredModel, model)
	}
	root := serverRoot(endpoint)
	if root == "" { root = "http://127.0.0.1:1234" }
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second); defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+"/api/v1/models", nil)
	if err != nil { return PreflightResult{}, err }
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return PreflightResult{}, fmt.Errorf("LM Studio preflight: %w", err) }
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil { return PreflightResult{}, err }
	if resp.StatusCode/100 != 2 { return PreflightResult{}, fmt.Errorf("LM Studio preflight status %s: %s", resp.Status, strings.TrimSpace(string(raw))) }
	var listing struct { Models []ListedModel `json:"models"` }
	if err := json.Unmarshal(raw, &listing); err != nil { return PreflightResult{}, fmt.Errorf("decode LM Studio model list: %w", err) }
	for _, m := range listing.Models {
		if m.Key != RequiredModel { continue }
		if m.Type != "llm" { return PreflightResult{}, fmt.Errorf("%s is not an LLM", RequiredModel) }
		if !m.Capabilities.Vision { return PreflightResult{}, fmt.Errorf("%s does not report vision capability", RequiredModel) }
		quant := ""
		if m.Quantization != nil { quant = strings.ToUpper(strings.TrimSpace(m.Quantization.Name)) }
		if quant != "F16" { return PreflightResult{}, fmt.Errorf("%s must be the F16 variant, got %q", RequiredModel, quant) }
		for _, inst := range m.LoadedInstances {
			if inst.ID != RequiredModel { continue }
			if inst.Config.ContextLength != RequiredContext { return PreflightResult{}, fmt.Errorf("%s context must be %d, got %d", RequiredModel, RequiredContext, inst.Config.ContextLength) }
			return PreflightResult{Model:RequiredModel, ModelKey:m.Key, InstanceID:inst.ID, ContextLength:inst.Config.ContextLength, ServerParallel:inst.Config.Parallel, Vision:true, Quantization:quant}, nil
		}
		return PreflightResult{}, fmt.Errorf("%s is available but not loaded with exact instance id %q", RequiredModel, RequiredModel)
	}
	return PreflightResult{}, fmt.Errorf("LM Studio does not list exact model key %q", RequiredModel)
}
