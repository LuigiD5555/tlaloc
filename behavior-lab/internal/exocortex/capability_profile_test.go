package exocortex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fixtureArtifact returns a SYNTHETIC_TEST_FIXTURE micro-ISA artifact. Its
// numbers are deliberately round/fake test values (0, 0.5, 1) and are never
// meant to resemble real P2-A evidence; they only exercise the compiler's
// structural and semantic rules.
func fixtureArtifact(t *testing.T) string {
	t.Helper()
	artifact := MicroISAArtifact{
		Schema:             MicroISAArtifactSchemaR0,
		ExperimentID:       "synthetic-test-fixture-r0",
		Records:            10,
		ExecutionErrors:    0,
		Frozen:             true,
		MaxSafeOpsSemantic: 1,
		MaxSafeOpsContract: 1,
		Opcodes: map[string]MicroISAOpcodeFinding{
			"EXTRACT_NUMBER": {
				IntrinsicVerdict:   VerdictStrong,
				SyntheticAccuracy:  floatPtr(1.0),
				PDFTransferVerdict: TransferPartial,
				TightCropAccuracy:  floatPtr(0.8),
				FullPageAccuracy:   floatPtr(0.5),
			},
			"EXTRACT_ENTITY": {
				IntrinsicVerdict:   VerdictStrong,
				PDFTransferVerdict: TransferDoesNotTransfer,
				TightCropAccuracy:  floatPtr(0.7),
				FullPageAccuracy:   floatPtr(0.3),
			},
			"SELECT_ONE": {
				IntrinsicVerdict:          VerdictUsable,
				FormalMaxSafeChoiceWidth:  intPtr(2),
				ObservedTestedChoiceWidth: intPtr(8),
			},
			"READ_SHORT_TEXT": {
				IntrinsicVerdict:  VerdictFragile,
				MaxUsefulChars:    intPtr(8),
				CharAccuracyCurve: map[string]float64{"4": 0.6, "8": 0.33},
			},
			"VISUAL_LOCATE": {
				IntrinsicVerdict:     VerdictUnusable,
				ResponseCollapse:     true,
				ExternalizeCandidate: true,
			},
		},
	}
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "PARROT_MICRO_ISA_R0.synthetic-fixture.json")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }

func TestCompileParrotProfile_PreservesFormalVsObservedChoiceWidth(t *testing.T) {
	path := fixtureArtifact(t)
	profile, err := CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	entry, ok := profile.Entry("SELECT_ONE")
	if !ok {
		t.Fatalf("SELECT_ONE entry missing")
	}
	if entry.Constraints.FormalMaxChoiceWidth != 2 {
		t.Fatalf("formal max choice width = %d, want 2 (must not be widened)", entry.Constraints.FormalMaxChoiceWidth)
	}
	if entry.Constraints.ObservedChoiceWidthEnvelope != 8 {
		t.Fatalf("observed choice width envelope = %d, want 8 (must not be collapsed to the formal rung)", entry.Constraints.ObservedChoiceWidthEnvelope)
	}
}

func TestCompileParrotProfile_ResponseCollapseIsNeverSmoothed(t *testing.T) {
	path := fixtureArtifact(t)
	profile, err := CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	entry, ok := profile.Entry("VISUAL_LOCATE")
	if !ok {
		t.Fatalf("VISUAL_LOCATE entry missing")
	}
	if !entry.ResponseCollapse {
		t.Fatalf("response_collapse must be preserved verbatim")
	}
	if entry.DeploymentRecommendation != DeploymentExternalize {
		t.Fatalf("deployment_recommendation = %q, want %q for a confirmed response collapse", entry.DeploymentRecommendation, DeploymentExternalize)
	}
}

func TestCompileParrotProfile_VisualFieldPreferenceFromTransferGap(t *testing.T) {
	path := fixtureArtifact(t)
	profile, err := CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	entry, _ := profile.Entry("EXTRACT_NUMBER")
	if entry.DeploymentRecommendation != DeploymentDeployConstrained {
		t.Fatalf("deployment_recommendation = %q, want %q for PARTIAL transfer", entry.DeploymentRecommendation, DeploymentDeployConstrained)
	}
	if len(entry.Constraints.AllowedVisualField) != 1 || entry.Constraints.AllowedVisualField[0] != VisualFieldTightCrop {
		t.Fatalf("allowed_visual_field = %v, want [TIGHT_CROP] when tight crop clearly outperforms full page", entry.Constraints.AllowedVisualField)
	}
	entity, _ := profile.Entry("EXTRACT_ENTITY")
	if entity.DeploymentRecommendation != DeploymentExternalize {
		t.Fatalf("EXTRACT_ENTITY deployment_recommendation = %q, want %q for DOES_NOT_TRANSFER", entity.DeploymentRecommendation, DeploymentExternalize)
	}
}

