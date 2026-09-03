package exocortex

import (
	"os"
	"path/filepath"
	"testing"
)

const realP2APath = "../../experiments/parrot-microisa-r0.1/results/PARROT_MICRO_ISA_R0.json"

func skipIfNoRealArtifact(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(realP2APath); err != nil {
		t.Skipf("real frozen P2-A artifact not present at %s", realP2APath)
	}
}

func TestCompileParrotProfileReal_ParsesTheOnDiskArtifact(t *testing.T) {
	skipIfNoRealArtifact(t)
	profile, err := CompileParrotProfileReal(realP2APath, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfileReal: %v", err)
	}
	if profile.MaxSafeOps != 1 {
		t.Fatalf("max_safe_ops = %d, want 1 (frozen artifact)", profile.MaxSafeOps)
	}
	if err := ValidateProfile(profile); err != nil {
		t.Fatalf("compiled profile fails ValidateProfile: %v", err)
	}
	// The compiler must never fabricate a profile: hash must be recorded.
	if profile.SourceArtifactHash == "" {
		t.Fatalf("compiled profile has no source_artifact_hash_sha256")
	}
	want := map[string]string{
		OpExtractNumber: DeploymentDeployConstrained, // FRAGILE + PARTIAL_TRANSFER
		OpExtractEntity: DeploymentExternalize,       // DOES_NOT_TRANSFER
		OpReadShortText: DeploymentExternalize,       // DOES_NOT_TRANSFER
		OpVisualLocate:  DeploymentDoNotDeploy,       // intrinsic UNUSABLE
	}
	for op, wantRec := range want {
		e, ok := profile.Entry(op)
		if !ok {
			t.Fatalf("compiled profile missing opcode %s", op)
		}
		if e.DeploymentRecommendation != wantRec {
			t.Fatalf("%s deployment = %s, want %s", op, e.DeploymentRecommendation, wantRec)
		}
	}
	sel, _ := profile.Entry(OpSelectOne)
	if sel.Constraints.FormalMaxChoiceWidth != 2 {
		t.Fatalf("SELECT_ONE formal choice width = %d, want 2 (frozen limits.choice_width)", sel.Constraints.FormalMaxChoiceWidth)
	}
	if sel.Constraints.ObservedChoiceWidthEnvelope <= 2 {
		t.Fatalf("SELECT_ONE observed envelope = %d, want > 2 (ladder walked wider)", sel.Constraints.ObservedChoiceWidthEnvelope)
	}
}

func TestLoadMicroISAArtifactReal_RejectsUnfrozen(t *testing.T) {
	dir := t.TempDir()
	resultsDir := filepath.Join(dir, "results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := `{"experiment_id":"x","max_safe_ops":1,"deployment_recommendation":{"EXTRACT_NUMBER":{"recommendation":"FRAGILE"}}}`
	path := filepath.Join(resultsDir, "PARROT_MICRO_ISA_R0.json")
	if err := os.WriteFile(path, []byte(artifact), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "FREEZE.json"), []byte(`{"global":{"frozen":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadMicroISAArtifactReal(path); err == nil {
		t.Fatalf("expected rejection of an unfrozen artifact")
	}
}
