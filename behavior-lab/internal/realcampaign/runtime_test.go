package realcampaign

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tlaloc.local/behaviorlab/internal/closedloop"
	"tlaloc.local/behaviorlab/internal/runrecord"
)

func TestNormalizeRejectsSyntheticAndWeakEvidence(t *testing.T) {
	program := writeCanonicalProgram(t, t.TempDir())
	base := Spec{CampaignID: "x", Phase: PhaseEvidence, Endpoint: "http://example.invalid/v1", Model: "SYNTHETIC_FAKE", Program: program, TemporalCarrier: "carrier", CandidateBuilder: "builder", OutputDir: t.TempDir(), TrialsPerModel: 3}
	if _, err := Normalize(base); err == nil {
		t.Fatal("expected synthetic model rejection")
	}
	base.Model = "real-vlm"
	base.TrialsPerModel = 2
	if _, err := Normalize(base); err == nil {
		t.Fatal("expected evidence trial floor rejection")
	}
}

func TestDoctorAndPrepareDiscoverOneRealModel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	dir := t.TempDir()
	program := writeCanonicalProgram(t, dir)
	fixturePNG := writeFrozenPNG(t, dir)
	carrier := filepath.Join(dir, "carrier")
	builder := filepath.Join(dir, "builder")
	carrierScript := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-out" ]; then out="$2"; shift 2; continue; fi
  shift
done
[ -n "$out" ] || exit 8
cp "$FIXTURE_PNG" "$out"
`
	builderScript := `#!/bin/sh
if [ "$1" = "capabilities" ]; then
cat <<'JSON'
{"schema":"origami.experimental-candidate.r0.capabilities","parent_profiles":["origami.temporal-carrier.r0.profile-1"],"supported_kinds":["LAYOUT","PROMPT"],"unsupported_kinds":["DEPTH_STRUCTURE"],"exact_plane_mutation":false,"max_mutations":8}
JSON
exit 0
fi
exit 9
`
	if err := os.WriteFile(carrier, []byte(carrierScript), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(builder, []byte(builderScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FIXTURE_PNG", fixturePNG)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "local-vision-model"}}})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "I can see the carrier."}}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	t.Setenv("TLALOC_VERSION", "6.0.0-alpha.21-test")
	spec := Spec{CampaignID: "doctor-test", Phase: PhaseSmoke, Endpoint: server.URL + "/v1", Program: program, TemporalCarrier: carrier, CandidateBuilder: builder, OutputDir: filepath.Join(dir, "campaign"), RunRecordRoot: filepath.Join(dir, "runs")}
	doc, err := Doctor(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Ready || !doc.VisionTransport || doc.SelectedModel != "local-vision-model" {
		t.Fatalf("bad doctor: %#v", doc)
	}
	prepared, err := Prepare(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Manifest.ModelID != "local-vision-model" || prepared.Manifest.BaselineBytes != 8192 {
		t.Fatalf("bad manifest: %#v", prepared.Manifest)
	}
	if prepared.Manifest.PromotionEligible || prepared.Manifest.CrossModelEvidence {
		t.Fatalf("smoke must not be promotion/cross-model evidence: %#v", prepared.Manifest)
	}
	var cfg closedloop.Config
	if err := readJSON(prepared.ConfigPath, &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoCandidates || cfg.TrialsPerModel != 1 || cfg.MaxGenerations != 1 {
		t.Fatalf("bad smoke config: %#v", cfg)
	}
	if cfg.Models[0].Name != "local-vision-model" {
		t.Fatalf("wrong model: %#v", cfg.Models)
	}
	recordPaths, err := filepath.Glob(filepath.Join(spec.RunRecordRoot, "*", "*.json"))
	if err != nil || len(recordPaths) != 1 {
		t.Fatalf("run records=%v err=%v", recordPaths, err)
	}
	record, err := runrecord.Load(recordPaths[0])
	if err != nil {
		t.Fatal(err)
	}
	if record.Model.IDReported != "local-vision-model" || record.Outcome.Verdict != "verify_pass" {
		t.Fatalf("bad run record: %#v", record)
	}
	if len(record.Trace) != 1 {
		t.Fatalf("prepare trace=%#v, want one observed transition", record.Trace)
	}
	event := record.Trace[0]
	if event.Sequence != 0 || event.From != "PENDING" || event.To != "PROMPT_COMPILED" || event.Actor != "realcampaign:prepare" {
		t.Fatalf("unexpected prepare transition: %#v", event)
	}
	if event.At == "" || event.LatencyMS < 0 {
		t.Fatalf("invalid prepare transition timing: %#v", event)
	}
}

func TestSelectModelRequiresChoiceWhenMultiple(t *testing.T) {
	if _, err := selectModel("", []string{"a", "b"}); err == nil {
		t.Fatal("expected explicit selection error")
	}
	m, err := selectModel("b", []string{"a", "b"})
	if err != nil || m != "b" {
		t.Fatalf("selection=%q err=%v", m, err)
	}
}

func writeFrozenPNG(t *testing.T, dir string) string {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 640, 640))
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	if b.Len() > 8192 {
		t.Fatalf("fixture PNG too large: %d", b.Len())
	}
	data := append([]byte(nil), b.Bytes()...)
	data = append(data, make([]byte, 8192-len(data))...)
	path := filepath.Join(dir, "fixture.png")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeCanonicalProgram(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "signal-chain.json")
	body := `{
  "schema":"origami.temporal-program.r0",
  "id":"signal-chain-r0",
  "automaton":{
    "schema":"origami.automaton.r0",
    "id":"signal-chain",
    "cells":[
      {"id":"A","kind":"SOURCE","initial_state":"ACTIVE","neighbors":["B"]},
      {"id":"B","kind":"RELAY","initial_state":"IDLE","neighbors":["A","C"]},
      {"id":"C","kind":"SINK","initial_state":"IDLE","neighbors":["B"]}
    ],
    "rules":[
      {"id":"r1","target_cell":"B","from_state":"IDLE","to_state":"ACTIVE","requires":[{"cell_id":"A","state":"ACTIVE"}]},
      {"id":"r2","target_cell":"A","from_state":"ACTIVE","to_state":"DONE","requires":[{"cell_id":"B","state":"ACTIVE"}]},
      {"id":"r3","target_cell":"C","from_state":"IDLE","to_state":"ACTIVE","requires":[{"cell_id":"B","state":"ACTIVE"}]},
      {"id":"r4","target_cell":"B","from_state":"ACTIVE","to_state":"DONE","requires":[{"cell_id":"C","state":"ACTIVE"}]}
    ]
  },
  "max_steps":8,
  "checkpoint_every":2
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
