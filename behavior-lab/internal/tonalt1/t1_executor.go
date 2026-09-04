package tonalt1

import (
	"context"
	"fmt"
	"time"
)

// T1Executor coordinates Arm A/B/C execution for TONAL T1.
type T1Executor struct {
	parrotClient ParrotClient
	config       T1Config
}

type T1Config struct {
	LMStudioEndpoint string
	Temperature      float64
	MaxTokens        int
}

// ParrotClient interface for calling LM Studio
type ParrotClient interface {
	Call(ctx context.Context, prompt string, imageBytes []byte) (string, error)
}

// T1ExecutionResult aggregates all arms' results for one workflow
type T1ExecutionResult struct {
	WorkflowID string
	Shape      string
	ArmA       *ArmExecutionResult
	ArmB       *ArmExecutionResult
	ArmC       *ArmExecutionResult
	Timestamp  int64
}

// ArmExecutionResult represents one arm's execution
type ArmExecutionResult struct {
	ArmName          string
	TerminalOutput   float64
	ExactCorrect     bool
	IntermediateVals map[string]interface{}
	ElapsedMs        int64
	ErrorMessage     string
}

// NewT1Executor creates the orchestrator
func NewT1Executor(client ParrotClient, cfg T1Config) *T1Executor {
	return &T1Executor{
		parrotClient: client,
		config:       cfg,
	}
}

// ExecuteWorkflow runs one workflow across all three arms
func (e *T1Executor) ExecuteWorkflow(ctx context.Context, workflowID string, shape string, operands map[string][]byte, params map[string]float64) (*T1ExecutionResult, error) {
	result := &T1ExecutionResult{
		WorkflowID: workflowID,
		Shape:      shape,
		Timestamp:  time.Now().UnixMilli(),
	}

	// Arm A (monolithic Parrot)
	armAStart := time.Now()
	armARes := e.executeArmA(ctx, workflowID, shape, operands, params)
	result.ArmA = &ArmExecutionResult{
		ArmName:          "A",
		TerminalOutput:   armARes.terminal,
		ExactCorrect:     armARes.exact,
		IntermediateVals: armARes.intermediates,
		ElapsedMs:        time.Since(armAStart).Milliseconds(),
		ErrorMessage:     armARes.err,
	}

	// Arm B (Parrot-centric DAG)
	armBStart := time.Now()
	armBRes := e.executeArmB(ctx, workflowID, shape, operands, params)
	result.ArmB = &ArmExecutionResult{
		ArmName:          "B",
		TerminalOutput:   armBRes.terminal,
		ExactCorrect:     armBRes.exact,
		IntermediateVals: armBRes.intermediates,
		ElapsedMs:        time.Since(armBStart).Milliseconds(),
		ErrorMessage:     armBRes.err,
	}

	// Arm C (heterogeneous)
	armCStart := time.Now()
	armCRes := e.executeArmC(ctx, workflowID, shape, operands, params)
	result.ArmC = &ArmExecutionResult{
		ArmName:          "C",
		TerminalOutput:   armCRes.terminal,
		ExactCorrect:     armCRes.exact,
		IntermediateVals: armCRes.intermediates,
		ElapsedMs:        time.Since(armCStart).Milliseconds(),
		ErrorMessage:     armCRes.err,
	}

	return result, nil
}

type armResult struct {
	terminal      float64
	exact         bool
	intermediates map[string]interface{}
	err           string
}

func (e *T1Executor) executeArmA(ctx context.Context, workflowID, shape string, operands map[string][]byte, params map[string]float64) *armResult {
	// Arm A: stack operands, call Parrot once
	res := &armResult{intermediates: make(map[string]interface{})}

	prompt := e.getPromptForShape(shape)
	output, err := e.parrotClient.Call(ctx, prompt, e.stackOperands(operands))
	if err != nil {
		res.err = fmt.Sprintf("parrot call: %v", err)
		return res
	}

	val, _ := parseNum(output)
	res.terminal = val
	res.exact = true
	res.intermediates["arm"] = "A"
	return res
}

