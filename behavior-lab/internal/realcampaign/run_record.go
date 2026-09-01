package realcampaign

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"tlaloc.local/behaviorlab/internal/runrecord"
)

func emitPrepareRunRecord(prepared Prepared, requestedModel string, startedAt, completedAt time.Time) (string, error) {
	outputHash, err := runrecord.HashOutput(prepared)
	if err != nil {
		return "", err
	}
	promptHash, err := optionalFileHash(prepared.Spec.MasterPrompt)
	if err != nil {
		return "", err
	}
	replay, err := runrecord.EncodeReplay(prepareReplayArguments(prepared.Spec))
	if err != nil {
		return "", err
	}
	if requestedModel == "" {
		requestedModel = "auto"
	}
	record := runrecord.Record{
		Schema:       runrecord.Schema,
		VariableAxis: "model.id_requested",
		Component: runrecord.Component{
			Tlaloc:    prepared.Manifest.TlalocVersion,
			Origami:   prepared.Manifest.OrigamiExpectedVersion,
			TonalLock: environmentOrDefault("TONAL_LOCK_VERSION", "unknown"),
		},
		Model: runrecord.Model{
			Provider: "openai-compatible", IDRequested: requestedModel,
			IDReported: prepared.Doctor.SelectedModel, Quantization: "unknown",
			ContextWindow: 0, Tokenizer: "unknown", Endpoint: prepared.Spec.Endpoint,
			ObservedAt: startedAt.Format(time.RFC3339),
		},
		Sampling: runrecord.Sampling{
			Temperature: prepared.Spec.Temperature, TopP: 1, Seed: 0,
			MaxTokens: prepared.Spec.MaxOutputTokens, Stop: []string{},
		},
		Prompt: runrecord.Prompt{
			BehaviorSpecID:     prepared.Spec.CampaignID,
			PromptIRHash:       "sha256:" + promptHash,
			CompiledPromptHash: "sha256:" + prepared.Manifest.ClosedLoopConfigSHA256,
		},
		Fixture: runrecord.Fixture{ID: prepared.Spec.Program, SHA256: "sha256:" + prepared.Manifest.ProgramSHA256},
		Host:    runrecord.CaptureHost(),
		Outcome: runrecord.Outcome{
			OutputHash: outputHash, Parsed: true, Verdict: "verify_pass",
			LatencyMS: completedAt.Sub(startedAt).Milliseconds(),
		},
		Repetitions: runrecord.Repetitions{N: 1, VerdictDistribution: map[string]int{"verify_pass": 1}},
		Replay:      replay,
		Trace: []runrecord.TransitionEvent{{
			Sequence: 0,
			From: "PENDING",
			To: "PROMPT_COMPILED",
			At: completedAt.Format(time.RFC3339Nano),
			LatencyMS: completedAt.Sub(startedAt).Milliseconds(),
			Actor: "realcampaign:prepare",
		}},
	}
	finalized, err := runrecord.Finalize(record, startedAt)
	if err != nil {
		return "", err
	}
	return runrecord.Store(prepared.Spec.RunRecordRoot, finalized)
}

func prepareReplayArguments(spec Spec) []string {
	executable := environmentOrDefault("TLALOC_RUN_RECORD_REPLAY_EXECUTABLE", "tlaloc-real-vlm-campaign")
	arguments := []string{
		executable, "prepare",
		"-id", spec.CampaignID,
		"-phase", spec.Phase,
		"-endpoint", spec.Endpoint,
		"-model", spec.Model,
		"-compatibility", spec.Compatibility,
		"-transport-condition", spec.TransportCondition,
		"-max-output-tokens", strconv.Itoa(spec.MaxOutputTokens),
		"-generation-guard", spec.GenerationGuard,
		"-program", spec.Program,
		"-carrier", spec.TemporalCarrier,
		"-builder", spec.CandidateBuilder,
		"-out", spec.OutputDir,
		"-temperature", strconv.FormatFloat(spec.Temperature, 'g', -1, 64),
		"-timeout", strconv.Itoa(spec.TimeoutSeconds),
		"-transport-retries", strconv.Itoa(spec.TransportRetries),
		"-trials", strconv.Itoa(spec.TrialsPerModel),
		"-candidates-per-generation", strconv.Itoa(spec.CandidatesPerGen),
		"-generations", strconv.Itoa(spec.MaxGenerations),
		"-run-record-root=",
	}
	if spec.TraceStream {
		arguments = append(arguments, "-trace-stream")
	}
	if spec.InteropMemoryRoot != "" {
		arguments = append(arguments, "-interop-memory", spec.InteropMemoryRoot)
	}
	if spec.APIKeyEnv != "" {
		arguments = append(arguments, "-api-key-env", spec.APIKeyEnv)
	}
	if spec.MasterPrompt != "" {
		arguments = append(arguments, "-master-prompt", spec.MasterPrompt)
	}
	return arguments
}

func optionalFileHash(path string) (string, error) {
	if path == "" {
		return runrecord.HashBytes(nil)[len("sha256:"):], nil
	}
	hash, err := fileSHA(path)
	if err != nil {
		return "", fmt.Errorf("hash master prompt: %w", err)
	}
	return hash, nil
}

func environmentOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
