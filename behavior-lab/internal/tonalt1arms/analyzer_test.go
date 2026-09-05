package tonalt1arms

import "testing"

func fixtureRunResult() RunResult {
	return RunResult{
		WorkflowRecords: []WorkflowRecord{
			{WorkflowID: "wf-1", Arm: "A", SemanticCorrect: true, ExactCorrect: true, ContractStatus: "OK", LatencyMS: 100},
			{WorkflowID: "wf-2", Arm: "A", SemanticCorrect: false, ContractStatus: "PARSE_FAILED", LatencyMS: 120},
			{WorkflowID: "wf-1", Arm: "B", SemanticCorrect: true, ExactCorrect: true, ContractStatus: "OK", LatencyMS: 200},
			{WorkflowID: "wf-2", Arm: "B", SemanticCorrect: true, ExactCorrect: false, ContractStatus: "OK", LatencyMS: 210},
			{WorkflowID: "wf-1", Arm: "C", SemanticCorrect: true, ContractStatus: "OK", LatencyMS: 90},
			{WorkflowID: "wf-2", Arm: "C", SemanticCorrect: false, ContractStatus: "CONTRACT_FAILURE", LatencyMS: 95},
		},
		Accounting: RunAccounting{
			PlannedModelCallSlots: 696, HTTPRequestAttempts: 690, ValidCompletions: 680,
			TransportFailures: 5, SchemaFailures: 3, ModelContractFailures: 2, BlockedByDependency: 6,
		},
	}
}

func fixtureCounterfactualOutcomes() []BlackboardCounterfactualOutcome {
	return []BlackboardCounterfactualOutcome{
		{Operation: "POISON", TerminalChanged: true},
		{Operation: "REMOVE", FailedClosed: true},
		{Operation: "POISON", PrimaryObservationUnavailable: true},
	}
}

func TestAnalyze_ArmSummaries(t *testing.T) {
	report := Analyze(fixtureRunResult(), nil)
	if len(report.ArmSummaries) != 3 {
		t.Fatalf("got %d arm summaries, want 3", len(report.ArmSummaries))
	}
	// Sorted by Arm: A, B, C.
	if report.ArmSummaries[0].Arm != "A" || report.ArmSummaries[1].Arm != "B" || report.ArmSummaries[2].Arm != "C" {
		t.Fatalf("arm summaries not sorted: %+v", report.ArmSummaries)
	}
	armA := report.ArmSummaries[0]
	if armA.N != 2 || armA.SemanticCorrect != 1 {
		t.Errorf("Arm A summary = %+v, want N=2 SemanticCorrect=1", armA)
	}
	if armA.FailureTaxonomy["PARSE_FAILED"] != 1 {
		t.Errorf("Arm A failure taxonomy = %+v, want PARSE_FAILED:1", armA.FailureTaxonomy)
	}
}

func TestAnalyze_PairedComparisons(t *testing.T) {
	report := Analyze(fixtureRunResult(), nil)
	if len(report.PairedComparisons) != 3 {
		t.Fatalf("got %d paired comparisons, want 3 (A-B, A-C, B-C)", len(report.PairedComparisons))
	}
}

func TestAnalyze_CallEconomics(t *testing.T) {
	report := Analyze(fixtureRunResult(), nil)
	if report.CallEconomics.PlannedModelCallSlots != 696 {
		t.Errorf("PlannedModelCallSlots = %d, want 696", report.CallEconomics.PlannedModelCallSlots)
	}
	if report.CallEconomics.HTTPRequestAttempts != 690 {
		t.Errorf("HTTPRequestAttempts = %d, want 690", report.CallEconomics.HTTPRequestAttempts)
	}
}

func TestAnalyze_CounterfactualSummary(t *testing.T) {
	report := Analyze(fixtureRunResult(), fixtureCounterfactualOutcomes())
	if report.CounterfactualSummary.Total != 3 {
		t.Errorf("Total = %d, want 3", report.CounterfactualSummary.Total)
	}
	if report.CounterfactualSummary.PrimaryObservationUnavailable != 1 {
		t.Errorf("PrimaryObservationUnavailable = %d, want 1", report.CounterfactualSummary.PrimaryObservationUnavailable)
	}
	if report.CounterfactualSummary.FailedClosed != 1 {
		t.Errorf("FailedClosed = %d, want 1", report.CounterfactualSummary.FailedClosed)
	}
	if report.CounterfactualSummary.TerminalChanged != 1 {
		t.Errorf("TerminalChanged = %d, want 1", report.CounterfactualSummary.TerminalChanged)
	}
}

// TestAnalyzer_DeterministicAcrossRuns is the required determinism proof:
// running the analyzer twice on the same fixture input must produce
// byte-identical marshaled output.
func TestAnalyzer_DeterministicAcrossRuns(t *testing.T) {
	fixture := fixtureRunResult()
	cf := fixtureCounterfactualOutcomes()

	report1 := Analyze(fixture, cf)
	bytes1, err := report1.MarshalDeterministicJSON()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 50; i++ {
		report2 := Analyze(fixtureRunResult(), fixtureCounterfactualOutcomes())
		bytes2, err := report2.MarshalDeterministicJSON()
		if err != nil {
			t.Fatal(err)
		}
		if string(bytes1) != string(bytes2) {
			t.Fatalf("run %d: analyzer output diverged:\n--- run 1 ---\n%s\n--- run %d ---\n%s", i, bytes1, i, bytes2)
		}
	}
}

func TestWilsonScoreInterval_Bounds(t *testing.T) {
	lower, upper := wilsonScoreInterval(10, 10)[0], wilsonScoreInterval(10, 10)[1]
	if lower < 0 || upper > 1 {
		t.Fatalf("interval [%v,%v] out of [0,1] bounds", lower, upper)
	}
	lower, upper = wilsonScoreInterval(0, 10)[0], wilsonScoreInterval(0, 10)[1]
	if lower < 0 || upper > 1 {
		t.Fatalf("interval [%v,%v] out of [0,1] bounds", lower, upper)
	}
}