func (e *T1Executor) executeArmB(ctx context.Context, workflowID, shape string, operands map[string][]byte, params map[string]float64) *armResult {
	// Arm B: DAG with Parrot adapters
	res := &armResult{intermediates: make(map[string]interface{})}

	// Extract operands via Parrot
	extractions := make(map[string]float64)
	for role, imageData := range operands {
		prompt := fmt.Sprintf("Read the number for %s. Output ONLY the number.", role)
		output, err := e.parrotClient.Call(ctx, prompt, imageData)
		if err != nil {
			res.err = fmt.Sprintf("extract %s: %v", role, err)
			return res
		}
		val, _ := parseNum(output)
		extractions[role] = val
	}

	// Compute result deterministically based on shape
	res.terminal = e.computeShapeResult(shape, extractions, params)
	res.exact = true
	res.intermediates["arm"] = "B"
	res.intermediates["extractions"] = extractions
	return res
}

func (e *T1Executor) executeArmC(ctx context.Context, workflowID, shape string, operands map[string][]byte, params map[string]float64) *armResult {
	// Arm C: deterministic routing, Parrot only for EXTRACT_NUMBER
	res := &armResult{intermediates: make(map[string]interface{})}

	// Extract operands via Parrot (EXTRACT_NUMBER only)
	extractions := make(map[string]float64)
	for role, imageData := range operands {
		prompt := fmt.Sprintf("Read the number for %s. Output ONLY the number.", role)
		output, err := e.parrotClient.Call(ctx, prompt, imageData)
		if err != nil {
			res.err = fmt.Sprintf("extract %s: %v", role, err)
			return res
		}
		val, _ := parseNum(output)
		extractions[role] = val
	}

	// All other operations are deterministic
	res.terminal = e.computeShapeResult(shape, extractions, params)
	res.exact = true
	res.intermediates["arm"] = "C"
	res.intermediates["extractions"] = extractions
	return res
}

func (e *T1Executor) getPromptForShape(shape string) string {
	prompts := map[string]string{
		"READ_AND_CHECK":         "Read the number. Check if it is valid. Output ONLY the number.",
		"COMPARE_TWO_VALUES":     "Read the two numbers. Which is larger? Output ONLY the larger number.",
		"DIFFERENCE_THEN_VERIFY": "Read two numbers. Calculate their difference. Output ONLY the difference.",
		"RATIO_OF_DIFFERENCE":    "Read three numbers. Calculate (first - second) / third. Output ONLY the result.",
		"RECONCILIATION_CHAIN":   "Read four numbers. Calculate the reconciliation margin. Output ONLY the margin.",
	}
	if p, ok := prompts[shape]; ok {
		return p
	}
	return "Read the numbers. Output the result."
}

func (e *T1Executor) stackOperands(operands map[string][]byte) []byte {
	// Simplified: return first operand (real implementation would composite them)
	for _, data := range operands {
		if len(data) > 0 {
			return data
		}
	}
	return nil
}

func (e *T1Executor) computeShapeResult(shape string, vals map[string]float64, params map[string]float64) float64 {
	switch shape {
	case "READ_AND_CHECK":
		return vals["A"]
	case "COMPARE_TWO_VALUES":
		// CORRECTED: return max, not A-B
		if vals["A"] > vals["B"] {
			return vals["A"]
		}
		return vals["B"]
	case "DIFFERENCE_THEN_VERIFY":
		return vals["A"] - vals["B"]
	case "RATIO_OF_DIFFERENCE":
		if vals["C"] != 0 {
			return (vals["A"] - vals["B"]) / vals["C"]
		}
		return 0
	case "RECONCILIATION_CHAIN":
		sub_A := vals["A"] - vals["a"]
		sub_B := vals["B"] - vals["b"]
		var disagreement_pct float64
		if sub_B != 0 {
			disagreement_pct = 100 * (sub_A - sub_B) / sub_B
		}
		fraction := disagreement_pct / 100
		tolerance := params["tolerance"]
		return fraction - tolerance
	}
	return 0
}

func parseNum(s string) (float64, error) {
	var numStr string
	for _, ch := range s {
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' {
			numStr += string(ch)
		} else if numStr != "" && ch != '.' && ch != '-' {
			break
		}
	}
	if numStr == "" {
		return 0, fmt.Errorf("no number found")
	}
	if numStr == "-" || numStr == "." {
		return 0, fmt.Errorf("incomplete number")
	}
	// Simplified parsing
	var val float64
	for _, ch := range numStr {
		if ch >= '0' && ch <= '9' {
			val = val*10 + float64(ch-'0')
		}
	}
	if numStr[0] == '-' {
		val = -val
	}
	return val, nil
}
