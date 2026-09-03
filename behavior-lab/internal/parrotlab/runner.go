package parrotlab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tlaloc.local/behaviorlab/internal/target"
)

// RunRecord is one immutable per-case result line (SPEC §31).
type RunRecord struct {
	RunID          string   `json:"run_id"`
	ExperimentID   string   `json:"experiment_id"`
	Stage          string   `json:"stage"`
	CaseID         string   `json:"case_id"`
	Repetition     int      `json:"repetition"`
	Model          RunModel `json:"model"`
	Capabilities   []string `json:"capabilities"`
	Operations     int      `json:"operations,omitempty"`
	BaseID         string   `json:"base_id,omitempty"`
	Sentinel       bool     `json:"sentinel,omitempty"`
	TaskFamily     string   `json:"task_family"`
	AddedPrimitive string   `json:"added_primitive,omitempty"`
	PriorContract  bool     `json:"prior_contract,omitempty"`
	HintCondition  string   `json:"hint_condition,omitempty"`
	Variant        string   `json:"variant,omitempty"`
	EvidenceCID    string   `json:"evidence_cid,omitempty"`
	SourceMethod   string   `json:"source_method,omitempty"`

	// Input audit (P-1 fix #8): everything Parrot actually received, so
	// no-leakage can be demonstrated after the fact from the record alone.
	PromptFileHash   string `json:"prompt_file_hash"`
	SystemPromptHash string `json:"system_prompt_hash"`
	UserText         string `json:"user_text"`
	UserTextHash     string `json:"user_text_hash"`
	ImageHash        string `json:"image_hash,omitempty"`

	Expected  Expected  `json:"expected"`
	Actual    Actual    `json:"actual"`
	Score     Score     `json:"score"`
	Resources Resources `json:"resources"`
	Error     string    `json:"error,omitempty"`
	Timestamp string    `json:"timestamp"`
}

type RunModel struct {
	ID          string  `json:"id"`
	Temperature float64 `json:"temperature"`
}

type Actual struct {
	Raw    string `json:"raw"`
	Parsed string `json:"parsed"`
}

// Resources is the per-call cost picture. Only wall time and provider-
// reported tokens are genuinely measurable here; CPU/RAM/VRAM live in a
// separate LM Studio process this cannot observe, so they are null with an
// explicit *_measured:false rather than a misleading 0 (P-1 fix #7).
// Instrumenting them for real belongs to Phase S.
type Resources struct {
	WallMS       int64  `json:"wall_ms"`
	TokensIn     int    `json:"tokens_in"`
	TokensOut    int    `json:"tokens_out"`
	CPUMS        *int64 `json:"cpu_ms"`
	RAMPeakMB    *int64 `json:"ram_peak_mb"`
	VRAMPeakMB   *int64 `json:"vram_peak_mb"`
	CPUMeasured  bool   `json:"cpu_measured"`
	RAMMeasured  bool   `json:"ram_measured"`
	VRAMMeasured bool   `json:"vram_measured"`
}

// RunOptions controls a single stage run.
type RunOptions struct {
	Stage          string
	DatasetPath    string // optional override; defaults to the stage's manifest dataset
	Repetitions    int
	SentinelOnly   bool
	TimeoutSeconds int
	// AllowUnfrozen skips the stage-freeze gate — for ad-hoc smoke runs
	// only. A real campaign run requires the global config and the stage
	// dataset to be frozen (P-1 fix #6).
	AllowUnfrozen bool
}

// RunReport summarises what a stage run produced.
type RunReport struct {
	Stage      string `json:"stage"`
	Cases      int    `json:"cases"`
	Records    int    `json:"records"`
	Errors     int    `json:"errors"`
	OutputFile string `json:"output_file"`
}

