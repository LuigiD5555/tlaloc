package runrecord

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnvHashExcludesOnlyIdentityOutcomeAndVariableAxis(t *testing.T) {
	baseline := finalizedFixture(t)
	changedAxis := baseline
	changedAxis.Model.IDRequested = "another-model"
	changedAxis.EnvHash = ""
	changedAxis.RunID = ""
	changedAxis = mustFinalize(t, changedAxis)
	if changedAxis.EnvHash != baseline.EnvHash {
		t.Fatalf("variable axis changed env_hash: got %s want %s", changedAxis.EnvHash, baseline.EnvHash)
	}

	changedOutcome := baseline
	changedOutcome.Outcome.LatencyMS++
	changedOutcome.EnvHash = ""
	changedOutcome.RunID = ""
	changedOutcome = mustFinalize(t, changedOutcome)
	if changedOutcome.EnvHash != baseline.EnvHash {
		t.Fatalf("outcome changed env_hash: got %s want %s", changedOutcome.EnvHash, baseline.EnvHash)
	}

	changedEnvironment := baseline
	changedEnvironment.Sampling.MaxTokens++
	changedEnvironment.EnvHash = ""
	changedEnvironment.RunID = ""
	changedEnvironment = mustFinalize(t, changedEnvironment)
	if changedEnvironment.EnvHash == baseline.EnvHash {
		t.Fatal("non-axis environment change preserved env_hash")
	}
}

func TestStoreIsImmutableAndIndexesRun(t *testing.T) {
	record := finalizedFixture(t)
	root := t.TempDir()
	recordPath, err := Store(root, record)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.RunID != record.RunID || loaded.EnvHash != record.EnvHash {
		t.Fatalf("loaded identity mismatch: %#v", loaded)
	}
	indexEntries, err := LoadIndex(filepath.Join(root, "index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(indexEntries) != 1 || !strings.Contains(indexEntries[0], record.RunID) {
		t.Fatalf("unexpected index: %#v", indexEntries)
	}
	if _, err := Store(root, record); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("immutable collision was not rejected: %v", err)
	}
}

func TestReplayFromRecordReproducesEnvAndOutputHashes(t *testing.T) {
	t.Setenv("TLALOC_RUN_RECORD_TEST_HELPER", "1")
	record := finalizedFixture(t)
	recordRoot := t.TempDir()
	recordPath, err := Store(recordRoot, record)
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplayAndVerify(context.Background(), recordPath); err != nil {
		t.Fatal(err)
	}
}

func TestRunRecordReplayHelper(t *testing.T) {
	if os.Getenv("TLALOC_RUN_RECORD_TEST_HELPER") != "1" {
		return
	}
	_, _ = os.Stdout.Write([]byte("deterministic-output\n"))
	os.Exit(0)
}

func finalizedFixture(t *testing.T) Record {
	t.Helper()
	replay, err := EncodeReplay([]string{os.Args[0], "-test.run=TestRunRecordReplayHelper"})
	if err != nil {
		t.Fatal(err)
	}
	return mustFinalize(t, Record{
		Schema: Schema, VariableAxis: "model.id_requested",
		Component: Component{Tlaloc: "6.0.0-alpha.21", Origami: "6.0.0-alpha.15", TonalLock: "0.1.0-alpha.5"},
		Model: Model{
			Provider: "local", IDRequested: "model-a", IDReported: "model-a",
			Quantization: "Q4", ContextWindow: 4096, Tokenizer: "tokenizer-a",
			Endpoint: "http://127.0.0.1:1234/v1", ObservedAt: "2026-09-01T12:00:00Z",
		},
		Sampling: Sampling{Temperature: 0, TopP: 1, Seed: 42, MaxTokens: 128, Stop: []string{}},
		Prompt:   Prompt{BehaviorSpecID: "spec-a", PromptIRHash: "sha256:prompt", CompiledPromptHash: "sha256:compiled"},
		Fixture:  Fixture{ID: "fixture-a", SHA256: "sha256:fixture"},
		Host:     Host{OS: "linux", CPU: "amd64", GPU: "none", RAMGB: 1, Go: "go1.22", Python: "Python 3"},
		Outcome: Outcome{
			OutputHash: HashBytes([]byte("deterministic-output\n")), Parsed: true,
			Verdict: "verify_pass", LatencyMS: 1,
		},
		Repetitions: Repetitions{N: 1, VerdictDistribution: map[string]int{"verify_pass": 1}},
		Replay:      replay, Trace: []TransitionEvent{},
	})
}

func mustFinalize(t *testing.T, record Record) Record {
	t.Helper()
	finalized, err := Finalize(record, time.Date(2026, 9, 1, 12, 0, 0, 123, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return finalized
}
