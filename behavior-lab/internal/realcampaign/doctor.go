package realcampaign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/temporalbench"
)

func Doctor(ctx context.Context, raw Spec) (DoctorResult, error) {
	spec, err := Normalize(raw)
	if err != nil {
		return DoctorResult{}, err
	}
	carrier, err := exec.LookPath(spec.TemporalCarrier)
	if err != nil {
		return DoctorResult{}, fmt.Errorf("temporal carrier: %w", err)
	}
	builder, err := exec.LookPath(spec.CandidateBuilder)
	if err != nil {
		return DoctorResult{}, fmt.Errorf("candidate builder: %w", err)
	}
	caps, err := queryBuilderCapabilities(ctx, builder)
	if err != nil {
		return DoctorResult{}, err
	}
	if caps.Schema != "origami.experimental-candidate.r0.capabilities" {
		return DoctorResult{}, fmt.Errorf("unexpected builder capability schema %q", caps.Schema)
	}
	if caps.ExactPlaneMutation {
		return DoctorResult{}, fmt.Errorf("candidate builder declares exact-plane mutation")
	}
	if !containsFold(caps.ParentProfiles, parentProfile) {
		return DoctorResult{}, fmt.Errorf("candidate builder does not support parent profile %s", parentProfile)
	}
	models, err := discoverModels(ctx, spec.Endpoint, apiKey(spec.APIKeyEnv))
	if err != nil {
		return DoctorResult{}, err
	}
	selected, err := selectModel(spec.Model, models)
	if err != nil {
		return DoctorResult{}, err
	}
	if err := validateRealModelID(selected); err != nil {
		return DoctorResult{}, err
	}
	compatibility, err := target.ResolveMultimodalCompatibility(spec.Compatibility)
	if err != nil {
		return DoctorResult{}, err
	}
	guard, err := target.ResolveGenerationGuard(spec.GenerationGuard)
	if err != nil {
		return DoctorResult{}, err
	}
	modelInterop := BuildModelInteropProfile(selected, compatibility.Name(), spec.TransportCondition)

	tmp, err := os.MkdirTemp("", "tlaloc-real-vlm-doctor-*")
	if err != nil {
		return DoctorResult{}, err
	}
	defer os.RemoveAll(tmp)
	baseline := filepath.Join(tmp, "baseline.png")
	if err := buildBaseline(ctx, carrier, spec.Program, baseline); err != nil {
		return DoctorResult{}, err
	}
	image, err := os.ReadFile(baseline)
	if err != nil {
		return DoctorResult{}, err
	}
	if err := validateCarrierBytes(image); err != nil {
		return DoctorResult{}, err
	}
	questions := temporalbench.CanonicalQuestions()
	if len(questions) == 0 {
		return DoctorResult{}, fmt.Errorf("temporal benchmark has no questions")
	}
	client := target.OpenAICompat{
		BaseURL:       spec.Endpoint,
		Model:         selected,
		Temperature:   spec.Temperature,
		APIKey:        apiKey(spec.APIKeyEnv),
		Compatibility: compatibility,
		MaxTokens:     spec.MaxOutputTokens,
		Guard:         guard,
	}
	if spec.TraceStream {
		client.Observer = target.NewWriterTraceObserver(os.Stderr)
	}
	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(spec.TimeoutSeconds)*time.Second)
	defer cancel()
	probe, err := client.CompletePerception(probeCtx, target.PerceptionInput{Question: questions[0].Text, Image: image, MediaType: "image/png"})
	if err != nil {
		return DoctorResult{}, fmt.Errorf("multimodal probe failed: %w", err)
	}
	carrierSHA, err := fileSHA(carrier)
	if err != nil {
		return DoctorResult{}, err
	}
	builderSHA, err := fileSHA(builder)
	if err != nil {
		return DoctorResult{}, err
	}
	programSHA, err := fileSHA(spec.Program)
	if err != nil {
		return DoctorResult{}, err
	}
	working := BuildWorkingConfiguration(spec, modelInterop, "DOCTOR_TRANSPORT", programSHA, hashBytes(image), probe.Content)
	workingPath, err := RecordWorkingConfiguration(spec.InteropMemoryRoot, working)
	if err != nil {
		return DoctorResult{}, fmt.Errorf("record working configuration: %w", err)
	}
	return DoctorResult{
		Schema:                   SpecSchema + ".doctor",
		Endpoint:                 spec.Endpoint,
		CompatibilityStrategy:    compatibility.Name(),
		TraceStream:              spec.TraceStream,
		MaxOutputTokens:          spec.MaxOutputTokens,
		GenerationGuard:          spec.GenerationGuard,
		ModelInterop:             modelInterop,
		WorkingConfigurationPath: workingPath,
		DiscoveredModels:         models,
		SelectedModel:            selected,
		VisionTransport:          true,
		ProbeResponse:            probe.Content,
		TemporalCarrier:          carrier,
		TemporalCarrierSHA256:    carrierSHA,
		CandidateBuilder:         builder,
		CandidateBuilderSHA256:   builderSHA,
		BuilderCapabilities:      caps,
		ProgramSHA256:            programSHA,
		ParentProfile:            parentProfile,
		Ready:                    true,
	}, nil
}

