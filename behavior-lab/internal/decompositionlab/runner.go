package decompositionlab

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// RunnerConfig is the shared, cross-record configuration for one T0 run.
// Nothing here is record-specific; it is built once per campaign/model.
type RunnerConfig struct {
	Registry    *tlaloque.Registry
	Store       blackboard.Store
	Endpoint    exocortex.ParrotEndpoint
	MarginRatio float64 // crop expansion margin (section 12 "crop expansion")
	CropDir     string  // where CROP_REGION writes its output PNGs
	StoreDir    string  // pdfmemory store root, required only for REAL locate (T0-B)

	// C0Baseline holds the imported frozen P0 direct-Parrot outcomes keyed
	// by base_id (BLOCKER 1). It is the ONLY source for C0 rows: the C0
	// condition never constructs a ModelAdapter or makes a model call.
	C0Baseline map[string]P0Outcome
}

// NewRegistry builds the Registry for one T0 run: the five R0 Tlaloques,
// wired through a CapabilityRouter that vetoes Parrot for any opcode the
// profile marks EXTERNALIZE/DO_NOT_DEPLOY (E4). This is the only place a
// Registry is constructed for T0 — reused across every record/condition.
func NewRegistry(profile exocortex.CapabilityProfile, endpoint exocortex.ParrotEndpoint) (*tlaloque.Registry, error) {
	registry := tlaloque.NewRegistry()
	for _, w := range []tlaloque.CapabilityWorker{
		exocortex.RegionLocateTlaloque{}, exocortex.RegionCropTlaloque{},
		exocortex.NumericTlaloque{}, exocortex.NormalizeTlaloque{}, exocortex.VerifyTlaloque{},
	} {
		if err := registry.Register(w); err != nil {
			return nil, err
		}
	}
	profiles := map[string]exocortex.CapabilityProfile{}
	for _, w := range exocortex.NewParrotTlaloques(profile, endpoint) {
		if err := registry.Register(w); err != nil {
			return nil, err
		}
		profiles[w.Descriptor().ID] = profile
	}
	registry.SetSelectionStrategy(exocortex.CapabilityRouter{Profiles: profiles})
	return registry, nil
}

// RecordOutcome is one raw T0 record: one P0 record under one condition.
// Every field here is either produced by an actual pipeline execution or
// left at its zero value — nothing is synthesized.
type RecordOutcome struct {
	BaseID    string    `json:"base_id"`
	Condition Condition `json:"condition"`
	Category  string    `json:"category"`
	RunID     string    `json:"run_id"`

	Attempted            bool    `json:"attempted"`
	ContractSuccess      bool    `json:"contract_success"`
	SemanticCorrect      bool    `json:"semantic_correct"`
	Abstained            bool    `json:"abstained"`
	UnsupportedAssertion bool    `json:"unsupported_assertion"`
	FormatFailure        bool    `json:"format_failure"`
	VisualExposureRatio  float64 `json:"visual_exposure_ratio"`
	ParrotCalls          int     `json:"parrot_calls"`
	DeterministicOps     int     `json:"deterministic_ops"`
	LatencyMS            int64   `json:"latency_ms"`
	RawText              string  `json:"raw_text,omitempty"`
	FinalValue           string  `json:"final_value,omitempty"`
	Error                string  `json:"error,omitempty"`
}

