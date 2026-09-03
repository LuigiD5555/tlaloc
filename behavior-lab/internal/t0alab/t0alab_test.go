package t0alab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/exocortex"
	"tlaloc.local/behaviorlab/internal/parrotlab"
)

const testP2A = "../../experiments/parrot-microisa-r0.1/results/PARROT_MICRO_ISA_R0.json"

func testProfile(t *testing.T) exocortex.CapabilityProfile {
	t.Helper()
	p, err := exocortex.CompileParrotProfileReal(testP2A, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Skipf("real P2-A artifact unavailable: %v", err)
	}
	return p
}

func mockEndpoint(t *testing.T, calls *int64, answer string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(calls, 1)
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}],"usage":{"prompt_tokens":5,"completion_tokens":1}}`, answer)
	}))
	t.Cleanup(s.Close)
	return s
}

func fixture(t *testing.T) (Config, parrotlab.T0ARecord) {
	t.Helper()
	dir := t.TempDir()
	datasetDir := filepath.Join(dir, "datasets")
	ds, _, err := parrotlab.GenerateT0A(42, 2, datasetDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Store: blackboard.New(filepath.Join(dir, "bb")), DatasetDir: datasetDir, MaxOutTok: 8}
	return cfg, ds.Records[0]
}

func TestD0_IsOneModelCallCarryingTwoCognitiveOps(t *testing.T) {
	profile := testProfile(t)
	var calls int64
	server := mockEndpoint(t, &calls, "A")
	cfg, record := fixture(t)
	cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
	reg, err := NewRegistry(profile, cfg.Endpoint)
	if err != nil {
		t.Fatal(err)
	}

	out := RunStimulus(context.Background(), cfg, reg, record, ConditionD0Direct)
	if out.Error != "" {
		t.Fatalf("D0 error: %s", out.Error)
	}
	if calls != 1 || out.ModelCalls != 1 {
		t.Fatalf("D0 model calls: server=%d outcome=%d, want 1", calls, out.ModelCalls)
	}
	if len(out.Steps) != 1 || out.Steps[0].CognitiveOpsGivenToModel != 2 {
		t.Fatalf("D0 must be one step giving the model 2 cognitive ops, got %+v", out.Steps)
	}
}

func TestD1_IsExactlyTwoOneOpModelCalls_AndCarriesStateExternally(t *testing.T) {
	profile := testProfile(t)
	var calls int64
	server := mockEndpoint(t, &calls, "123")
	cfg, record := fixture(t)
	cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
	reg, err := NewRegistry(profile, cfg.Endpoint)
	if err != nil {
		t.Fatal(err)
	}

	out := RunStimulus(context.Background(), cfg, reg, record, ConditionD1ExternalSeq)
	if out.Error != "" {
		t.Fatalf("D1 error: %s", out.Error)
	}
	if calls != 2 || out.ModelCalls != 2 {
		t.Fatalf("D1 model calls: server=%d outcome=%d, want exactly 2", calls, out.ModelCalls)
	}
	modelSteps := 0
	for _, st := range out.Steps {
		if st.ExecutorType == "MODEL" {
			modelSteps++
			if st.CognitiveOpsGivenToModel != 1 {
				t.Fatalf("D1 model step %s gives %d cognitive ops, want exactly 1", st.StepID, st.CognitiveOpsGivenToModel)
			}
		}
	}
	if modelSteps != 2 {
		t.Fatalf("D1 has %d model steps, want 2", modelSteps)
	}
	// OP1's result must be externally persisted: the blackboard snapshot
	// changes between op1 and op2.
	var op1, op2 StepTrace
	for _, st := range out.Steps {
		switch st.StepID {
		case "op1_a":
			op1 = st
		case "op2_b":
			op2 = st
		}
	}
	if op1.StateAfterHash == "" || op1.StateAfterHash != op2.StateBeforeHash {
		t.Fatalf("D1 OP1 result not carried into OP2 via external state: op1.after=%q op2.before=%q", op1.StateAfterHash, op2.StateBeforeHash)
	}
	if op1.StateBeforeHash == op1.StateAfterHash {
		t.Fatalf("D1 OP1 did not persist anything to the blackboard")
	}
}

func TestD2_ExternalizesOP1_OneModelCall(t *testing.T) {
	profile := testProfile(t)
	var calls int64
	server := mockEndpoint(t, &calls, "123")
	cfg, record := fixture(t)
	cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
	reg, _ := NewRegistry(profile, cfg.Endpoint)

	out := RunStimulus(context.Background(), cfg, reg, record, ConditionD2OracleExternalOp1)
	if out.Error != "" {
		t.Fatalf("D2 error: %s", out.Error)
	}
	if calls != 1 || out.ModelCalls != 1 {
		t.Fatalf("D2 model calls: server=%d outcome=%d, want exactly 1", calls, out.ModelCalls)
	}
	for _, st := range out.Steps {
		if st.StepID == "op1_a" && st.ExecutorType != "DETERMINISTIC" {
			t.Fatalf("D2 OP1 must be deterministic, got %s", st.ExecutorType)
		}
	}
}

// LEAKAGE AUDIT (final, pre-execution): no D-condition workflow path may
// read the expected answer (record.Larger) before post-execution scoring.
// This test poisons Larger with the WRONG label and asserts the pipeline's
// pre-scoring answer is unaffected, while scoring dutifully uses the wrong
// label.
func TestNoLeakage_ExpectedAnswerNeverSteersTheWorkflow(t *testing.T) {
	profile := testProfile(t)
	cfg, record := fixture(t)
	// Force a scene where A is truly larger, then LIE in the label.
	record.ValueA, record.ValueB = 900, 100
	record.OracleOperandA = "900"
	record.Larger = "B" // deliberately wrong

	for _, cond := range []Condition{ConditionD1ExternalSeq, ConditionD2OracleExternalOp1, ConditionD3Verify} {
		var calls int64
		// OP2 (operand B) reads "100" from the model; D1 also reads A="900".
		server := mockEndpoint(t, &calls, "100")
		cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
		reg, _ := NewRegistry(profile, cfg.Endpoint)
		if cond == ConditionD1ExternalSeq {
			// D1 needs A from the model too; a single fixed "100" would make
			// A==B. Use a per-call sequence: first call A="900", then B="100".
			server.Close()
			var seq int64
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt64(&calls, 1)
				n := atomic.AddInt64(&seq, 1)
				ans := "100"
				if n == 1 {
					ans = "900"
				}
				fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, ans)
			}))
			t.Cleanup(server.Close)
			cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
			reg, _ = NewRegistry(profile, cfg.Endpoint)
		}

		out := RunStimulus(context.Background(), cfg, reg, record, cond)
		if out.Error != "" {
			t.Fatalf("%s error: %s", cond, out.Error)
		}
		// Pipeline computed the answer from the operands alone: A(900) > B(100) => "A".
		if out.FinalAnswer != "A" {
			t.Fatalf("%s: workflow answer %q, want \"A\" (derived from operands, not the poisoned label)", cond, out.FinalAnswer)
		}
		// Scoring used the poisoned label, so it must disagree.
		if out.SemanticCorrect {
			t.Fatalf("%s: semantic_correct=true against a poisoned label proves the label leaked into scoring only, not the workflow — but here it must be false", cond)
		}
	}
}

func TestD2_IsExplicitOracleIntervention_NotRealExtraction(t *testing.T) {
	// D2/D3 obtain operand A from T0ARecord.OracleOperandA (generator scene
	// truth), NOT from the rendered pixels. Document + lock that: change the
	// crop-A image to show a different number and assert operand A is still
	// the oracle value.
	profile := testProfile(t)
	cfg, record := fixture(t)
	record.OracleOperandA = "777"
	record.ValueA = 777
	record.ValueB = 111
	record.Larger = "A"
	var calls int64
	server := mockEndpoint(t, &calls, "111") // model would read B=111
	cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
	reg, _ := NewRegistry(profile, cfg.Endpoint)

	out := RunStimulus(context.Background(), cfg, reg, record, ConditionD2OracleExternalOp1)
	if out.Error != "" {
		t.Fatalf("D2 error: %s", out.Error)
	}
	// A=777 (oracle) vs B=111 (model) => "A".
	if out.FinalAnswer != "A" {
		t.Fatalf("D2 answer %q, want \"A\" from oracle A=777 vs model B=111", out.FinalAnswer)
	}
	if !OracleConditions()[ConditionD2OracleExternalOp1] {
		t.Fatalf("D2 must be registered as an oracle condition")
	}
}

func TestT0ADataset_SHA256IsStableAcrossRegeneration(t *testing.T) {
	_, h1, err := parrotlab.GenerateT0A(20260903, 40, filepath.Join(t.TempDir(), "a"))
	if err != nil {
		t.Fatal(err)
	}
	_, h2, err := parrotlab.GenerateT0A(20260903, 40, filepath.Join(t.TempDir(), "b"))
	if err != nil {
		t.Fatal(err)
	}
	const frozen = "5e69b1acf1348024a1f30068b32a051a3e959c80a70f742e6fcc0b06576dccfc"
	if h1 != h2 || h1 != frozen {
		t.Fatalf("T0-A dataset sha256 drift: %s / %s, frozen %s", h1, h2, frozen)
	}
}

func TestD3_VerifyGatesTheFinalAnswer(t *testing.T) {
	profile := testProfile(t)
	var calls int64
	// Same number for both operands -> deterministic COMPARE == EQUAL ->
	// Verify must refuse to promote a Fact and the outcome abstains.
	server := mockEndpoint(t, &calls, "500")
	cfg, record := fixture(t)
	cfg.Endpoint = exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "m"}
	reg, _ := NewRegistry(profile, cfg.Endpoint)

	out := RunStimulus(context.Background(), cfg, reg, record, ConditionD3Verify)
	if out.Error != "" {
		t.Fatalf("D3 error: %s", out.Error)
	}
	// D2/D3 externalize OP1 with the true value_a; OP2 returns 500. If
	// value_a != 500 they differ and a Fact is promoted; the point tested
	// here is that a verify step exists and gates promotion.
	hasVerify := false
	for _, st := range out.Steps {
		if st.Opcode == exocortex.OpVerify {
			hasVerify = true
		}
	}
	if !hasVerify {
		t.Fatalf("D3 has no verify step")
	}
	if out.UnsupportedAssertion && !out.Abstained {
		t.Fatalf("an unsupported D3 outcome must also abstain")
	}
	if out.Abstained && out.SemanticCorrect {
		t.Fatalf("an abstained outcome must never be scored semantically correct")
	}
}