func TestCompileParrotProfile_RejectsUnfrozenArtifact(t *testing.T) {
	dir := t.TempDir()
	artifact := MicroISAArtifact{
		Schema: MicroISAArtifactSchemaR0, ExperimentID: "x", Records: 1, Frozen: false,
		MaxSafeOpsSemantic: 1, MaxSafeOpsContract: 1,
		Opcodes: map[string]MicroISAOpcodeFinding{"EXTRACT_NUMBER": {IntrinsicVerdict: VerdictStrong}},
	}
	body, _ := json.Marshal(artifact)
	path := filepath.Join(dir, "unfrozen.json")
	os.WriteFile(path, body, 0o644)
	if _, err := CompileParrotProfile(path, "id", "model", "r0"); err == nil {
		t.Fatalf("expected error compiling an unfrozen artifact")
	}
}

func TestCompileParrotProfile_RejectsExecutionErrors(t *testing.T) {
	dir := t.TempDir()
	artifact := MicroISAArtifact{
		Schema: MicroISAArtifactSchemaR0, ExperimentID: "x", Records: 10, Frozen: true, ExecutionErrors: 2,
		MaxSafeOpsSemantic: 1, MaxSafeOpsContract: 1,
		Opcodes: map[string]MicroISAOpcodeFinding{"EXTRACT_NUMBER": {IntrinsicVerdict: VerdictStrong}},
	}
	body, _ := json.Marshal(artifact)
	path := filepath.Join(dir, "errors.json")
	os.WriteFile(path, body, 0o644)
	if _, err := CompileParrotProfile(path, "id", "model", "r0"); err == nil {
		t.Fatalf("expected error compiling an artifact with execution errors")
	}
}

func TestCompileParrotProfile_RejectsNarrowerObservedThanFormal(t *testing.T) {
	dir := t.TempDir()
	artifact := MicroISAArtifact{
		Schema: MicroISAArtifactSchemaR0, ExperimentID: "x", Records: 10, Frozen: true,
		MaxSafeOpsSemantic: 1, MaxSafeOpsContract: 1,
		Opcodes: map[string]MicroISAOpcodeFinding{
			"SELECT_ONE": {IntrinsicVerdict: VerdictUsable, FormalMaxSafeChoiceWidth: intPtr(4), ObservedTestedChoiceWidth: intPtr(2)},
		},
	}
	body, _ := json.Marshal(artifact)
	path := filepath.Join(dir, "inverted.json")
	os.WriteFile(path, body, 0o644)
	if _, err := CompileParrotProfile(path, "id", "model", "r0"); err == nil {
		t.Fatalf("expected error when observed envelope is narrower than the formal rung")
	}
}

func TestVerifySourceArtifact_DetectsTamper(t *testing.T) {
	path := fixtureArtifact(t)
	profile, err := CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	if err := VerifySourceArtifact(profile); err != nil {
		t.Fatalf("VerifySourceArtifact on untouched artifact: %v", err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("tamper: %v", err)
	}
	if err := VerifySourceArtifact(profile); err == nil {
		t.Fatalf("expected VerifySourceArtifact to detect tampering")
	}
}

func TestWriteAndLoadProfile_RoundTrip(t *testing.T) {
	path := fixtureArtifact(t)
	profile, err := CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	out := filepath.Join(t.TempDir(), "profile.json")
	if err := WriteProfile(out, profile); err != nil {
		t.Fatalf("WriteProfile: %v", err)
	}
	reloaded, err := LoadProfile(out)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if reloaded.ProfileID != profile.ProfileID || len(reloaded.Capabilities) != len(profile.Capabilities) {
		t.Fatalf("round trip mismatch: %+v vs %+v", reloaded, profile)
	}
}

func TestValidateProfile_RejectsDuplicateOpcode(t *testing.T) {
	profile := CapabilityProfile{
		Schema: ProfileSchemaR0, ProfileID: "x@r0", ExecutorID: "x", ExecutorKind: "MODEL",
		SourceExperiment: "e", SourceArtifactHash: "h", MaxSafeOps: 1,
		Capabilities: []CapabilityEntry{
			{Opcode: "EXTRACT_NUMBER", DeploymentRecommendation: DeploymentDeploy},
			{Opcode: "EXTRACT_NUMBER", DeploymentRecommendation: DeploymentDeploy},
		},
	}
	if err := ValidateProfile(profile); err == nil {
		t.Fatalf("expected duplicate opcode to be rejected")
	}
}
