package selectionprofiles

import (
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

func profiledDescriptor(workerID, modelID, condition string) tlaloque.CapabilityDescriptor {
	return tlaloque.CapabilityDescriptor{
		ID:           workerID,
		Capability:   "CLASSIFY",
		Scope:        tlaloque.ScopeGeneral,
		Engine:       tlaloque.EngineModel,
		InputSchema:  "text",
		OutputSchema: "class",
		EmpiricalProfile: &tlaloque.EmpiricalProfileRef{
			ModelID:   modelID,
			Condition: condition,
		},
	}
}

func TestBuildMapsExactModelConditionsToWorkers(t *testing.T) {
	summary := learningmemory.Summary{
		Schema: "tlaloc.learning-memory.r0.summary",
		ModelProfiles: []learningmemory.ModelPerformanceProfile{
			{ModelID: "DeepSeek", Condition: "protocol-A", Cases: 100, Passes: 90, Failures: 10, Accuracy: 0.90},
			{ModelID: "LFM2-VL", Condition: "protocol-C", Cases: 100, Passes: 70, Failures: 30, Accuracy: 0.70},
		},
		PairwiseProfiles: []learningmemory.PairwiseModelProfile{
			{ModelA: "DeepSeek", ConditionA: "protocol-A", ModelB: "LFM2-VL", ConditionB: "protocol-C", SharedCases: 100, FailureOverlap: 0.2, Complementarity: 0.8, OracleSuccess: 0.97},
		},
	}
	descriptors := []tlaloque.CapabilityDescriptor{
		profiledDescriptor("deepseek-worker", "DeepSeek", "protocol-A"),
		profiledDescriptor("lfm-worker", "LFM2-VL", "protocol-C"),
	}
	catalog, source, err := Build(summary, descriptors)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Schema != CatalogSchema || catalog.SummarySchema != summary.Schema {
		t.Fatalf("catalog metadata=%+v", catalog)
	}
	if len(catalog.Workers) != 2 || !catalog.Workers[0].Observed || !catalog.Workers[1].Observed {
		t.Fatalf("workers=%+v", catalog.Workers)
	}
	deepseek, ok := source.WorkerMetric("deepseek-worker")
	if !ok || deepseek.Cases != 100 || deepseek.Accuracy != 0.90 {
		t.Fatalf("deepseek metric=%+v ok=%v", deepseek, ok)
	}
	pair, ok := source.PairMetric("lfm-worker", "deepseek-worker")
	if !ok || pair.SharedCases != 100 || pair.Complementarity != 0.8 {
		t.Fatalf("pair=%+v ok=%v", pair, ok)
	}
	if len(catalog.Pairs) != 1 || catalog.Pairs[0].OracleSuccess != 0.97 {
		t.Fatalf("catalog pairs=%+v", catalog.Pairs)
	}
}

func TestBuildKeepsSameModelConditionsDistinct(t *testing.T) {
	summary := learningmemory.Summary{
		ModelProfiles: []learningmemory.ModelPerformanceProfile{
			{ModelID: "Qwen", Condition: "json", Cases: 10, Accuracy: 0.9},
			{ModelID: "Qwen", Condition: "plain", Cases: 10, Accuracy: 0.6},
		},
		PairwiseProfiles: []learningmemory.PairwiseModelProfile{
			{ModelA: "Qwen", ConditionA: "json", ModelB: "Qwen", ConditionB: "plain", SharedCases: 10, Complementarity: 0.4},
		},
	}
	catalog, source, err := Build(summary, []tlaloque.CapabilityDescriptor{
		profiledDescriptor("qwen-json", "Qwen", "json"),
		profiledDescriptor("qwen-plain", "Qwen", "plain"),
	})
	if err != nil {
		t.Fatal(err)
	}
	jsonMetric, _ := source.WorkerMetric("qwen-json")
	plainMetric, _ := source.WorkerMetric("qwen-plain")
	if jsonMetric.Accuracy != 0.9 || plainMetric.Accuracy != 0.6 {
		t.Fatalf("json=%+v plain=%+v", jsonMetric, plainMetric)
	}
	if len(catalog.Pairs) != 1 || catalog.Pairs[0].Complementarity != 0.4 {
		t.Fatalf("pairs=%+v", catalog.Pairs)
	}
}

func TestBuildPersistsUnobservedProfileButLeavesSelectionPriorUntouched(t *testing.T) {
	catalog, source, err := Build(learningmemory.Summary{}, []tlaloque.CapabilityDescriptor{
		profiledDescriptor("new-worker", "NewModel", "protocol-X"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Workers) != 1 || catalog.Workers[0].Observed {
		t.Fatalf("workers=%+v", catalog.Workers)
	}
	if metric, ok := source.WorkerMetric("new-worker"); ok {
		t.Fatalf("unobserved worker must not create empirical quality evidence: %+v", metric)
	}
}

func TestBuildIgnoresDescriptorsWithoutEmpiricalProfile(t *testing.T) {
	plain := tlaloque.CapabilityDescriptor{
		ID:           "parser",
		Capability:   "PARSE",
		Scope:        tlaloque.ScopeGeneral,
		Engine:       tlaloque.EngineDeterministic,
		InputSchema:  "text",
		OutputSchema: "json",
	}
	catalog, _, err := Build(learningmemory.Summary{}, []tlaloque.CapabilityDescriptor{plain})
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Workers) != 0 || len(catalog.Pairs) != 0 {
		t.Fatalf("catalog=%+v", catalog)
	}
}

func TestBuildRejectsDuplicateWorkerDescriptors(t *testing.T) {
	desc := profiledDescriptor("dup", "M", "cfg")
	if _, _, err := Build(learningmemory.Summary{}, []tlaloque.CapabilityDescriptor{desc, desc}); err == nil {
		t.Fatal("expected duplicate worker descriptors to fail")
	}
}
