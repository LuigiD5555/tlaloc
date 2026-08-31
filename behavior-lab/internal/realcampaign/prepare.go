package realcampaign

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tlaloc.local/behaviorlab/internal/closedloop"
)

func Prepare(ctx context.Context, raw Spec) (Prepared, error) {
	spec, err := Normalize(raw)
	if err != nil {
		return Prepared{}, err
	}
	doc, err := Doctor(ctx, spec)
	if err != nil {
		return Prepared{}, err
	}
	spec.Model = doc.SelectedModel
	phaseDir := filepath.Join(spec.OutputDir, strings.ToLower(spec.Phase))
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		return Prepared{}, err
	}
	baseline := filepath.Join(phaseDir, "baseline.png")
	if err := buildBaseline(ctx, doc.TemporalCarrier, spec.Program, baseline); err != nil {
		return Prepared{}, err
	}
	baselineBody, err := os.ReadFile(baseline)
	if err != nil {
		return Prepared{}, err
	}
	if len(baselineBody) != 8192 {
		return Prepared{}, fmt.Errorf("baseline bytes=%d, want 8192", len(baselineBody))
	}
	memoryRoot := filepath.Join(phaseDir, "learning-memory")
	closedOut := filepath.Join(phaseDir, "closed-loop")
	tlalocVersion := detectTlalocVersion()
	cfg := closedloop.Config{
		Schema:                       closedloop.ConfigSchema,
		RunID:                        spec.CampaignID + "-" + strings.ToLower(spec.Phase),
		BenchmarkID:                  "origami-temporal-native-r0",
		OutputDir:                    closedOut,
		MemoryRoot:                   memoryRoot,
		MasterPrompt:                 spec.MasterPrompt,
		OrigamiVersion:               "6.0.0-alpha.15",
		TlalocVersion:                tlalocVersion,
		TrialsPerModel:               spec.TrialsPerModel,
		CandidatesPerGeneration:     spec.CandidatesPerGen,
		MaxGenerations:              spec.MaxGenerations,
		MinIncumbentImprovement:     0.01,
		DiagnosticRetries:           true,
		Conditions:                  append([]string(nil), spec.Conditions...),
		OutcomeMetric:               closedloop.OutcomeNative,
		Models: []closedloop.ModelConfig{{
			Name:             doc.SelectedModel,
			Provider:         "OPENAI_COMPAT",
			BaseURL:          spec.Endpoint,
			Model:            doc.SelectedModel,
			Compatibility:    spec.Compatibility,
			APIKeyEnv:        spec.APIKeyEnv,
			Temperature:      spec.Temperature,
			TimeoutSeconds:   spec.TimeoutSeconds,
			TransportRetries: spec.TransportRetries,
		}},
		Baseline:                    closedloop.SpecimenConfig{ID: "signal-chain-r0", PNG: baseline},
		AutoCandidates:              true,
		CandidateBuilder:            []string{doc.CandidateBuilder},
		AutoCandidateBaseProfileID:  parentProfile,
		AutoCandidatesPerGeneration: 4,
	}
	configPath := filepath.Join(phaseDir, "closed-loop.json")
	if err := writeJSON(configPath, cfg); err != nil {
		return Prepared{}, err
	}
	configSHA, err := fileSHA(configPath)
	if err != nil {
		return Prepared{}, err
	}
	baselineSHA, err := fileSHA(baseline)
	if err != nil {
		return Prepared{}, err
	}
	status := "SMOKE_TRANSPORT_AND_BEHAVIOR_CHECK"
	policy := "REAL_MODEL_SINGLE_TRIAL_ISOLATED_MEMORY_NOT_PROMOTION_ELIGIBLE"
	if spec.Phase == PhaseEvidence {
		status = "REAL_MODEL_REPEATED_EVIDENCE_READY"
		policy = "REAL_MODEL_SINGLE_MODEL_REPEATED_TRIALS_REQUIRES_CROSS_MODEL_CONFIRMATION_FOR_PROMOTION"
	}
	manifest := Manifest{
		Schema:                   ManifestSchema,
		CampaignID:               spec.CampaignID,
		Phase:                    spec.Phase,
		Status:                   status,
		Endpoint:                 spec.Endpoint,
		CompatibilityStrategy:    spec.Compatibility,
		ModelID:                  doc.SelectedModel,
		ModelInterop:             doc.ModelInterop,
		WorkingConfigurationPath: doc.WorkingConfigurationPath,
		TlalocVersion:            tlalocVersion,
		OrigamiExpectedVersion:   "6.0.0-alpha.15",
		ProgramPath:              spec.Program,
		ProgramSHA256:            doc.ProgramSHA256,
		BaselinePNG:              baseline,
		BaselineSHA256:           baselineSHA,
		BaselineBytes:            len(baselineBody),
		TemporalCarrier:          doc.TemporalCarrier,
		TemporalCarrierSHA256:    doc.TemporalCarrierSHA256,
		CandidateBuilder:         doc.CandidateBuilder,
		CandidateBuilderSHA256:   doc.CandidateBuilderSHA256,
		BuilderCapabilities:      doc.BuilderCapabilities,
		ClosedLoopConfig:         configPath,
		ClosedLoopConfigSHA256:   configSHA,
		MemoryRoot:               memoryRoot,
		EvidencePolicy:           policy,
		PromotionEligible:        false,
		CrossModelEvidence:       false,
	}
	manifestPath := filepath.Join(phaseDir, "manifest.json")
	if err := writeJSON(manifestPath, manifest); err != nil {
		return Prepared{}, err
	}
	return Prepared{Spec: spec, Doctor: doc, Manifest: manifest, ManifestPath: manifestPath, ConfigPath: configPath}, nil
}