// RunRecord executes one P0 record under one Condition end-to-end and
// scores it against ExpectedAnswer only after the pipeline has finished —
// ExpectedAnswer never appears on any path an executor can read (section
// 17's oracle rule, generalized to every condition).
//
// Each Step runs as its own single-node SwarmRunner.Run call against one
// shared BlackboardRuntime/RunID: a fixed, bounded sequencer (E0.13, E0.14)
// that still reuses SwarmRunner/Registry/BlackboardRuntime for every actual
// step execution, rather than introducing a second scheduler.
func RunRecord(ctx context.Context, cfg RunnerConfig, record P0Record, condition Condition) RecordOutcome {
	start := time.Now()
	outcome := RecordOutcome{BaseID: record.BaseID, Condition: condition, Category: record.Category}

	// BLOCKER 1: C0 is the imported frozen P0 direct-Parrot baseline. It
	// never touches the ModelAdapter, the Registry, or the endpoint.
	if condition == ConditionC0ParrotDirect {
		return c0FromBaseline(outcome, cfg, record, start)
	}

	runID := fmt.Sprintf("t0-%s-%s-%d", strings.ToLower(string(condition)), record.BaseID, time.Now().UnixNano())
	outcome.RunID = runID
	bb := &tlaloque.BlackboardRuntime{Store: cfg.Store, RunID: runID}
	runner := tlaloque.SwarmRunner{Registry: cfg.Registry, Blackboard: bb}

	runStep := func(stepID, capability string, input any) (json.RawMessage, error) {
		step, err := exocortex.Step{ID: stepID, Opcode: capability, Output: exocortex.Address(stepID)}.Normalize()
		if err != nil {
			return nil, err
		}
		plan, err := exocortex.StepsToPlan(stepID+"-plan", []exocortex.Step{step}, 1)
		if err != nil {
			return nil, err
		}
		body, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		report, err := runner.Run(ctx, plan, runID, body)
		if err != nil {
			return nil, err
		}
		return report.TerminalOutputs[stepID], nil
	}

	imagePath := record.PageImagePath
	visualField := exocortex.VisualFieldFullPage
	outcome.VisualExposureRatio = 1.0

	if condition.UsesLocateRegion() {
		outcome.Attempted = true
		locateInput := exocortex.RegionLocateInput{}
		if condition.IsOracle() {
			locateInput = exocortex.RegionLocateInput{
				Mode: exocortex.LocateModeOracle, OracleAddress: record.EvidenceAddress, OracleDocID: record.DocID,
				OraclePage: record.Page, OracleBBox: record.EvidenceBBox, OraclePageW: record.PageWidth, OraclePageH: record.PageHeight,
			}
		} else {
			locateInput = exocortex.RegionLocateInput{Mode: exocortex.LocateModeReal, Question: record.Question, StoreDir: cfg.StoreDir}
		}
		locateOut, err := runStep("locate", exocortex.OpLocateRegion, locateInput)
		if err != nil {
			outcome.Error = err.Error()
			return finish(outcome, start)
		}
		var located exocortex.RegionLocateResult
		if err := json.Unmarshal(locateOut, &located); err != nil {
			outcome.Error = fmt.Sprintf("decode locate output: %v", err)
			return finish(outcome, start)
		}
		outcome.DeterministicOps++

		pageW, pageH := located.PageWidth, located.PageHeight
		if pageW == 0 || pageH == 0 {
			pageW, pageH = record.PageWidth, record.PageHeight
		}
		cropPath := filepath.Join(cfg.CropDir, fmt.Sprintf("%s-%s.png", record.BaseID, strings.ToLower(string(condition))))
		cropInput := exocortex.RegionCropInput{
			PageImagePath: record.PageImagePath, PageWidth: pageW, PageHeight: pageH,
			BBox: located.BBox, MarginRatio: cfg.MarginRatio, OutputPath: cropPath,
		}
		cropOut, err := runStep("crop", exocortex.OpCropRegion, cropInput)
		if err != nil {
			outcome.Error = err.Error()
			return finish(outcome, start)
		}
		var cropped struct {
			CropPath            string  `json:"crop_path"`
			VisualExposureRatio float64 `json:"visual_exposure_ratio"`
		}
		if err := json.Unmarshal(cropOut, &cropped); err != nil {
			outcome.Error = fmt.Sprintf("decode crop output: %v", err)
			return finish(outcome, start)
		}
		outcome.DeterministicOps++
		outcome.VisualExposureRatio = cropped.VisualExposureRatio
		imagePath = cropped.CropPath
		visualField = exocortex.VisualFieldTightCrop
	}

	outcome.Attempted = true
	parrotOut, err := runStep("parrot", record.Opcode, exocortex.ParrotInput{
		ImagePath: imagePath, VisualField: visualField,
		CharCount: record.OperandCharCount, ChoiceWidth: record.OperandChoiceWidth,
	})
	if err != nil {
		outcome.Error = err.Error()
		return finish(outcome, start)
	}
	outcome.ParrotCalls++
	var parrotResult exocortex.ParrotOutput
	if err := json.Unmarshal(parrotOut, &parrotResult); err != nil {
		outcome.Error = fmt.Sprintf("decode parrot output: %v", err)
		return finish(outcome, start)
	}
	outcome.RawText = parrotResult.Text
	outcome.FinalValue = parrotResult.Text

	if condition.UsesNormalize() {
		normOut, err := runStep("normalize", exocortex.OpNormalize, exocortex.NormalizeInput{
			Raw: parrotResult.Text, TargetType: opcodeTargetType(record.Opcode),
		})
		if err != nil {
			outcome.Error = err.Error()
			return finish(outcome, start)
		}
		outcome.DeterministicOps++
		var normalized exocortex.NormalizeOutput
		if err := json.Unmarshal(normOut, &normalized); err != nil {
			outcome.Error = fmt.Sprintf("decode normalize output: %v", err)
			return finish(outcome, start)
		}
		if opcodeTargetType(record.Opcode) == exocortex.TargetTypeNumber {
			if normalized.IsNumber {
				outcome.FinalValue = strconv.FormatFloat(normalized.AsNumber, 'f', -1, 64)
			} else {
				outcome.FinalValue = ""
				outcome.FormatFailure = true
			}
		} else {
			outcome.FinalValue = normalized.Trimmed
		}
	}

	if condition.UsesVerify() {
		verifyOut, err := runStep("verify", exocortex.OpVerify, exocortex.VerifyInput{
			TargetKey: "normalize", FactID: record.BaseID, ExpectedType: opcodeTargetType(record.Opcode),
		})
		if err != nil {
			outcome.Error = err.Error()
			return finish(outcome, start)
		}
		outcome.DeterministicOps++
		var fact blackboard.Fact
		if err := json.Unmarshal(verifyOut, &fact); err != nil {
			outcome.Error = fmt.Sprintf("decode verify output: %v", err)
			return finish(outcome, start)
		}
		if fact.Status != blackboard.FactVerified {
			outcome.UnsupportedAssertion = true
			outcome.Abstained = true
			outcome.FinalValue = ""
		}
	}

	outcome.ContractSuccess = outcome.Error == "" && !outcome.FormatFailure && !outcome.UnsupportedAssertion && strings.TrimSpace(outcome.FinalValue) != ""
	if outcome.ContractSuccess {
		outcome.SemanticCorrect = ScoreSemantic(record.Opcode, outcome.FinalValue, record.ExpectedAnswer)
	}
	return finish(outcome, start)
}

