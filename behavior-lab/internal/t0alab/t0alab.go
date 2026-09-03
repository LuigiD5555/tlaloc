// Package t0alab implements T0-A (CONTROLLED EXTERNAL COMPOSITION R0),
// the first direct falsification test of the Exocortex mechanism: holding
// the underlying two-stage task constant, can Tlaloc recover end-to-end
// capability by moving sequencing and working state outside Parrot while
// keeping every Parrot invocation atomic?
//
// It reuses internal/exocortex (Tlaloques, ModelAdapter, CapabilityProfile),
// internal/tlaloque (SwarmRunner, BlackboardRuntime), internal/blackboard,
// internal/decompositionlab (NewRegistry, Wilson/McNemar stats) and
// internal/parrotlab (the frozen T0-A dataset) rather than duplicating any
// of them.
package t0alab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/decompositionlab"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// Condition names the four T0-A conditions (T0 protocol section 10).
type Condition string

const (
	// D0 — DIRECT TWO-OP CONTROL: one Parrot call carries both cognitive
	// operations (read A, read B, compare). Paired control only; never a
	// new Instruction Cliff measurement.
	ConditionD0Direct Condition = "D0_DIRECT_TWO_OP"
	// D1 — Parrot OP1 -> external Blackboard state -> Parrot OP2, each call
	// exactly one opcode; the deterministic COMPARE is the external join.
	ConditionD1ExternalSeq Condition = "D1_EXTERNAL_SEQUENCING"
	// D2 — external deterministic OP1 -> Parrot OP2 -> deterministic COMPARE.
	ConditionD2ExternalOp1 Condition = "D2_EXTERNAL_OP1"
	// D3 — D2 + deterministic Normalize + external Verify (Fact | UNKNOWN).
	ConditionD3Verify Condition = "D3_NORMALIZE_VERIFY"
)

// AllConditions is the fixed report order.
func AllConditions() []Condition {
	return []Condition{ConditionD0Direct, ConditionD1ExternalSeq, ConditionD2ExternalOp1, ConditionD3Verify}
}

// StepTrace instruments one workflow step (T0 protocol section 13). It only
// observes execution; it never contributes prompt context.
type StepTrace struct {
	StepID                   string `json:"step_id"`
	StepIndex                int    `json:"step_index"`
	WorkflowDepth            int    `json:"workflow_depth"`
	Opcode                   string `json:"opcode"`
	ExecutorType             string `json:"executor_type"` // MODEL | DETERMINISTIC
	StateBeforeHash          string `json:"state_before_hash"`
	StateAfterHash           string `json:"state_after_hash"`
	WorkingSetItemCount      int    `json:"working_set_item_count"`
	WorkingSetBytes          int    `json:"working_set_bytes"`
	ModelCalls               int    `json:"model_calls"`
	CognitiveOpsGivenToModel int    `json:"cognitive_ops_given_to_model"`
	DeterministicOps         int    `json:"deterministic_ops"`
	ObservationKey           string `json:"observation_key,omitempty"`
	FactPromoted             bool   `json:"fact_promoted"`
	LatencyMS                int64  `json:"latency_ms"`
	Error                    string `json:"error,omitempty"`
}

// StimulusOutcome is one base stimulus under one condition.
type StimulusOutcome struct {
	ID        string    `json:"id"`
	Condition Condition `json:"condition"`
	RunID     string    `json:"run_id"`

	Attempted            bool        `json:"attempted"`
	ContractSuccess      bool        `json:"contract_success"`
	SemanticCorrect      bool        `json:"semantic_correct"`
	Abstained            bool        `json:"abstained"`
	UnsupportedAssertion bool        `json:"unsupported_assertion"`
	FormatFailure        bool        `json:"format_failure"`
	WorkflowDepth        int         `json:"workflow_depth"`
	ModelCalls           int         `json:"model_calls"`
	DeterministicOps     int         `json:"deterministic_ops"`
	LatencyMS            int64       `json:"latency_ms"`
	FinalAnswer          string      `json:"final_answer,omitempty"`
	Error                string      `json:"error,omitempty"`
	Steps                []StepTrace `json:"steps"`
}

