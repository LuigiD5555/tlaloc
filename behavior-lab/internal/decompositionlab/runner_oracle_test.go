package decompositionlab

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/exocortex"
)

// fixtureSelectProfile compiles a SYNTHETIC_TEST_FIXTURE profile that
// deploys SELECT_ONE (formal choice width 2) and EXTRACT_NUMBER, via the
// public compiler entry point so no CapabilityProfile numbers are hand-
// invented here.
func fixtureSelectProfile(t *testing.T) exocortex.CapabilityProfile {
	t.Helper()
	two, eight := 2, 8
	artifact := exocortex.MicroISAArtifact{
		Schema: exocortex.MicroISAArtifactSchemaR0, ExperimentID: "synthetic-fixture", Records: 10, Frozen: true,
		MaxSafeOpsSemantic: 1, MaxSafeOpsContract: 1,
		Opcodes: map[string]exocortex.MicroISAOpcodeFinding{
			"SELECT_ONE": {
				IntrinsicVerdict:  exocortex.VerdictStrong,
				TightCropAccuracy: floatPtrT(0.85), FullPageAccuracy: floatPtrT(0.85),
				FormalMaxSafeChoiceWidth: &two, ObservedTestedChoiceWidth: &eight,
			},
			"EXTRACT_NUMBER": {
				IntrinsicVerdict: exocortex.VerdictStrong, PDFTransferVerdict: exocortex.TransferPartial,
				TightCropAccuracy: floatPtrT(0.8), FullPageAccuracy: floatPtrT(0.5),
			},
		},
	}
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	profile, err := exocortex.CompileParrotProfile(path, "parrot-lfm2-vl-1.6b", "lfm2-vl-1.6b", "r0")
	if err != nil {
		t.Fatalf("CompileParrotProfile: %v", err)
	}
	return profile
}

// oracleRecord is a locate/SELECT_ONE record with a frozen image-space
// EvidenceBBox and the recipe the frozen dataset carries.
func oracleRecord(t *testing.T, dir string) P0Record {
	t.Helper()
	pagePath := filepath.Join(dir, "page.png")
	writePageImage(t, pagePath, 1050, 1500)
	return P0Record{
		BaseID: "e2e-locate-01", Question: `Which one — "A" or "B"?`, ExpectedAnswer: "A",
		Category: CategoryLocate, TaskFamily: "choice", DocID: "fold-bench", Page: 200,
		PageImagePath: pagePath, PageWidth: 1050, PageHeight: 1500,
		EvidenceAddress: "ohf://fold-bench/pages/000200",
		EvidenceBBox:    &canonicaldoc.BBox{X1: 100, Y1: 400, X2: 900, Y2: 560},
		Choices:         []string{"A", "B"},
		Recipe:          []AtomicStep{{ID: "select", Opcode: "SELECT_ONE", EvidenceRefs: []string{"ev1"}, OutputKey: "answer", ChoiceWidth: 2}},
	}
}

func TestRunRecord_OracleConditionRequiresFrozenBBox(t *testing.T) {
	server := newParrotServer(t, "A")
	defer server.Close()
	profile := fixtureSelectProfile(t)
	dir := t.TempDir()
	rec := oracleRecord(t, dir)
	rec.EvidenceBBox = nil // simulate a skipped T0-B0 prepare
	cfg := newRunnerConfig(t, server, profile)

	outcome := RunRecord(context.Background(), cfg, rec, ConditionC1OracleCrop)
	if outcome.Error == "" || !strings.Contains(outcome.Error, "EvidenceBBox") {
		t.Fatalf("expected a hard 'no frozen EvidenceBBox' error, got %+v", outcome)
	}
	if outcome.ParrotCalls != 0 {
		t.Fatalf("a missing oracle bbox must never fall through to a model call / full page")
	}
}

func TestRunRecord_C1C2C3ShareTheSameFrozenCrop(t *testing.T) {
	server := newParrotServer(t, "A")
	defer server.Close()
	profile := fixtureSelectProfile(t)
	dir := t.TempDir()
	rec := oracleRecord(t, dir)
	cfg := newRunnerConfig(t, server, profile)

	read := func(cond Condition) []byte {
		out := RunRecord(context.Background(), cfg, rec, cond)
		if out.Error != "" {
			t.Fatalf("%s: %s", cond, out.Error)
		}
		body, err := os.ReadFile(filepath.Join(cfg.CropDir, rec.BaseID+"-"+strings.ToLower(string(cond))+".png"))
		if err != nil {
			t.Fatalf("read crop for %s: %v", cond, err)
		}
		return body
	}
	c1, c2, c3 := read(ConditionC1OracleCrop), read(ConditionC2Normalize), read(ConditionC3Verify)
	if string(c1) != string(c2) || string(c2) != string(c3) {
		t.Fatalf("C1/C2/C3 must crop the identical frozen oracle bbox for one base_id (sizes %d/%d/%d)", len(c1), len(c2), len(c3))
	}
}

func TestRunRecord_RealConditionIgnoresOracleBBox(t *testing.T) {
	server := newParrotServer(t, "A")
	defer server.Close()
	profile := fixtureSelectProfile(t)
	dir := t.TempDir()
	rec := oracleRecord(t, dir) // EvidenceBBox is set, but B1 must not use it

	// An empty-index mini store: the real locator's deterministic search
	// finds no candidate and errors. That error proves the pipeline went to
	// RealLocate (search) and not OracleLocate (frozen bbox).
	store := writeMiniStore(t, 200, 756, 1080, numericFixtureRegions())
	cfg := newRunnerConfig(t, server, profile)
	cfg.StoreDir = store

	outcome := RunRecord(context.Background(), cfg, rec, ConditionB1RealCrop)
	if outcome.Error == "" || !strings.Contains(strings.ToLower(outcome.Error), "real locate") {
		t.Fatalf("expected the REAL locator (deterministic search) to run, got %+v", outcome)
	}
}
