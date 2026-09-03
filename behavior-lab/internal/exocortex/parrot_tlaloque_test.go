package exocortex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// tinyPNG returns the smallest valid PNG bytes, sufficient for
// CompletePerception's transport layer (it never decodes the image itself).
func writeTinyPNG(t *testing.T, path string) {
	t.Helper()
	// 1x1 transparent PNG.
	data := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write tiny png: %v", err)
	}
}

func TestParrotTlaloque_ContractViolationNeverReachesTheModel(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	profile := fixtureProfile(t) // EXTRACT_NUMBER: TIGHT_CROP only
	worker := ParrotTlaloque{Opcode: OpExtractNumber, Profile: profile, Endpoint: ParrotEndpoint{BaseURL: server.URL, Model: "lfm2-vl-1.6b"}}
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "page.png")
	writeTinyPNG(t, imgPath)
	input, _ := json.Marshal(ParrotInput{ImagePath: imgPath, VisualField: VisualFieldFullPage})

	_, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "n1", Input: input})
	if err == nil {
		t.Fatalf("expected CAPABILITY_CONTRACT_VIOLATION for full-page EXTRACT_NUMBER")
	}
	if _, ok := err.(*ContractViolationError); !ok {
		t.Fatalf("expected *ContractViolationError, got %T: %v", err, err)
	}
	if calls != 0 {
		t.Fatalf("model endpoint was called %d times; a contract violation must never reach the model", calls)
	}
}

func TestParrotTlaloque_SendsFixedInstructionAndWritesUnverifiedObservation(t *testing.T) {
	var receivedQuestion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		messages := body["messages"].([]any)
		user := messages[1].(map[string]any)
		content := user["content"].([]any)
		receivedQuestion = content[0].(map[string]any)["text"].(string)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"126"}}],"usage":{"prompt_tokens":50,"completion_tokens":2}}`)
	}))
	defer server.Close()

	profile := fixtureProfile(t)
	worker := ParrotTlaloque{Opcode: OpExtractNumber, Profile: profile, Endpoint: ParrotEndpoint{BaseURL: server.URL, Model: "lfm2-vl-1.6b"}}
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "crop.png")
	writeTinyPNG(t, imgPath)
	input, _ := json.Marshal(ParrotInput{ImagePath: imgPath, VisualField: VisualFieldTightCrop})

	resp, err := worker.Execute(context.Background(), tlaloque.CapabilityRequest{NodeID: "fashion_mnist_count", Input: input})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	wantInstruction, _ := FixedInstruction(OpExtractNumber)
	if receivedQuestion != wantInstruction {
		t.Fatalf("model received %q, want the fixed instruction %q", receivedQuestion, wantInstruction)
	}
	if len(resp.Observations) != 1 {
		t.Fatalf("expected exactly one Observation, got %d", len(resp.Observations))
	}
	obs := resp.Observations[0]
	if obs.Key != "fashion_mnist_count" {
		t.Fatalf("observation key = %q, want the node id", obs.Key)
	}
	var text string
	json.Unmarshal(obs.Value, &text)
	if text != "126" {
		t.Fatalf("observation value = %q, want \"126\"", text)
	}
	// The response is a raw Observation, never a Fact: nothing in this
	// package lets a model call promote its own output (E0.6, E0.12).
	if obs.Provenance["source"] == "" {
		t.Fatalf("expected provenance to record the producing executor")
	}
}

func TestNewParrotTlaloques_SkipsExternalizedAndNonEligibleOpcodes(t *testing.T) {
	profile := fixtureProfile(t)
	workers := NewParrotTlaloques(profile, ParrotEndpoint{BaseURL: "http://example.invalid", Model: "lfm2-vl-1.6b"})
	opcodes := map[string]bool{}
	for _, w := range workers {
		opcodes[w.Opcode] = true
	}
	if opcodes["EXTRACT_ENTITY"] {
		t.Fatalf("EXTRACT_ENTITY is EXTERNALIZE in the fixture profile; it must not get a Parrot worker")
	}
	if !opcodes["EXTRACT_NUMBER"] {
		t.Fatalf("EXTRACT_NUMBER is DEPLOY_WITH_CONSTRAINTS in the fixture profile; it should get a Parrot worker")
	}
	if opcodes[OpCompareNumbers] {
		t.Fatalf("COMPARE_NUMBERS has no fixed one-op instruction; it must not get a Parrot worker")
	}
}