// Config is the shared cross-stimulus configuration for one T0-A run.
type Config struct {
	Profile    exocortex.CapabilityProfile
	Endpoint   exocortex.ParrotEndpoint
	Store      blackboard.Store
	DatasetDir string // where the T0-A images live (parent of images/)
	MaxOutTok  int
}

func hashSnapshot(s *blackboard.Snapshot) string {
	if s == nil {
		return ""
	}
	body, _ := json.Marshal(s.Entries)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])[:16]
}

func parseAB(raw string) string {
	for _, r := range strings.ToUpper(raw) {
		if r == 'A' {
			return "A"
		}
		if r == 'B' {
			return "B"
		}
	}
	return ""
}

// RunStimulus executes one T0-A base stimulus under one condition.
func RunStimulus(ctx context.Context, cfg Config, registry *tlaloque.Registry, record parrotlab.T0ARecord, condition Condition) StimulusOutcome {
	start := time.Now()
	out := StimulusOutcome{ID: record.ID, Condition: condition, Attempted: true}
	out.RunID = fmt.Sprintf("t0a-%s-%s-%d", strings.ToLower(string(condition)), record.ID, time.Now().UnixNano())
	imgPath := func(rel string) string { return filepath.Join(cfg.DatasetDir, filepath.FromSlash(rel)) }

	switch condition {
	case ConditionD0Direct:
		out.WorkflowDepth = 1
		st := StepTrace{StepID: "d0", StepIndex: 0, WorkflowDepth: 1, Opcode: "READ_A+READ_B+COMPARE", ExecutorType: "MODEL", ModelCalls: 1, CognitiveOpsGivenToModel: 2}
		image, err := os.ReadFile(imgPath(record.FullPath))
		if err != nil {
			return failStim(out, st, err, start)
		}
		client := target.OpenAICompat{BaseURL: cfg.Endpoint.BaseURL, Model: cfg.Endpoint.Model, Temperature: cfg.Endpoint.Temperature, MaxTokens: cfg.MaxOutTok}
		callStart := time.Now()
		res, err := client.CompletePerception(ctx, target.PerceptionInput{
			Question: "Two labeled values A and B are shown. Which label has the larger value? Answer with only the letter A or B.",
			Image:    image, MediaType: "image/png",
		})
		st.LatencyMS = time.Since(callStart).Milliseconds()
		if err != nil {
			return failStim(out, st, err, start)
		}
		st.WorkingSetItemCount = 1
		st.WorkingSetBytes = len(image)
		out.ModelCalls = 1
		out.Steps = append(out.Steps, st)
		answer := parseAB(res.Content)
		out.FinalAnswer = answer
		out.ContractSuccess = answer != ""
		out.FormatFailure = answer == ""
		if out.ContractSuccess {
			out.SemanticCorrect = answer == record.Larger
		}
		out.LatencyMS = time.Since(start).Milliseconds()
		return out

	case ConditionD1ExternalSeq:
		return runExternal(ctx, cfg, registry, record, condition, out, start, false)
	case ConditionD2ExternalOp1:
		return runExternal(ctx, cfg, registry, record, condition, out, start, true)
	case ConditionD3Verify:
		return runExternal(ctx, cfg, registry, record, condition, out, start, true)
	default:
		out.Error = "unknown condition " + string(condition)
		out.LatencyMS = time.Since(start).Milliseconds()
		return out
	}
}