func validateCarrierBytes(image []byte) error {
	if len(image) != 8192 {
		return fmt.Errorf("expected frozen 8192-byte temporal carrier, got %d", len(image))
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(image))
	if err != nil {
		return fmt.Errorf("temporal carrier is not a valid PNG: %w", err)
	}
	if cfg.Width != 640 || cfg.Height != 640 {
		return fmt.Errorf("expected 640x640 temporal carrier, got %dx%d", cfg.Width, cfg.Height)
	}
	return nil
}

func discoverModels(ctx context.Context, base, key string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model discovery: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("model discovery status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("model discovery JSON: %w", err)
	}
	models := []string{}
	for _, m := range payload.Data {
		if strings.TrimSpace(m.ID) != "" {
			models = append(models, m.ID)
		}
	}
	sort.Strings(models)
	if len(models) == 0 {
		return nil, fmt.Errorf("endpoint returned no models")
	}
	return models, nil
}

func selectModel(requested string, models []string) (string, error) {
	if requested != "" {
		for _, m := range models {
			if m == requested {
				return m, nil
			}
		}
		return "", fmt.Errorf("requested model %q not reported by endpoint; available: %s", requested, strings.Join(models, ", "))
	}
	if len(models) == 1 {
		return models[0], nil
	}
	return "", fmt.Errorf("multiple models are available; select one explicitly: %s", strings.Join(models, ", "))
}

func queryBuilderCapabilities(ctx context.Context, builder string) (BuilderCapabilities, error) {
	cmd := exec.CommandContext(ctx, builder, "capabilities")
	body, err := cmd.CombinedOutput()
	if err != nil {
		return BuilderCapabilities{}, fmt.Errorf("candidate builder capabilities: %v: %s", err, strings.TrimSpace(string(body)))
	}
	var caps BuilderCapabilities
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&caps); err != nil {
		return BuilderCapabilities{}, err
	}
	return caps, nil
}

func buildBaseline(ctx context.Context, carrier, program, out string) error {
	cmd := exec.CommandContext(ctx, carrier, "-mode", "build", "-in", program, "-out", out)
	body, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build baseline: %v: %s", err, strings.TrimSpace(string(body)))
	}
	return nil
}

func apiKey(env string) string {
	if strings.TrimSpace(env) == "" {
		return ""
	}
	return os.Getenv(env)
}

func containsFold(in []string, want string) bool {
	for _, x := range in {
		if strings.EqualFold(strings.TrimSpace(x), want) {
			return true
		}
	}
	return false
}
