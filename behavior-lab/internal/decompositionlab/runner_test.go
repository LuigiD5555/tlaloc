package decompositionlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/exocortex"
)

// fixtureExocortexProfile builds the same SYNTHETIC_TEST_FIXTURE profile
// internal/exocortex's own tests use, via the public compiler entry point,
// so this package never hand-invents CapabilityProfile numbers either.
func fixtureExocortexProfile(t *testing.T) exocortex.CapabilityProfile {
	t.Helper()
	artifact := exocortex.MicroISAArtifact{
		Schema: exocortex.MicroISAArtifactSchemaR0, ExperimentID: "synthetic-fixture", Records: 10, Frozen: true,
		MaxSafeOpsSemantic: 1, MaxSafeOpsContract: 1,
		Opcodes: map[string]exocortex.MicroISAOpcodeFinding{
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

func floatPtrT(v float64) *float64 { return &v }

func writePageImage(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 10, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode page png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write page png: %v", err)
	}
}

func newParrotServer(t *testing.T, answer string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}],"usage":{"prompt_tokens":10,"completion_tokens":1}}`, answer)
	}))
}

func baseRecord(t *testing.T, dir string) P0Record {
	t.Helper()
	pagePath := filepath.Join(dir, "page.png")
	writePageImage(t, pagePath, 600, 800)
	return P0Record{
		BaseID: "p0-img-000", Question: "how many samples", ExpectedAnswer: "126",
		Category: CategoryNumeric, DocID: "doc1", Page: 1, PageImagePath: pagePath,
		PageWidth: 600, PageHeight: 800, EvidenceAddress: "ohf://carrier/docs/doc1/pages/000001/regions/0002",
		EvidenceBBox: &canonicaldoc.BBox{X1: 10, Y1: 200, X2: 300, Y2: 260},
		Opcode:       exocortex.OpExtractNumber,
	}
}

func newRunnerConfig(t *testing.T, server *httptest.Server, profile exocortex.CapabilityProfile) RunnerConfig {
	t.Helper()
	registry, err := NewRegistry(profile, exocortex.ParrotEndpoint{BaseURL: server.URL, Model: "lfm2-vl-1.6b"})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return RunnerConfig{
		Registry: registry, Store: blackboard.New(t.TempDir()),
		MarginRatio: 0.1, CropDir: t.TempDir(),
	}
}

func TestRunRecord_C1_OracleCropAllowsExtractNumber(t *testing.T) {
	server := newParrotServer(t, "126")
	defer server.Close()
	profile := fixtureExocortexProfile(t)
	dir := t.TempDir()
	record := baseRecord(t, dir)
	cfg := newRunnerConfig(t, server, profile)

	outcome := RunRecord(context.Background(), cfg, record, ConditionC1OracleCrop)
	if outcome.Error != "" {
		t.Fatalf("unexpected error: %s", outcome.Error)
	}
	if !outcome.ContractSuccess {
		t.Fatalf("expected contract success for a tight-crop EXTRACT_NUMBER call, got %+v", outcome)
	}
	if !outcome.SemanticCorrect {
		t.Fatalf("expected semantic_correct=true (126 == 126), got %+v", outcome)
	}
	if outcome.VisualExposureRatio <= 0 || outcome.VisualExposureRatio >= 1 {
		t.Fatalf("visual_exposure_ratio = %v, want strictly between 0 and 1 for a real crop", outcome.VisualExposureRatio)
	}
	if outcome.ParrotCalls != 1 {
		t.Fatalf("parrot_calls = %d, want 1 (P1: one op per invocation)", outcome.ParrotCalls)
	}
}

func TestRunRecord_C0_IsImportedNeverReRun(t *testing.T) {
	// BLOCKER 1: C0 is the frozen P0 direct-Parrot baseline. It must be
	// imported, never re-run through the Exocortex ModelAdapter. The server
	// here would answer "126" if the pipeline ever called it — the test
	// asserts it does not.
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		fmt.Fprint(w, `{"choices":[{"message":{"content":"126"}}]}`)
	}))
	defer server.Close()
	profile := fixtureExocortexProfile(t)
	dir := t.TempDir()
	record := baseRecord(t, dir)
	cfg := newRunnerConfig(t, server, profile)
	cfg.C0Baseline = map[string]P0Outcome{
		record.BaseID: {BaseID: record.BaseID, Attempted: true, ContractSuccess: true, SemanticCorrect: true, OriginalOutput: "126", LatencyMS: 4321},
	}

	outcome := RunRecord(context.Background(), cfg, record, ConditionC0ParrotDirect)
	if called {
		t.Fatalf("C0 must make zero model calls, but the endpoint was hit")
	}
	if outcome.Error != "" {
		t.Fatalf("unexpected error: %s", outcome.Error)
	}
	if outcome.ParrotCalls != 0 {
		t.Fatalf("C0 parrot_calls = %d, want 0 (the frozen call belongs to P0, not this run)", outcome.ParrotCalls)
	}
	if !outcome.ContractSuccess || !outcome.SemanticCorrect || outcome.LatencyMS != 4321 {
		t.Fatalf("C0 row not faithfully imported from the baseline: %+v", outcome)
	}
	if outcome.VisualExposureRatio != 1.0 {
		t.Fatalf("visual_exposure_ratio = %v, want 1.0 for C0", outcome.VisualExposureRatio)
	}
}

func TestRunRecord_C0_MissingBaselineIsAnErrorNotAModelCall(t *testing.T) {
	server := newParrotServer(t, "126")
	defer server.Close()
	profile := fixtureExocortexProfile(t)
	record := baseRecord(t, t.TempDir())
	cfg := newRunnerConfig(t, server, profile) // no C0Baseline set

	outcome := RunRecord(context.Background(), cfg, record, ConditionC0ParrotDirect)
	if outcome.Error == "" {
		t.Fatalf("expected an explicit missing-baseline error, got %+v", outcome)
	}
	if outcome.ParrotCalls != 0 {
		t.Fatalf("a missing C0 baseline must never fall through to a model call")
	}
}

func TestRunRecord_C3_VerifyPromotesCorrectNumberToVerifiedFact(t *testing.T) {
	server := newParrotServer(t, " 126 ")
	defer server.Close()
	profile := fixtureExocortexProfile(t)
	dir := t.TempDir()
	record := baseRecord(t, dir)
	cfg := newRunnerConfig(t, server, profile)

	outcome := RunRecord(context.Background(), cfg, record, ConditionC3Verify)
	if outcome.Error != "" {
		t.Fatalf("unexpected error: %s", outcome.Error)
	}
	if !outcome.ContractSuccess || outcome.UnsupportedAssertion {
		t.Fatalf("expected a verified fact, got %+v", outcome)
	}
	if !outcome.SemanticCorrect {
		t.Fatalf("expected semantic_correct=true, got %+v", outcome)
	}
	if outcome.DeterministicOps < 3 {
		t.Fatalf("deterministic_ops = %d, want at least 3 (locate, crop, normalize, verify)", outcome.DeterministicOps)
	}
}

func TestRunRecord_C3_UnparseableModelOutputIsUnsupportedNotFabricated(t *testing.T) {
	server := newParrotServer(t, "I cannot tell")
	defer server.Close()
	profile := fixtureExocortexProfile(t)
	dir := t.TempDir()
	record := baseRecord(t, dir)
	cfg := newRunnerConfig(t, server, profile)

	outcome := RunRecord(context.Background(), cfg, record, ConditionC3Verify)
	if outcome.Error != "" {
		t.Fatalf("unexpected error: %s", outcome.Error)
	}
	if !outcome.UnsupportedAssertion || !outcome.Abstained {
		t.Fatalf("expected an UNSUPPORTED/abstained outcome for unparseable model text, got %+v", outcome)
	}
	if outcome.SemanticCorrect {
		t.Fatalf("an abstained record must never be scored as semantically correct")
	}
	if outcome.FinalValue != "" {
		t.Fatalf("an unsupported fact must not leave a fabricated final_value, got %q", outcome.FinalValue)
	}
}

func TestScoreSemantic_NumericToleranceAndTextCaseInsensitivity(t *testing.T) {
	if !ScoreSemantic(exocortex.OpExtractNumber, "126.0000001", "126") {
		t.Fatalf("expected numeric tolerance to accept a tiny float rounding difference")
	}
	if ScoreSemantic(exocortex.OpExtractNumber, "127", "126") {
		t.Fatalf("expected 127 != 126 to score as incorrect")
	}
	if !ScoreSemantic(exocortex.OpReadShortLabel, "Fashion MNIST", "fashion mnist") {
		t.Fatalf("expected case-insensitive text match")
	}
}