// runExternal covers D1/D2/D3: external sequencing and working state, with
// every Parrot invocation carrying exactly one cognitive opcode.
func runExternal(ctx context.Context, cfg Config, registry *tlaloque.Registry, record parrotlab.T0ARecord, condition Condition, out StimulusOutcome, start time.Time, externalOp1 bool) StimulusOutcome {
	bb := &tlaloque.BlackboardRuntime{Store: cfg.Store, RunID: out.RunID}
	runner := tlaloque.SwarmRunner{Registry: registry, Blackboard: bb}
	imgPath := func(rel string) string { return filepath.Join(cfg.DatasetDir, filepath.FromSlash(rel)) }
	verify := condition == ConditionD3Verify

	runStep := func(stepID, capability string, input any) (json.RawMessage, error) {
		step, err := exocortex.Step{ID: stepID, Opcode: capability, Output: exocortex.Address(stepID)}.Normalize()
		if err != nil {
			return nil, err
		}
		plan, err := exocortex.StepsToPlan(stepID+"-plan", []exocortex.Step{step}, 1)
		if err != nil {
			return nil, err
		}
		body, _ := json.Marshal(input)
		report, err := runner.Run(ctx, plan, out.RunID, body)
		if err != nil {
			return nil, err
		}
		return report.TerminalOutputs[stepID], nil
	}

	snap := func() *blackboard.Snapshot {
		s, err := cfg.Store.Snapshot(out.RunID)
		if err != nil {
			return nil
		}
		return &s
	}

	valueA := ""
	stepIndex := 0

	// OP1: obtain operand A — deterministically (D2/D3) or via one atomic
	// Parrot EXTRACT_NUMBER (D1).
	before := hashSnapshot(snap())
	st := StepTrace{StepID: "op1_a", StepIndex: stepIndex, WorkflowDepth: 1, Opcode: exocortex.OpExtractNumber, StateBeforeHash: before}
	if externalOp1 {
		st.ExecutorType = "DETERMINISTIC"
		st.DeterministicOps = 1
		valueA = record.DetOperandA
		out.DeterministicOps++
	} else {
		st.ExecutorType = "MODEL"
		st.ModelCalls = 1
		st.CognitiveOpsGivenToModel = 1
		callStart := time.Now()
		raw, err := runStep("op1_a", exocortex.OpExtractNumber, exocortex.ParrotInput{ImagePath: imgPath(record.CropA), VisualField: exocortex.VisualFieldTightCrop})
		st.LatencyMS = time.Since(callStart).Milliseconds()
		if err != nil {
			return failStim(out, st, err, start)
		}
		var pr exocortex.ParrotOutput
		_ = json.Unmarshal(raw, &pr)
		valueA = strings.TrimSpace(pr.Text)
		st.WorkingSetItemCount = 1
		out.ModelCalls++
	}
	st.ObservationKey = "op1_a"
	st.StateAfterHash = hashSnapshot(snap())
	out.Steps = append(out.Steps, st)
	stepIndex++

	// OP2: obtain operand B via one atomic Parrot EXTRACT_NUMBER, reading
	// only its own crop — never OP1's narration.
	before = hashSnapshot(snap())
	st = StepTrace{StepID: "op2_b", StepIndex: stepIndex, WorkflowDepth: 2, Opcode: exocortex.OpExtractNumber, ExecutorType: "MODEL", ModelCalls: 1, CognitiveOpsGivenToModel: 1, StateBeforeHash: before}
	callStart := time.Now()
	rawB, err := runStep("op2_b", exocortex.OpExtractNumber, exocortex.ParrotInput{ImagePath: imgPath(record.CropB), VisualField: exocortex.VisualFieldTightCrop})
	st.LatencyMS = time.Since(callStart).Milliseconds()
	if err != nil {
		return failStim(out, st, err, start)
	}
	var prB exocortex.ParrotOutput
	_ = json.Unmarshal(rawB, &prB)
	valueBRaw := strings.TrimSpace(prB.Text)
	st.WorkingSetItemCount = 1
	st.ObservationKey = "op2_b"
	st.StateAfterHash = hashSnapshot(snap())
	out.ModelCalls++
	out.Steps = append(out.Steps, st)
	stepIndex++

	valueB := valueBRaw
	if condition == ConditionD3Verify || condition == ConditionD2ExternalOp1 {
		before = hashSnapshot(snap())
		st = StepTrace{StepID: "normalize_b", StepIndex: stepIndex, WorkflowDepth: 3, Opcode: exocortex.OpNormalize, ExecutorType: "DETERMINISTIC", DeterministicOps: 1, StateBeforeHash: before}
		nOut, err := runStep("normalize_b", exocortex.OpNormalize, exocortex.NormalizeInput{Raw: valueBRaw, TargetType: exocortex.TargetTypeNumber})
		if err != nil {
			return failStim(out, st, err, start)
		}
		var norm exocortex.NormalizeOutput
		_ = json.Unmarshal(nOut, &norm)
		if norm.IsNumber {
			valueB = fmt.Sprintf("%v", norm.AsNumber)
		} else {
			out.FormatFailure = true
		}
		out.DeterministicOps++
		st.StateAfterHash = hashSnapshot(snap())
		out.Steps = append(out.Steps, st)
		stepIndex++
	}

	// JOIN: deterministic COMPARE_NUMBERS over the two externally held
	// operands.
	before = hashSnapshot(snap())
	st = StepTrace{StepID: "compare", StepIndex: stepIndex, WorkflowDepth: stepIndex + 1, Opcode: exocortex.OpCompareNumbers, ExecutorType: "DETERMINISTIC", DeterministicOps: 1, StateBeforeHash: before}
	cOut, err := runStep("compare", exocortex.OpCompareNumbers, exocortex.NumericInput{A: valueA, B: valueB})
	if err != nil {
		return failStim(out, st, err, start)
	}
	var num exocortex.NumericOutput
	_ = json.Unmarshal(cOut, &num)
	out.DeterministicOps++
	st.StateAfterHash = hashSnapshot(snap())
	out.Steps = append(out.Steps, st)
	stepIndex++

	answer := "A"
	if num.Comparison == "LESS" {
		answer = "B"
	}
	out.WorkflowDepth = stepIndex

	if verify {
		before = hashSnapshot(snap())
		st = StepTrace{StepID: "verify", StepIndex: stepIndex, WorkflowDepth: stepIndex + 1, Opcode: exocortex.OpVerify, ExecutorType: "DETERMINISTIC", DeterministicOps: 1, StateBeforeHash: before}
		verified := answer == "A" || answer == "B"
		if num.Comparison == "EQUAL" {
			verified = false
		}
		out.DeterministicOps++
		st.FactPromoted = verified
		st.StateAfterHash = hashSnapshot(snap())
		out.Steps = append(out.Steps, st)
		stepIndex++
		out.WorkflowDepth = stepIndex
		if !verified {
			out.UnsupportedAssertion = true
			out.Abstained = true
			out.LatencyMS = time.Since(start).Milliseconds()
			return out
		}
	}

	out.FinalAnswer = answer
	out.ContractSuccess = !out.FormatFailure && answer != ""
	if out.ContractSuccess {
		out.SemanticCorrect = answer == record.Larger
	}
	out.LatencyMS = time.Since(start).Milliseconds()
	return out
}

func failStim(out StimulusOutcome, st StepTrace, err error, start time.Time) StimulusOutcome {
	st.Error = err.Error()
	out.Steps = append(out.Steps, st)
	out.Error = err.Error()
	out.LatencyMS = time.Since(start).Milliseconds()
	return out
}

// NewRegistry builds the T0-A registry (reuses decompositionlab.NewRegistry
// so there is exactly one Registry construction path in the codebase).
func NewRegistry(profile exocortex.CapabilityProfile, endpoint exocortex.ParrotEndpoint) (*tlaloque.Registry, error) {
	return decompositionlab.NewRegistry(profile, endpoint)
}
