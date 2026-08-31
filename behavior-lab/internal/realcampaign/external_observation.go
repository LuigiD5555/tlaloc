package realcampaign

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ExternalObservationSchema = "tlaloc.model-interop-external-observation.r0"

type ExternalObservation struct {
	Schema          string              `json:"schema"`
	RecordedAt      string              `json:"recorded_at"`
	Stage           string              `json:"stage"`
	Outcome         string              `json:"outcome"`
	ModelInterop    ModelInteropProfile `json:"model_interop"`
	Endpoint        string              `json:"endpoint,omitempty"`
	MediaType       string              `json:"media_type,omitempty"`
	Temperature     float64             `json:"temperature,omitempty"`
	ResponseSHA256  string              `json:"response_sha256,omitempty"`
	ResponseText    string              `json:"response_text,omitempty"`
	Notes           string              `json:"notes,omitempty"`
}

type ExternalObservationRegistry struct {
	Schema       string                `json:"schema"`
	ModelID      string                `json:"model_id"`
	Family       string                `json:"family"`
	Observations []ExternalObservation `json:"observations"`
}

func BuildExternalObservation(modelID, compatibility, transport, endpoint, stage, outcome, response, notes string, temperature float64) (ExternalObservation, error) {
	if err := validateRealModelID(modelID); err != nil {
		return ExternalObservation{}, err
	}
	if strings.TrimSpace(modelID) == "" {
		return ExternalObservation{}, fmt.Errorf("model id is required")
	}
	profile := BuildModelInteropProfile(modelID, compatibility, transport)
	return ExternalObservation{
		Schema: ExternalObservationSchema,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Stage: strings.ToUpper(strings.TrimSpace(stage)),
		Outcome: strings.ToUpper(strings.TrimSpace(outcome)),
		ModelInterop: profile,
		Endpoint: strings.TrimSpace(endpoint),
		MediaType: "image/png",
		Temperature: temperature,
		ResponseSHA256: hashText(response),
		ResponseText: response,
		Notes: strings.TrimSpace(notes),
	}, nil
}

func RecordExternalObservation(root string, obs ExternalObservation) (string, error) {
	if strings.TrimSpace(root) == "" {
		root = DefaultInteropMemoryRoot()
	}
	modelDir := filepath.Join(root, safeSegment(obs.ModelInterop.Family), modelKey(obs.ModelInterop.ModelID))
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(modelDir, "external-observations.json")
	reg := ExternalObservationRegistry{
		Schema: "tlaloc.model-interop-external-observation-registry.r0",
		ModelID: obs.ModelInterop.ModelID,
		Family: obs.ModelInterop.Family,
	}
	if body, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(body, &reg); err != nil {
			return "", fmt.Errorf("read external interoperability registry: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	reg.Observations = append(reg.Observations, obs)
	body, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