// RunStage executes one stage and appends RunRecords to
// runs/<stage>/<stage>.jsonl.
func RunStage(ctx context.Context, exp *Experiment, opts RunOptions) (RunReport, error) {
	if opts.Repetitions < 1 {
		opts.Repetitions = 1
	}
	datasetPath := opts.DatasetPath
	if datasetPath == "" {
		var err error
		if datasetPath, err = exp.StageDataset(opts.Stage); err != nil {
			return RunReport{}, err
		}
	}
	cases, err := LoadCases(datasetPath)
	if err != nil {
		return RunReport{}, err
	}
	if problems := Validate(cases); len(problems) > 0 {
		return RunReport{}, fmt.Errorf("dataset invalid: %d problem(s); first: %v", len(problems), problems[0])
	}

	if !opts.AllowUnfrozen && opts.DatasetPath == "" {
		if err := requireStageFrozen(exp, opts.Stage); err != nil {
			return RunReport{}, err
		}
	}

	compatibility, err := target.ResolveMultimodalCompatibility(exp.Model.Compatibility)
	if err != nil {
		return RunReport{}, err
	}
	client := target.OpenAICompat{
		BaseURL:        exp.Model.Endpoint,
		Model:          exp.Model.ID,
		Temperature:    exp.Model.Temperature,
		MaxTokens:      exp.Model.MaxTokens,
		Compatibility:  compatibility,
		RequestTimeout: time.Duration(max(opts.TimeoutSeconds, 60)) * time.Second,
	}

	outDir := filepath.Join(exp.Root, "runs", opts.Stage)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return RunReport{}, err
	}
	outFile := filepath.Join(outDir, opts.Stage+".jsonl")
	handle, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return RunReport{}, err
	}
	defer handle.Close()
	encoder := json.NewEncoder(handle)

	promptFileHash := exp.PromptHash()
	systemPromptHash := hashString(exp.Prompt.System)
	report := RunReport{Stage: opts.Stage, OutputFile: outFile}
	stamp := time.Now().UTC().Format("20060102T150405Z")

	for _, item := range cases {
		if opts.SentinelOnly && !item.Sentinel {
			continue
		}
		report.Cases++
		image, imageErr := item.ImageBytes()
		if imageErr != nil {
			return report, fmt.Errorf("case %s: %w", item.CaseID, imageErr)
		}
		userText := item.UserText()
		for repetition := 1; repetition <= opts.Repetitions; repetition++ {
			if ctx.Err() != nil {
				return report, ctx.Err()
			}
			record := RunRecord{
				RunID:            fmt.Sprintf("%s-%s-r%d-%s", exp.Manifest.ExperimentID, item.CaseID, repetition, stamp),
				ExperimentID:     exp.Manifest.ExperimentID,
				Stage:            opts.Stage,
				CaseID:           item.CaseID,
				Repetition:       repetition,
				Model:            RunModel{ID: exp.Model.ID, Temperature: exp.Model.Temperature},
				Capabilities:     item.Capabilities,
				Operations:       item.Operations,
				BaseID:           item.BaseID,
				Sentinel:         item.Sentinel,
				TaskFamily:       item.TaskFamily,
				AddedPrimitive:   item.AddedPrimitive,
				PriorContract:    item.PriorContract,
				HintCondition:    item.HintCondition,
				Variant:          item.Variant,
				EvidenceCID:      item.EvidenceCID,
				SourceMethod:     item.SourceMethod,
				PromptFileHash:   promptFileHash,
				SystemPromptHash: systemPromptHash,
				UserText:         userText,
				UserTextHash:     hashString(userText),
				Expected:         item.Expected,
				Timestamp:        time.Now().UTC().Format(time.RFC3339),
			}
			if len(image) > 0 {
				record.ImageHash = hashBytes(image)
			}

			started := time.Now()
			raw, callErr := invoke(ctx, client, exp.Prompt.System, userText, image, &record.Resources)
			record.Resources.WallMS = time.Since(started).Milliseconds()

			if callErr != nil {
				record.Error = callErr.Error()
				report.Errors++
			} else {
				record.Actual = Actual{Raw: raw}
				record.Score = ScoreAnswer(item, raw)
				record.Actual.Parsed = record.Score.Parsed
			}
			if err := encoder.Encode(record); err != nil {
				return report, err
			}
			report.Records++
		}
	}
	return report, nil
}

func invoke(ctx context.Context, client target.OpenAICompat, system, user string, image []byte, res *Resources) (string, error) {
	if len(image) == 0 {
		result, err := client.CompleteText(ctx, system, user)
		if err != nil {
			return "", err
		}
		res.TokensIn = result.PromptTokensReported
		res.TokensOut = result.CompletionTokensReported
		return result.Content, nil
	}
	result, err := client.CompletePerception(ctx, target.PerceptionInput{
		SystemPrompt: system,
		Question:     user,
		Image:        image,
		MediaType:    "image/png",
	})
	if err != nil {
		return "", err
	}
	res.TokensIn = result.PromptTokensReported
	res.TokensOut = result.CompletionTokensReported
	return result.Content, nil
}
