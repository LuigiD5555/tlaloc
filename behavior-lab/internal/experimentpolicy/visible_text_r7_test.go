package experimentpolicy

import "testing"

func TestVisibleTextFidelityR7RequiresSynchronousProtocol(t *testing.T) {
	semantic := SemanticManifest{Schema: SemanticSchemaR1, ProgramSHA256: "p", Facts: []SemanticFact{
		{Key: "EXECUTION_POLICY_COMPLIANCE", Value: executeDontSummarizeToStableR1},
		{Key: "SYNCHRONOUS_EXECUTION_FIDELITY", Value: freezeSelectApplyTogetherR1},
	}}
	facts := []SemanticFact{{Key: "EXECUTION_POLICY_COMPLIANCE.MODE_TEXT", Value: executionComplianceVisibleFactsR1["EXECUTION_POLICY_COMPLIANCE.MODE_TEXT"]}}
	for k, v := range synchronousExecutionFidelityVisibleFactsR1 {
		facts = append(facts, SemanticFact{Key: k, Value: v})
	}
	visible := VisibleTextManifest{Schema: VisibleTextSchemaR1, ProgramSHA256: "p", Facts: facts}
	candidate := CandidateManifest{Schema: CandidateSchemaR1, ID: "r7", ProgramSHA256: "p"}
	r := CheckVisibleTextFidelity(candidate, semantic, visible)
	if !r.Pass {
		t.Fatalf("expected R7 visible protocol pass: %#v", r)
	}

	visible.Facts = visible.Facts[:len(visible.Facts)-1]
	r = CheckVisibleTextFidelity(candidate, semantic, visible)
	if r.Pass {
		t.Fatal("missing R7 protocol fact must fail")
	}
}
