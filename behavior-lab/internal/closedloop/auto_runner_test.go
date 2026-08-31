package closedloop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
)

func TestRunAutoGeneratesBuildsAndEvaluatesCandidate(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir,"baseline.png")
	candidateTemplate := filepath.Join(dir,"candidate-template.png")
	base := writeTestPNG(t,basePath,0)
	candidate := writeTestPNG(t,candidateTemplate,255)
	_ = base
	candidate64 := base64.StdEncoding.EncodeToString(candidate)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		var req struct{Messages []struct{Role string `json:"role"`;Content json.RawMessage `json:"content"`} `json:"messages"`}
		if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{t.Errorf("decode: %v",err);w.WriteHeader(400);return}
		body:=string(req.Messages[len(req.Messages)-1].Content)
		system:=string(req.Messages[0].Content)
		isCandidate:=strings.Contains(body,candidate64)
		question:=extractQuestion(body)
		diagnostic:=strings.Contains(system,"DIAGNOSTIC MODE")
		answer:=answerFor(question)
		if !isCandidate && strings.Contains(question,"causes B") {
			answer="I cannot locate the transition."
			if diagnostic { answer += "\nORIGAMI_DEBUG_R0={\"schema\":\"tlaloc.origami-debug-trace.r0\",\"status\":\"FAIL\",\"last_completed_stage\":\"ROSETTA\",\"selected_codec\":\"ST2\",\"last_instruction\":\"READ_ROSETTA\",\"next_instruction\":\"LOCATE_T2\",\"failure_code\":\"T2_NOT_FOUND\",\"evidence_refs\":[\"T0\",\"T1\"],\"confidence\":0.9}" }
		}
		w.Header().Set("Content-Type","application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices":[]any{map[string]any{"message":map[string]any{"content":answer}}}})
	})); defer server.Close()

	builder := filepath.Join(dir,"builder")
	script := `#!/bin/sh
if [ "${1:-}" = "capabilities" ]; then
cat <<'JSON'
{"schema":"origami.experimental-candidate.r0.capabilities","parent_profiles":["origami.temporal-carrier.r0.profile-1"],"supported_kinds":["LAYOUT"],"unsupported_kinds":["DEPTH_STRUCTURE"],"exact_plane_mutation":false,"max_mutations":8}
JSON
exit 0
fi
cp "$FAKE_CANDIDATE_TEMPLATE" "$TLALOC_OUTPUT_PNG"
printf '{"candidate_id":"%s"}\n' "$TLALOC_CANDIDATE_ID"
`
	if err:=os.WriteFile(builder,[]byte(script),0o755);err!=nil{t.Fatal(err)}
	t.Setenv("FAKE_CANDIDATE_TEMPLATE",candidateTemplate)

	cfg := Config{
		Schema:ConfigSchema,RunID:"auto-test",OutputDir:filepath.Join(dir,"run"),MemoryRoot:filepath.Join(dir,"memory"),
		TrialsPerModel:1,CandidatesPerGeneration:1,MaxGenerations:1,DiagnosticRetries:true,
		Conditions:[]string{"NATIVE_PNG_ONLY"},OutcomeMetric:OutcomeNative,
		Models:[]ModelConfig{{Name:"fake",Provider:"OPENAI_COMPAT",BaseURL:server.URL,Model:"fake-vlm",TimeoutSeconds:10}},
		Baseline:SpecimenConfig{ID:"baseline",PNG:basePath},
		AutoCandidates:true,CandidateBuilder:[]string{builder},AutoCandidateBaseProfileID:"origami.temporal-carrier.r0.profile-1",AutoCandidatesPerGeneration:2,
	}
	if err:=ValidateAutoReady(context.Background(),cfg);err!=nil{t.Fatal(err)}
	report,err:=RunAuto(context.Background(),cfg);if err!=nil{t.Fatal(err)}
	if len(report.ExecutionErrors)!=0{t.Fatalf("execution errors: %+v",report.ExecutionErrors)}
	if len(report.Generations)!=1{t.Fatalf("generations=%d",len(report.Generations))}
	g:=report.Generations[0]
	if len(g.SelectedIDs)!=1||!strings.HasPrefix(g.SelectedIDs[0],"auto-layout-"){t.Fatalf("unexpected auto selection: %+v",g.SelectedIDs)}
	if len(g.Candidates)!=1||g.Candidates[0].Scores.MeanNative!=1{t.Fatalf("auto candidate did not recover: %+v",g.Candidates)}
	if len(g.Outcomes)!=1||g.Outcomes[0].Delta<=0||!g.Outcomes[0].Advanceable{t.Fatalf("missing positive auto outcome: %+v",g.Outcomes)}
	if !g.IncumbentAdvanced||report.FinalIncumbentID!=g.SelectedIDs[0]{t.Fatalf("auto candidate did not become incumbent: %+v",g)}
	if _,err:=os.Stat(g.Candidates[0].PNG);err!=nil{t.Fatalf("auto candidate PNG missing: %v",err)}

	store:=learningmemory.New(cfg.MemoryRoot);events,err:=store.LoadAll();if err!=nil{t.Fatal(err)}
	hasChange,hasOutcome:=false,false
	for _,e:=range events{
		if e.CandidateID==g.SelectedIDs[0]&&e.EventType==learningmemory.EventChange{hasChange=true}
		if e.CandidateID==g.SelectedIDs[0]&&e.EventType==learningmemory.EventOutcome{hasOutcome=true}
	}
	if !hasChange||!hasOutcome{t.Fatalf("auto loop memory links missing change=%v outcome=%v",hasChange,hasOutcome)}
}
