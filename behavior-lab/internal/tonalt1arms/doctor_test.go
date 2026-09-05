package tonalt1arms

import (
	"context"
	"testing"

	"tlaloc.local/behaviorlab/internal/lfm2boundary"
)

func realDoctorConfig(t *testing.T) DoctorConfig {
	t.Helper()
	return DoctorConfig{
		WorkflowsPath:  RepoPathHelper("experiments/tonal-t1/d4/T1_D4_WORKFLOWS.json"),
		ArmAPolicyPath: RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_A_POLICY.json"),
		ArmBPolicyPath: RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_B_POLICY.json"),
		ArmCPolicyPath: RepoPathHelper("experiments/tonal-t1/d4/T1_D5_ARM_C_POLICY.json"),
		V2GoldPath:     RepoPathHelper("internal/tonalt1/v2_frozen/T1_D4_GOLD_v2_FULL.json"),
	}
}

func TestRunDoctor_AllChecksPassOrHonestlyUnavailable(t *testing.T) {
	results := RunDoctor(context.Background(), realDoctorConfig(t))
	if len(results) == 0 {
		t.Fatal("RunDoctor returned zero checks")
	}
	for _, r := range results {
		if r.Status == "FAIL" {
			t.Errorf("check %s (%s) FAILED: %s", r.ID, r.Name, r.Evidence)
		}
		if r.Evidence == "" {
			t.Errorf("check %s (%s) has empty Evidence", r.ID, r.Name)
		}
	}

	passed, failed, notAvailable := SummarizeDoctorResults(results)
	if failed != 0 {
		t.Fatalf("failed = %d, want 0", failed)
	}
	t.Logf("doctor summary: %d passed, %d failed, %d not available", passed, failed, notAvailable)
}

// TestRunDoctor_WeightsGuardIsHonestlyReported confirms the doctor's
// MODEL_WEIGHTS_IDENTITY_GUARD check reports NOT_AVAILABLE rather than a
// fabricated PASS, matching RunIdentityPreflight's own behavior.
func TestRunDoctor_WeightsGuardIsHonestlyReported(t *testing.T) {
	results := RunDoctor(context.Background(), realDoctorConfig(t))
	found := false
	for _, r := range results {
		if r.Name == "MODEL_WEIGHTS_IDENTITY_GUARD" {
			found = true
			if r.Status == "PASS" {
				t.Fatal("MODEL_WEIGHTS_IDENTITY_GUARD reported PASS -- this must be NOT_AVAILABLE given the known sidecar mismatch")
			}
			if r.Status != "NOT_AVAILABLE" {
				t.Fatalf("status = %q, want NOT_AVAILABLE", r.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected a MODEL_WEIGHTS_IDENTITY_GUARD check in RunDoctor's results")
	}
}

// TestRunDoctor_WithFakeIdentityDial exercises the alternate path where a
// dial function IS supplied -- still must never report PASS, since the
// underlying weightsIdentityGuard investigation result doesn't change.
func TestRunDoctor_WithFakeIdentityDial(t *testing.T) {
	cfg := realDoctorConfig(t)
	cfg.IdentityDial = fakeDial(lfm2boundary.PreflightResult{Model: lfm2boundary.RequiredModel, ContextLength: lfm2boundary.RequiredContext}, nil)
	results := RunDoctor(context.Background(), cfg)
	for _, r := range results {
		if r.Name == "MODEL_WEIGHTS_IDENTITY_GUARD" && r.Status == "PASS" {
			t.Fatal("MODEL_WEIGHTS_IDENTITY_GUARD reported PASS even with a fake dial -- weights identity was never actually verified")
		}
	}
}

func TestRunDoctor_ImageManifestCheck_ReportsNotAvailableWithoutManifest(t *testing.T) {
	cfg := realDoctorConfig(t)
	results := RunDoctor(context.Background(), cfg)
	for _, r := range results {
		if r.ID == "D08" && r.Status != "NOT_AVAILABLE" {
			t.Fatalf("D08 status = %q, want NOT_AVAILABLE (no manifest supplied)", r.Status)
		}
	}
}

func TestRunDoctor_ImageManifestCheck_PassesWithRealManifest(t *testing.T) {
	manifest, err := LoadImageManifest(RepoPathHelper("internal/tonalt1/v2_frozen/TONAL_T1_IMAGE_MANIFEST_FINAL.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := realDoctorConfig(t)
	cfg.ImageManifest = manifest
	results := RunDoctor(context.Background(), cfg)
	for _, r := range results {
		if r.ID == "D08" && r.Status != "PASS" {
			t.Fatalf("D08 status = %q, want PASS with a real manifest supplied: %s", r.Status, r.Evidence)
		}
	}
}
