package tonalt1arms

import (
	"encoding/json"
	"os"
)

// Operand is one numeric value in a workflow, with its PDF location and properties.
type Operand struct {
	CandidateID           string  `json:"candidate_id"`
	Role                  string  `json:"role"`
	NumericValue          float64 `json:"numeric_value"`
	MorphologyFamily      string  `json:"morphology_family"`
	Page                  int     `json:"page"`
	RegionID              string  `json:"region_id"`
	OperandHash           string  `json:"operand_hash"`
	EligibleAsDenominator bool    `json:"eligible_as_denominator"`
}

// Workflow is one frozen task, frozen operands and expected gold result.
type Workflow struct {
	WorkflowID        string      `json:"workflow_id"`
	Shape             string      `json:"shape"`
	NaturalDepth      int         `json:"natural_depth"`
	CriticalPathDepth int         `json:"critical_path_depth"`
	Operands          []Operand   `json:"operands"`
	DistinctPages     []int       `json:"distinct_pages"`
	WorkflowHash      string      `json:"workflow_hash"`
	Gold              interface{} `json:"gold"` // Always null in the input; included for completeness
}

// Gold is the expected ground truth for a workflow.
type Gold struct {
	WorkflowID         string             `json:"workflow_id"`
	Shape              string             `json:"shape"`
	OperandValues      map[string]float64 `json:"operand_values"`
	IntermediateValues map[string]float64 `json:"intermediate_values"`
	FinalExpectedValue float64            `json:"final_expected_value"`
	FinalStatus        string             `json:"final_status"`
	IntermediateSteps  interface{}        `json:"intermediate_steps"` // Currently always null
}

// PoisonTrial describes a counterfactual operand mutation and its expected effect.
type PoisonTrial struct {
	WorkflowID       string  `json:"workflow_id"`
	Shape            string  `json:"shape"`
	OperandRole      string  `json:"operand_role"`
	CandidateID      string  `json:"candidate_id"`
	OriginalValue    float64 `json:"original_value"`
	PoisonValue      float64 `json:"poison_value"`
	ExpectedBehavior string  `json:"expected_behavior"` // "downstream result changes or safe contract failure"
}

// RemoveTrial describes a counterfactual operand removal and its expected effect.
type RemoveTrial struct {
	WorkflowID          string  `json:"workflow_id"`
	Shape               string  `json:"shape"`
	OperandRole         string  `json:"operand_role"`
	CandidateID         string  `json:"candidate_id"`
	OriginalValue       float64 `json:"original_value"`
	ExpectedFailureCode string  `json:"expected_failure_code"` // "UNKNOWN|UNSUPPORTED|INPUT_BUILD_FAILURE"
}

// ArmAPolicy is the frozen policy for Arm A (monolithic Parrot).
type ArmAPolicy struct {
	ArmID                     string                 `json:"arm_id"`
	Name                      string                 `json:"name"`
	Description               string                 `json:"description"`
	PolicyHash                string                 `json:"policy_hash"`
	ModelCallCountPerWorkflow int                    `json:"model_call_count_per_workflow"`
	ImageConstruction         map[string]interface{} `json:"image_construction"`
	ParrotPrompt              map[string]interface{} `json:"parrot_prompt"`
	Parser                    string                 `json:"parser"`
	FailureHandling           string                 `json:"failure_handling"`
	Frozen                    bool                   `json:"frozen"`
}

// ArmBPolicy is the frozen policy for Arm B (Parrot-centric DAG).
type ArmBPolicy struct {
	ArmID              string                 `json:"arm_id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	PolicyHash         string                 `json:"policy_hash"`
	DeterministicNodes []string               `json:"deterministic_nodes"`
	ParrotAdapters     map[string]AdapterSpec `json:"parrot_adapters"`
	AdapterIsolation   string                 `json:"adapter_isolation"`
	Frozen             bool                   `json:"frozen"`
}

// AdapterSpec describes a Parrot adapter in a policy.
type AdapterSpec struct {
	PromptTemplate string `json:"prompt_template"`
	Temperature    int    `json:"temperature"`
	MaxTokens      int    `json:"max_tokens"`
}

// ArmCPolicy is the frozen policy for Arm C (heterogeneous Tonal).
type ArmCPolicy struct {
	ArmID            string            `json:"arm_id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	PolicyHash       string            `json:"policy_hash"`
	RegistryBehavior string            `json:"registry_behavior"`
	ExecutorRouting  map[string]string `json:"executor_routing"`
	Frozen           bool              `json:"frozen"`
}

// LoadWorkflows loads the frozen workflow array from a JSON file.
func LoadWorkflows(path string) ([]Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var workflows []Workflow
	if err := json.Unmarshal(data, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}

// LoadGold loads the frozen gold-standard array from a JSON file.
func LoadGold(path string) ([]Gold, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var golds []Gold
	if err := json.Unmarshal(data, &golds); err != nil {
		return nil, err
	}
	return golds, nil
}

// LoadPoisonTrials loads the frozen poison-counterfactual array.
func LoadPoisonTrials(path string) ([]PoisonTrial, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var trials []PoisonTrial
	if err := json.Unmarshal(data, &trials); err != nil {
		return nil, err
	}
	return trials, nil
}

// LoadRemoveTrials loads the frozen remove-counterfactual array.
func LoadRemoveTrials(path string) ([]RemoveTrial, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var trials []RemoveTrial
	if err := json.Unmarshal(data, &trials); err != nil {
		return nil, err
	}
	return trials, nil
}

// LoadArmAPolicy loads the frozen Arm A policy.
func LoadArmAPolicy(path string) (*ArmAPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var policy ArmAPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// LoadArmBPolicy loads the frozen Arm B policy.
func LoadArmBPolicy(path string) (*ArmBPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var policy ArmBPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// LoadArmCPolicy loads the frozen Arm C policy.
func LoadArmCPolicy(path string) (*ArmCPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var policy ArmCPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}
