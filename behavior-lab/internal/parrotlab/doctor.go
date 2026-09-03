package parrotlab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DoctorReport is a pre-flight check of the model endpoint.
type DoctorReport struct {
	Endpoint       string   `json:"endpoint"`
	Reachable      bool     `json:"reachable"`
	Models         []string `json:"models"`
	RequestedModel string   `json:"requested_model"`
	RequestedFound bool     `json:"requested_found"`
	Notes          []string `json:"notes"`
}

// Doctor queries {endpoint}/models and checks the configured model is served.
func Doctor(ctx context.Context, exp *Experiment) (DoctorReport, error) {
	report := DoctorReport{Endpoint: exp.Model.Endpoint, RequestedModel: exp.Model.ID}
	url := strings.TrimRight(exp.Model.Endpoint, "/") + "/models"

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return report, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		report.Notes = append(report.Notes, fmt.Sprintf("endpoint unreachable: %v", err))
		return report, nil
	}
	defer response.Body.Close()
	report.Reachable = response.StatusCode/100 == 2
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		report.Notes = append(report.Notes, "could not parse /models response")
		return report, nil
	}
	for _, model := range payload.Data {
		report.Models = append(report.Models, model.ID)
		if model.ID == exp.Model.ID {
			report.RequestedFound = true
		}
	}
	if exp.Model.ID == "" {
		report.Notes = append(report.Notes, "MODEL.json id is empty")
	} else if !report.RequestedFound {
		report.Notes = append(report.Notes, fmt.Sprintf("configured model %q not served by endpoint", exp.Model.ID))
	}
	if exp.Model.Temperature != 0 {
		report.Notes = append(report.Notes, "MODEL.json temperature is not 0 (R0 requires 0)")
	}
	return report, nil
}

// LMStudioModelInfo is the subset of GET {endpoint}/api/v0/models this lab
// records for instrument identity.
type LMStudioModelInfo struct {
	ContextSize  *int   `json:"context_size"`
	MaxContext   *int   `json:"max_context_length"`
	Quantization string `json:"quantization"`
	Architecture string `json:"architecture"`
	Found        bool   `json:"found"`
}

// ProbeLMStudioModel reads {endpoint}/api/v0/models for the configured model.
func ProbeLMStudioModel(ctx context.Context, exp *Experiment) (LMStudioModelInfo, error) {
	info := LMStudioModelInfo{}
	base := strings.TrimSuffix(strings.TrimRight(exp.Model.Endpoint, "/"), "/v1")
	url := base + "/api/v0/models"
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return info, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return info, err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	var payload struct {
		Data []struct {
			ID               string `json:"id"`
			Quantization     string `json:"quantization"`
			Arch             string `json:"arch"`
			MaxContextLength *int   `json:"max_context_length"`
			LoadedContext    *int   `json:"loaded_context_length"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return info, err
	}
	for _, model := range payload.Data {
		if model.ID != exp.Model.ID {
			continue
		}
		info.Found = true
		info.Quantization = model.Quantization
		info.Architecture = model.Arch
		info.MaxContext = model.MaxContextLength
		info.ContextSize = model.LoadedContext
	}
	return info, nil
}

// WriteModelIdentity fills MODEL.json's identity fields from a live probe and
// local GGUF hashes, before the experiment is frozen. Unmeasurable fields are
// left null with an explicit *_measured:false flag rather than invented.
func WriteModelIdentity(ctx context.Context, exp *Experiment) (map[string]any, error) {
	info, probeErr := ProbeLMStudioModel(ctx, exp)
	changed := map[string]any{}

	if info.ContextSize != nil {
		exp.Model.ContextSize = info.ContextSize
		changed["context_size"] = *info.ContextSize
	}
	if info.Quantization != "" {
		exp.Model.Quantization = info.Quantization
		changed["quantization"] = info.Quantization
	}

	hashes := map[string]string{}
	measured := len(exp.Model.ModelFilePaths) > 0
	for _, path := range exp.Model.ModelFilePaths {
		digest, err := sha256File(path)
		if err != nil {
			measured = false
			changed["model_file_hash_error:"+filepath.Base(path)] = err.Error()
			continue
		}
		hashes[filepath.Base(path)] = digest
	}
	if len(hashes) > 0 {
		exp.Model.ModelFileHashes = hashes
	}
	exp.Model.ModelFileHashesMeasured = measured && len(hashes) == len(exp.Model.ModelFilePaths)
	changed["model_file_hashes"] = hashes
	changed["model_file_hashes_measured"] = exp.Model.ModelFileHashesMeasured

	// runtime_version is not exposed by the LM Studio API or a CLI here.
	exp.Model.RuntimeVersion = nil
	exp.Model.RuntimeVersionMeasured = false
	changed["runtime_version"] = nil
	changed["runtime_version_measured"] = false

	raw, err := json.MarshalIndent(exp.Model, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(exp.Root, "MODEL.json"), append(raw, '\n'), 0o644); err != nil {
		return nil, err
	}
	if probeErr != nil {
		changed["probe_error"] = probeErr.Error()
	}
	return changed, nil
}

func sha256File(path string) (string, error) {
	handle, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, handle); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
