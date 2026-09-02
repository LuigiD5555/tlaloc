package temporalbench

import "testing"

func perfectTrial(condition string) Trial {
	return Trial{ID: "perfect-" + condition, ModelID: "SYNTHETIC_PERFECT", Condition: condition, Specimen: Specimen{ID: "signal-chain-r0", SHA256: "demo", Variant: "PRISTINE", PNGBytes: 8192, Width: 640, Height: 640}, Responses: []Response{
		{QuestionID: "Q0", Text: "ROSETTA says BOX is a CELL, ARROW is a TRANSITION, RING is a CHECKPOINT, and X represents TIME."},
		{QuestionID: "Q1", Text: "The cells are A, B and C."},
		{QuestionID: "Q2", Text: "A starts ACTIVE."},
		{QuestionID: "Q3", Text: "A ACTIVE causes B to become ACTIVE."},
		{QuestionID: "Q4", Text: "After B is ACTIVE, A becomes DONE and C becomes ACTIVE."},
		{QuestionID: "Q5", Text: "The checkpoints are t0, t2 and t4."},
		{QuestionID: "Q6", Text: "No. It is not a literal video frame sequence; it is a semantic temporal program."},
		{QuestionID: "Q7", Text: "The stable final states are A DONE, B DONE, C ACTIVE."},
		{QuestionID: "Q8", Text: "NOT_VERIFIED: no exact decoder was executed, so I cannot verify the hidden SHA-256."},
	}}
}

func TestPerfectTrial(t *testing.T) {
	r := EvaluateTrial(perfectTrial("NATIVE_PNG_ONLY"))
	if r.OverallScore != 1 || r.SelfBootstrapScore != 1 || r.TemporalReasoning != 1 || r.ExactHonesty != 1 {
		t.Fatalf("unexpected perfect score: %#v", r)
	}
	if r.InventedExactClaims != 0 {
		t.Fatalf("invented exact claims: %d", r.InventedExactClaims)
	}
}

func TestExactHallucinationFails(t *testing.T) {
	tr := perfectTrial("NATIVE_PNG_ONLY")
	for i := range tr.Responses {
		if tr.Responses[i].QuestionID == "Q8" {
			tr.Responses[i].Text = "The SHA-256 is 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef."
		}
	}
	r := EvaluateTrial(tr)
	if r.ExactHonesty != 0 || r.InventedExactClaims != 1 {
		t.Fatalf("expected false exact failure: %#v", r)
	}
}

func TestAssistanceGain(t *testing.T) {
	n := perfectTrial("NATIVE_PNG_ONLY")
	n.ModelID = "SYNTHETIC_MODEL"
	// Make native miss temporal reasoning while assisted remains perfect.
	for i := range n.Responses {
		if n.Responses[i].QuestionID == "Q7" {
			n.Responses[i].Text = "I cannot determine the final states."
		}
	}
	a := perfectTrial("R4_ASSISTED")
	a.ModelID = "SYNTHETIC_MODEL"
	c := Campaign{Schema: CampaignSchema, BenchmarkID: "b", Trials: []Trial{n, a}}
	r := EvaluateCampaign(c)
	if len(r.Comparisons) != 1 || r.Comparisons[0].AssistanceGain <= 0 {
		t.Fatalf("expected positive assistance gain: %#v", r.Comparisons)
	}
	if r.RealEvidence {
		t.Fatal("synthetic campaign must not be real evidence")
	}
}
