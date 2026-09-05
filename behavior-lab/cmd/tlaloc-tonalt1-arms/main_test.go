package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
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

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