// c0FromBaseline builds the C0 row purely from the imported frozen P0
// outcome for this base_id. Zero model calls; ParrotCalls stays 0 because
// the recorded Parrot call belongs to the frozen P0 experiment, not this
// run.
func c0FromBaseline(outcome RecordOutcome, cfg RunnerConfig, record P0Record, start time.Time) RecordOutcome {
	outcome.RunID = "c0-imported-" + record.BaseID
	base, ok := cfg.C0Baseline[record.BaseID]
	if !ok {
		outcome.Error = "C0 baseline missing for base_id " + record.BaseID + " (run `prepare` / pass --p0-baseline)"
		return finish(outcome, start)
	}
	outcome.Attempted = base.Attempted
	outcome.ContractSuccess = base.ContractSuccess
	outcome.SemanticCorrect = base.SemanticCorrect
	outcome.Abstained = base.Abstained
	outcome.UnsupportedAssertion = base.UnsupportedAssertion
	outcome.FormatFailure = base.FormatFailure
	outcome.VisualExposureRatio = 1.0
	outcome.LatencyMS = base.LatencyMS
	outcome.RawText = base.OriginalOutput
	outcome.FinalValue = base.OriginalOutput
	return outcome
}

func finish(outcome RecordOutcome, start time.Time) RecordOutcome {
	outcome.LatencyMS = time.Since(start).Milliseconds()
	return outcome
}

// opcodeTargetType is the fixed, pre-registered mapping from Micro-ISA
// opcode to the primitive type Normalize/Verify convert toward. It is
// declared once, here, and never adjusted per record or per run.
func opcodeTargetType(opcode string) string {
	switch strings.ToUpper(strings.TrimSpace(opcode)) {
	case exocortex.OpExtractNumber, exocortex.OpCompareNumbers:
		return exocortex.TargetTypeNumber
	case exocortex.OpSelectOne, exocortex.OpSameDifferent:
		return exocortex.TargetTypeChoice
	default:
		return exocortex.TargetTypeText
	}
}

// ScoreSemantic is T0's single, pre-registered scoring rule (section 31:
// "no tuning ... no post-hoc condition changes"). Numeric opcodes compare
// with a small float tolerance; everything else compares case-insensitive
// trimmed text. It is applied only after the pipeline has finished and
// ExpectedAnswer was never available to any executor along the way.
func ScoreSemantic(opcode, got, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if opcodeTargetType(opcode) == exocortex.TargetTypeNumber {
		gv, gerr := strconv.ParseFloat(got, 64)
		wv, werr := strconv.ParseFloat(want, 64)
		if gerr != nil || werr != nil {
			return false
		}
		const tolerance = 1e-6
		diff := gv - wv
		if diff < 0 {
			diff = -diff
		}
		return diff <= tolerance
	}
	return strings.EqualFold(got, want)
}
