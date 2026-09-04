package exocortex

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const (
	profileR1Path     = "../../profiles/parrot-lfm2-vl-1.6b-r1.json"
	profileR1HashFull = "8acc959ba72334e64704c9f5b114bfb5230ca1f58375421c17a956e26b9ba729"
	fixturePagePath   = "../parrotpresent/testdata/page.png"
)

type fakePerception struct {
	calls   int
	reply   string
	fail    error
	lastPNG int
}

func (f *fakePerception) CompletePerception(_ context.Context, in target.PerceptionInput) (target.PerceptionResult, error) {
	f.calls++
	f.lastPNG = len(in.Image)
	if f.fail != nil {
		return target.PerceptionResult{}, f.fail
	}
	return target.PerceptionResult{Content: f.reply, PromptTokensReported: 11, CompletionTokensReported: 3}, nil
}

func loadFrozenProfile(t *testing.T) CapabilityProfileR1 {
	t.Helper()
	profile, err := LoadCapabilityProfileR1(profileR1Path)
	if err != nil {
		t.Fatalf("LoadCapabilityProfileR1: %v", err)
	}
	if profile.ProfileHash != profileR1HashFull {
		t.Fatalf("frozen profile hash drifted: %s", profile.ProfileHash)
	}
	return profile
}

func newTestParrotR1(t *testing.T, client perceptionCompleter) *ParrotTlaloqueR1 {
	t.Helper()
	workers, id, hash, err := NewParrotTlaloqueR1(profileR1Path, "8acc959b", t.TempDir(), ParrotEndpoint{Model: "lfm2-vl-1.6b"})
	if err != nil {
		t.Fatalf("NewParrotTlaloqueR1: %v", err)
	}
	if id != "parrot-lfm2-vl-1.6b@r1.0.0" || hash != profileR1HashFull {
		t.Fatalf("unexpected profile identity id=%q hash=%q", id, hash)
	}
	worker := workers[0].(*ParrotTlaloqueR1)
	worker.client = client
	return worker
}

func TestNewParrotTlaloqueR1_HardValidatesTheFrozenHash(t *testing.T) {
	if _, _, _, err := NewParrotTlaloqueR1(profileR1Path, "deadbeef", "", ParrotEndpoint{Model: "m"}); err == nil {
		t.Fatal("expected a hash mismatch error")
	}
	if _, _, _, err := NewParrotTlaloqueR1("../../profiles/PROFILE_R1_VALIDATION.json", "", "", ParrotEndpoint{Model: "m"}); err == nil {
		t.Fatal("expected an error loading a non-profile document")
	}
}

// --- Layer A: adapter decisions, asserted via AdapterR1.Prepare itself ---

func TestAdapterR1_LowScaleBelowSafe_UpscalesToPreferred(t *testing.T) {
	adapter := AdapterR1{Profile: loadFrozenProfile(t)}
	decision, err := adapter.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 8, VisualFieldName: "LINE"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !transformApplied(decision, "UPSCALE_TO_PREFERRED") {
		t.Fatalf("8 px should trigger UPSCALE_TO_PREFERRED; transforms=%+v", decision.Transformations)
	}
	if got := decision.ResultingWorkingSet["target_line_height_px"]; got != float64(32) {
		t.Fatalf("target line height = %v, want 32 (profile preferred)", got)
	}
}

func TestAdapterR1_AlreadyAtPreferred_NoScaleTransform(t *testing.T) {
	adapter := AdapterR1{Profile: loadFrozenProfile(t)}
	decision, err := adapter.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 32, VisualFieldName: "LINE"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if transformApplied(decision, "UPSCALE_TO_PREFERRED") {
		t.Fatalf("32 px is already >= safe scale; no upscale expected; transforms=%+v", decision.Transformations)
	}
}

func TestAdapterR1_MissingVisualOperand_RejectsBeforeModelCall(t *testing.T) {
	adapter := AdapterR1{Profile: loadFrozenProfile(t)}
	decision, err := adapter.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: false})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !decision.Rejected || decision.ModelCallCount != 0 {
		t.Fatalf("missing visual operand must be rejected with zero model calls; got %+v", decision)
	}
}

