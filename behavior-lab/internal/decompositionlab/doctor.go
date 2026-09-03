package decompositionlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/exocortex"
)

// DoctorResult is what `tlaloc-exocortex-decomposition doctor` reports
// before any scientific run: every referenced artifact exists and hash-
// verifies, the compiled CapabilityProfile is valid, and (when an endpoint
// is reachable) the requested model is actually being served. Doctor never
// fabricates readiness — a check that cannot run leaves Ready=false with a
// reason.
type DoctorResult struct {
	Manifest          Manifest                    `json:"manifest"`
	Profile           exocortex.CapabilityProfile `json:"profile"`
	DiscoveredModels  []string                    `json:"discovered_models,omitempty"`
	EndpointReachable bool                        `json:"endpoint_reachable"`
	ModelServed       bool                        `json:"model_served"`
	Reasons           []string                    `json:"reasons,omitempty"`
	Ready             bool                        `json:"ready"`
}

// Doctor validates the T0 evidence chain end to end: P0 dataset hash and
// shape, P2-A artifact freeze/hash and its compiled CapabilityProfile, and
// (best-effort, never fatal to the other checks) whether the configured
// LM Studio-compatible endpoint currently serves the requested model.
func Doctor(ctx context.Context, spec Spec) (DoctorResult, error) {
	var result DoctorResult
	manifest, _, err := Freeze(spec)
	if err != nil {
		return DoctorResult{}, err
	}
	result.Manifest = manifest

	profile, err := exocortex.CompileParrotProfile(spec.MicroISAArtifactPath, spec.ExecutorID, spec.ModelID, spec.ProfileVersion)
	if err != nil {
		return DoctorResult{}, fmt.Errorf("compile capability profile: %w", err)
	}
	if err := exocortex.VerifySourceArtifact(profile); err != nil {
		return DoctorResult{}, err
	}
	result.Profile = profile

	if strings.TrimSpace(spec.Endpoint) == "" {
		result.Reasons = append(result.Reasons, "no endpoint configured; scientific run cannot proceed")
		return result, nil
	}
	models, err := discoverModels(ctx, spec.Endpoint)
	if err != nil {
		result.Reasons = append(result.Reasons, fmt.Sprintf("endpoint not reachable: %v", err))
		return result, nil
	}
	result.EndpointReachable = true
	result.DiscoveredModels = models
	for _, m := range models {
		if m == spec.ModelID {
			result.ModelServed = true
		}
	}
	if !result.ModelServed {
		result.Reasons = append(result.Reasons, fmt.Sprintf("endpoint does not report model %q; available: %s", spec.ModelID, strings.Join(models, ", ")))
	}
	result.Ready = result.EndpointReachable && result.ModelServed
	return result, nil
}

func discoverModels(ctx context.Context, base string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode model list: %w", err)
	}
	models := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		if strings.TrimSpace(m.ID) != "" {
			models = append(models, m.ID)
		}
	}
	sort.Strings(models)
	return models, nil
}
