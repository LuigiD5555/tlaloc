package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

// TestCLI_PreflightDryRun_MakesNoNetworkCall builds and runs the real
// binary's preflight subcommand without the live-confirmation flag and
// asserts it reports a dry run and exits 0 -- zero LM Studio contact.
func TestCLI_PreflightDryRun_MakesNoNetworkCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess CLI test in -short mode")
	}
	out, err := runCLI(t, "preflight", "-root", "../..")
	if err != nil {
		t.Fatalf("preflight dry-run failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("expected DRY RUN in output, got: %s", out)
	}
}

// TestCLI_RunDryRun_MakesNoNetworkCall mirrors the preflight test for the
// run subcommand.
func TestCLI_RunDryRun_MakesNoNetworkCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess CLI test in -short mode")
	}
	out, err := runCLI(t, "run", "-root", "../..")
	if err != nil {
		t.Fatalf("run dry-run failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("expected DRY RUN in output, got: %s", out)
	}
	if !strings.Contains(out, "raw + experience") {
		t.Errorf("run dry-run did not include Experimental Spine persistence: %s", out)
	}
}

// TestCLI_ExperienceBackfill_MakesNoNetworkCall proves an already-frozen T1
// result can be projected into the common experience bundle without model
// access. The fixture contains only local JSON files.
func TestCLI_ExperienceBackfill_MakesNoNetworkCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess CLI test in -short mode")
	}
	rawDir := t.TempDir()
	writeFixtureJSON(t, filepath.Join(rawDir, "workflow_records.json"), []tonalt1arms.WorkflowRecord{{
		RunID: "run-cli", WorkflowID: "wf-001", Family: "SINGLE", Arm: "A",
		SemanticCorrect: true, ExactCorrect: true, ContractStatus: "OK",
	}})
	writeFixtureJSON(t, filepath.Join(rawDir, "node_call_records.json"), []tonalt1arms.NodeCallRecord{{
		RunID: "run-cli", WorkflowID: "wf-001", Arm: "A", NodeID: "n1",
		Capability: "EXTRACT_NUMBER", Operation: "EXTRACT_NUMBER", Model: "lfm2-vl-1.6b", RequestIndex: 0,
		TransportStatus: "OK", SchemaStatus: "OK", ContractStatus: "OK", LatencyMS: 9,
	}})
	writeFixtureJSON(t, filepath.Join(rawDir, "run_accounting.json"), tonalt1arms.RunAccounting{
		PlannedModelCallSlots: 1,
		HTTPRequestAttempts:   1,
		ValidCompletions:      1,
	})

	out, err := runCLI(t, "experience", "-raw", rawDir, "-observed-at", "2026-09-04T20:00:00Z")
	if err != nil {
		t.Fatalf("experience backfill failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "summary.json") {
		t.Errorf("expected bundle paths in output, got: %s", out)
	}
	if _, err := os.Stat(filepath.Join(rawDir, "experience", "summary.json")); err != nil {
		t.Errorf("summary.json not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rawDir, "experience", "episodes", "2026-09", "t1-run-cli-wf-001-A.json")); err != nil {
		t.Errorf("episode not written: %v", err)
	}
}

// TestCLI_LiveConfirmFlag_StillRefuses proves the hard stop holds even when
// the live-confirmation flag is explicitly passed -- this task's binary
// must never actually dial LM Studio, regardless of flags.
func TestCLI_LiveConfirmFlag_StillRefuses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess CLI test in -short mode")
	}
	out, err := runCLI(t, "preflight", "-root", "../..", "-i-understand-this-calls-lm-studio")
	if err == nil {
		t.Fatalf("expected the binary to exit non-zero even with the confirmation flag; output: %s", out)
	}
	if !strings.Contains(out, "hard stop") {
		t.Errorf("expected a hard-stop message, got: %s", out)
	}
}

// TestCLI_Doctor_ExitsZeroOnRealFixtures runs the real doctor subcommand
// end-to-end as a subprocess and checks it exits 0 (no FAIL) against the
// real frozen fixtures.
func TestCLI_Doctor_ExitsZeroOnRealFixtures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess CLI test in -short mode")
	}
	out, err := runCLI(t, "doctor", "-root", "../..")
	if err != nil {
		t.Fatalf("doctor failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "doctor summary") {
		t.Errorf("expected a doctor summary line, got: %s", out)
	}
}

func TestCLI_NoArgs_ShowsUsageAndExitsNonZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess CLI test in -short mode")
	}
	_, err := runCLI(t)
	if err == nil {
		t.Fatal("expected non-zero exit with no arguments")
	}
}

func writeFixtureJSON(t *testing.T, path string, value any) {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
