package realcampaign

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const WorkingConfigurationSchema = "tlaloc.model-working-configuration.r0"

type WorkingConfiguration struct {
	Schema              string              `json:"schema"`
	RecordedAt          string              `json:"recorded_at"`
	Stage               string              `json:"stage"`
	Outcome             string              `json:"outcome"`
	ModelInterop        ModelInteropProfile `json:"model_interop"`
	Endpoint            string              `json:"endpoint"`
	MediaType           string              `json:"media_type"`
	Temperature         float64             `json:"temperature"`
	TimeoutSeconds      int                 `json:"timeout_seconds"`
	TransportRetries    int                 `json:"transport_retries"`
	Conditions          []string            `json:"conditions,omitempty"`
	MasterPromptSHA256  string              `json:"master_prompt_sha256,omitempty"`
	ProgramSHA256       string              `json:"program_sha256,omitempty"`
	CarrierSHA256       string              `json:"carrier_sha256,omitempty"`
	ProbeResponseSHA256 string              `json:"probe_response_sha256,omitempty"`
}

type WorkingConfigurationRegistry struct {
	Schema         string                 `json:"schema"`
	ModelID        string                 `json:"model_id"`
	Family         string                 `json:"family"`
	Configurations []WorkingConfiguration `json:"configurations"`
}

func DefaultInteropMemoryRoot() string {
	if root := strings.TrimSpace(os.Getenv("TLALOC_INTEROP_MEMORY")); root != "" {
		return root
	}
	if state := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); state != "" {
		return filepath.Join(state, "tlaloc", "model-interop")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".local", "state", "tlaloc", "model-interop")
	}
	return filepath.Join("runs", "model-interop")
}

func BuildWorkingConfiguration(spec Spec, profile ModelInteropProfile, stage, programSHA, carrierSHA, probeResponse string) WorkingConfiguration {
	return WorkingConfiguration{
		Schema: WorkingConfigurationSchema,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Stage: strings.ToUpper(strings.TrimSpace(stage)),
		Outcome: "PASS",
		ModelInterop: profile,
		Endpoint: spec.Endpoint,
		MediaType: "image/png",
		Temperature: spec.Temperature,
		TimeoutSeconds: spec.TimeoutSeconds,
		TransportRetries: spec.TransportRetries,
		Conditions: append([]string(nil), spec.Conditions...),
		MasterPromptSHA256: hashText(spec.MasterPrompt),
		ProgramSHA256: programSHA,
		CarrierSHA256: carrierSHA,
		ProbeResponseSHA256: hashText(probeResponse),
	}
}

func RecordWorkingConfiguration(root string, cfg WorkingConfiguration) (string, error) {
	if strings.TrimSpace(root) == "" {
		root = DefaultInteropMemoryRoot()
	}
	modelDir := filepath.Join(root, safeSegment(cfg.ModelInterop.Family), modelKey(cfg.ModelInterop.ModelID))
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(modelDir, "working-configurations.json")
	reg := WorkingConfigurationRegistry{
		Schema: "tlaloc.model-working-configuration-registry.r0",
		ModelID: cfg.ModelInterop.ModelID,
		Family: cfg.ModelInterop.Family,
	}
	if body, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(body, &reg); err != nil {
			return "", fmt.Errorf("read interoperability registry: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	fingerprint := workingConfigurationFingerprint(cfg)
	for _, existing := range reg.Configurations {
		if workingConfigurationFingerprint(existing) == fingerprint {
			return path, nil
		}
	}
	reg.Configurations = append(reg.Configurations, cfg)
	sort.SliceStable(reg.Configurations, func(i, j int) bool {
		return reg.Configurations[i].RecordedAt < reg.Configurations[j].RecordedAt
	})
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

func workingConfigurationFingerprint(cfg WorkingConfiguration) string {
	stable := struct {
		Stage string `json:"stage"`
		ModelInterop ModelInteropProfile `json:"model_interop"`
		Endpoint string `json:"endpoint"`
		MediaType string `json:"media_type"`
		Temperature float64 `json:"temperature"`
		TimeoutSeconds int `json:"timeout_seconds"`
		TransportRetries int `json:"transport_retries"`
		Conditions []string `json:"conditions"`
		MasterPromptSHA256 string `json:"master_prompt_sha256"`
	}{
		Stage: cfg.Stage,
		ModelInterop: cfg.ModelInterop,
		Endpoint: cfg.Endpoint,
		MediaType: cfg.MediaType,
		Temperature: cfg.Temperature,
		TimeoutSeconds: cfg.TimeoutSeconds,
		TransportRetries: cfg.TransportRetries,
		Conditions: cfg.Conditions,
		MasterPromptSHA256: cfg.MasterPromptSHA256,
	}
	body, _ := json.Marshal(stable)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func modelKey(modelID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(modelID)))
	return safeSegment(modelID) + "-" + hex.EncodeToString(sum[:6])
}

func safeSegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

func hashText(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