func TestAdapterR1_HighContext_CropsToOperandLine(t *testing.T) {
	adapter := AdapterR1{Profile: loadFrozenProfile(t)}
	decision, err := adapter.Prepare(AdaptRequestR1{Opcode: OpExtractNumber, HasVisualOperand: true, LineHeightPx: 32, VisualFieldName: "FULL_PAGE"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !transformApplied(decision, "CROP_TO_OPERAND_LINE") {
		t.Fatalf("FULL_PAGE should trigger CROP_TO_OPERAND_LINE; transforms=%+v", decision.Transformations)
	}
}

// --- Layer C: the Tlaloque end to end (fake endpoint, no real model) ---

func parrotInput(t *testing.T, pageImage string, bbox *canonicaldoc.BBox) json.RawMessage {
	t.Helper()
	return parrotInputWithPage(t, pageImage, bbox, 200, 300)
}

func parrotInputWithPage(t *testing.T, pageImage string, bbox *canonicaldoc.BBox, storeW, storeH float64) json.RawMessage {
	t.Helper()
	in := ParrotR1Input{Opcode: OpExtractNumber, PageImagePath: pageImage}
	if bbox != nil {
		in.Region = &ParrotR1Region{Page: 1, BBox: bbox, PageWidth: storeW, PageHeight: storeH}
	}
	body, _ := json.Marshal(in)
	return body
}

// Requirement 3: the Tlaloque must not assume canonical store coordinates
// equal rendered-image pixels. The fixture page is 200 px wide; a store
// page width of 100 means k = 2 image px per store unit.
func TestParrotR1_StoreCoordinatesAreNotImagePixels(t *testing.T) {
	fake := &fakePerception{reply: "314"}
	worker := newTestParrotR1(t, fake)
	// operand line ~5 store units tall -> 10 image px -> below the 16 px
	// safe scale -> upscale to the profile preferred 32
	bbox := &canonicaldoc.BBox{X1: 10, Y1: 50, X2: 90, Y2: 55}
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{
		NodeID: "wf::coord", Input: parrotInputWithPage(t, fixturePagePath, bbox, 100, 150),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected exactly one model call, got %d", fake.calls)
	}
	var out ParrotR1Output
	_ = json.Unmarshal(resp.Output, &out)
	if out.SubmittedLineHeightPx < 31 || out.SubmittedLineHeightPx > 33 {
		t.Fatalf("submitted line height = %d, want ~32 (5 store units * 32/5 affine scale)", out.SubmittedLineHeightPx)
	}
	if !transformApplied(*out.AdapterDecision, "UPSCALE_TO_PREFERRED") {
		t.Fatalf("a 10 image-px line should trigger the profile upscale: %+v", out.AdapterDecision.Transformations)
	}
}

func TestParrotR1_MissingVisualOperand_MakesZeroModelCalls(t *testing.T) {
	fake := &fakePerception{reply: "999"}
	worker := newTestParrotR1(t, fake)
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "n1", Input: parrotInput(t, "", nil)})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if fake.calls != 0 {
		t.Fatalf("expected zero model calls, got %d", fake.calls)
	}
	var out ParrotR1Output
	_ = json.Unmarshal(resp.Output, &out)
	if out.ModelCalls != 0 || out.AdapterDecision == nil || !out.AdapterDecision.Rejected {
		t.Fatalf("expected a rejected decision with 0 model calls, got %+v", out)
	}
	if resp.Usage == nil || resp.Usage.UpstreamCalls != 0 {
		t.Fatalf("usage must report 0 upstream calls, got %+v", resp.Usage)
	}
	if resp.Observations[0].Provenance["status"] != "UNSUPPORTED" {
		t.Fatalf("expected UNSUPPORTED status, got %q", resp.Observations[0].Provenance["status"])
	}
}

func TestParrotR1_HappyPath_ExactlyOneModelCall_WithProfileTrace(t *testing.T) {
	fake := &fakePerception{reply: "512"}
	worker := newTestParrotR1(t, fake)
	// line bbox 8 px tall on a 300-page -> low scale -> upscale to 32
	bbox := &canonicaldoc.BBox{X1: 20, Y1: 100, X2: 180, Y2: 108}
	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "wf::extract_a", Input: parrotInput(t, fixturePagePath, bbox)})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected exactly one model call, got %d", fake.calls)
	}
	if fake.lastPNG == 0 {
		t.Fatal("model was called without an image")
	}
	var value string
	if err := json.Unmarshal(resp.Observations[0].Value, &value); err != nil || value != "512" {
		t.Fatalf("observation value = %q (%v), want 512", value, err)
	}
	if resp.Observations[0].Provenance["profile_hash"] != profileR1HashFull {
		t.Fatalf("provenance profile_hash = %q", resp.Observations[0].Provenance["profile_hash"])
	}
	if !strings.Contains(resp.Observations[0].Provenance["adapter_rules_applied"], "LOW_SCALE") {
		t.Fatalf("expected LOW_SCALE in adapter_rules_applied, got %q", resp.Observations[0].Provenance["adapter_rules_applied"])
	}
	var out ParrotR1Output
	_ = json.Unmarshal(resp.Output, &out)
	if out.ModelCalls != 1 || out.AdapterDecision == nil {
		t.Fatalf("output must carry ModelCalls=1 and the decision trace, got %+v", out)
	}
	if out.SubmittedLineHeightPx < 31 || out.SubmittedLineHeightPx > 33 {
		t.Fatalf("submitted line height = %d, want ~32", out.SubmittedLineHeightPx)
	}
	if resp.Usage.UpstreamCalls != 1 {
		t.Fatalf("usage upstream calls = %d, want 1", resp.Usage.UpstreamCalls)
	}
}

func TestParrotR1_Descriptor_IsGenerativeEXTRACT_NUMBER(t *testing.T) {
	worker := newTestParrotR1(t, &fakePerception{})
	d := worker.Descriptor()
	if d.Capability != OpExtractNumber || d.Engine != tlaloque.EngineModel || d.Deterministic {
		t.Fatalf("unexpected descriptor %+v", d)
	}
}

func TestParrotR1_InputContract_CarriesNoGroundTruthField(t *testing.T) {
	banned := []string{"gold", "expected", "answer", "scorer", "baseid", "benchmark", "target_answer"}
	for _, typ := range []reflect.Type{reflect.TypeOf(ParrotR1Input{}), reflect.TypeOf(ParrotR1Region{}), reflect.TypeOf(AdaptRequestR1{})} {
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			for _, bad := range banned {
				if strings.Contains(name, bad) {
					t.Fatalf("%s.%s looks like a ground-truth field", typ.Name(), typ.Field(i).Name)
				}
			}
		}
	}
}
