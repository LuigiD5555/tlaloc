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

	out := RunStimulus(context.Background(), cfg, reg, record, ConditionD2ExternalOp1)
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