func Run(ctx context.Context, raw Spec) (Prepared, closedloop.Report, error) {
	prepared, err := Prepare(ctx, raw)
	if err != nil {
		return Prepared{}, closedloop.Report{}, err
	}
	var cfg closedloop.Config
	if err := readJSON(prepared.ConfigPath, &cfg); err != nil {
		return prepared, closedloop.Report{}, err
	}
	if err := closedloop.ValidateAutoReady(ctx, cfg); err != nil {
		return prepared, closedloop.Report{}, err
	}
	report, err := closedloop.RunAuto(ctx, cfg)
	if err != nil {
		return prepared, report, err
	}
	working := BuildWorkingConfiguration(prepared.Spec, prepared.Doctor.ModelInterop, "CAMPAIGN_RUN", prepared.Doctor.ProgramSHA256, prepared.Manifest.BaselineSHA256, "")
	if len(report.Generations) > 0 && len(working.Evidence) > 0 {
		last := report.Generations[len(report.Generations)-1].Baseline.Scores
		working.Evidence[0].MeanNativeScore = last.MeanNative
		working.Evidence[0].MeanOverallScore = last.MeanOverall
		working.Evidence[0].ExecutionErrors = len(report.ExecutionErrors)
	}
	if _, recErr := RecordWorkingConfiguration(prepared.Spec.InteropMemoryRoot, working); recErr != nil {
		return prepared, report, fmt.Errorf("record campaign working configuration: %w", recErr)
	}
	return prepared, report, nil
}

func detectTlalocVersion() string {
	if v := strings.TrimSpace(os.Getenv("TLALOC_VERSION")); v != "" {
		return v
	}
	candidates := []string{"VERSION", "../VERSION"}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append([]string{filepath.Join(dir, "..", "VERSION")}, candidates...)
	}
	for _, path := range candidates {
		if body, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(body)) != "" {
			return strings.TrimSpace(string(body)), nil
		}
	}
	return "UNKNOWN"
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func readJSON(path string, value any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(value)
}
