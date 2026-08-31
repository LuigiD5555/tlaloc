package closedloop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

func TestClosedLoopBaselineDiagnosticCandidateOutcome(t *testing.T){
	dir:=t.TempDir();basePath:=filepath.Join(dir,"baseline.png");candidatePath:=filepath.Join(dir,"candidate.png");base:=writeTestPNG(t,basePath,0);candidate:=writeTestPNG(t,candidatePath,255);base64Candidate:=base64.StdEncoding.EncodeToString(candidate);_ = base
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		var req struct{Messages []struct{Role string `json:"role"`;Content json.RawMessage `json:"content"`} `json:"messages"`};if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{t.Errorf("decode: %v",err);w.WriteHeader(400);return}
		body:=string(req.Messages[len(req.Messages)-1].Content);system:=string(req.Messages[0].Content);isCandidate:=strings.Contains(body,base64Candidate);question:=extractQuestion(body);diagnostic:=strings.Contains(system,"DIAGNOSTIC MODE")
		answer:=answerFor(question)
		if !isCandidate&&strings.Contains(question,"causes B") { answer="I cannot locate the transition.";if diagnostic{answer += "\nORIGAMI_DEBUG_R0={\"schema\":\"tlaloc.origami-debug-trace.r0\",\"status\":\"FAIL\",\"last_completed_stage\":\"ROSETTA\",\"selected_codec\":\"ST2\",\"last_instruction\":\"READ_ROSETTA\",\"next_instruction\":\"LOCATE_T2\",\"failure_code\":\"T2_NOT_FOUND\",\"evidence_refs\":[\"T0\",\"T1\"],\"confidence\":0.9}"}}
		w.Header().Set("Content-Type","application/json");_ = json.NewEncoder(w).Encode(map[string]any{"choices":[]any{map[string]any{"message":map[string]any{"content":answer}}}})
	}));defer server.Close()
	cfg:=Config{Schema:ConfigSchema,RunID:"test-run",OutputDir:filepath.Join(dir,"run"),MemoryRoot:filepath.Join(dir,"memory"),TrialsPerModel:1,CandidatesPerGeneration:1,MaxGenerations:1,DiagnosticRetries:true,Conditions:[]string{"NATIVE_PNG_ONLY"},OutcomeMetric:OutcomeNative,Models:[]ModelConfig{{Name:"fake",Provider:"OPENAI_COMPAT",BaseURL:server.URL,Model:"fake-vlm",TimeoutSeconds:10}},Baseline:SpecimenConfig{ID:"baseline",PNG:basePath},Candidates:[]CandidateConfig{{ID:"candidate-layout",PNG:candidatePath,BaseProfileID:"profile-3",Mutations:[]visualsearch.Mutation{{Kind:visualsearch.MutationLayout,Target:"T1_TO_T2_ENTRY_ROUTE",Value:"EXPLICIT_DIRECTIONAL_ANCHOR",Experimental:true}}}}}
	report,err:=Run(context.Background(),cfg);if err!=nil{t.Fatal(err)};if len(report.ExecutionErrors)!=0{t.Fatalf("execution errors: %+v",report.ExecutionErrors)};if len(report.Generations)!=1{t.Fatalf("generations=%d",len(report.Generations))};g:=report.Generations[0];if g.Baseline.Scores.MeanNative>=1{t.Fatalf("baseline should expose failure: %+v",g.Baseline.Scores)};if len(g.Candidates)!=1||g.Candidates[0].Scores.MeanNative!=1{t.Fatalf("candidate did not recover: %+v",g.Candidates)};if len(g.Outcomes)!=1||g.Outcomes[0].Delta<=0{t.Fatalf("missing positive outcome: %+v",g.Outcomes)}
	store:=learningmemory.New(cfg.MemoryRoot);events,err:=store.LoadAll();if err!=nil{t.Fatal(err)};hasDebug,hasChange,hasOutcome:=false,false,false;for _,e:=range events{if e.FailureCode=="T2_NOT_FOUND"{hasDebug=true};if e.EventType==learningmemory.EventChange&&e.CandidateID=="candidate-layout"{hasChange=true};if e.EventType==learningmemory.EventOutcome&&e.CandidateID=="candidate-layout"{hasOutcome=true}};if !hasDebug||!hasChange||!hasOutcome{t.Fatalf("memory missing loop links debug=%v change=%v outcome=%v",hasDebug,hasChange,hasOutcome)}
}

func writeTestPNG(t *testing.T,path string,v uint8)[]byte{t.Helper();img:=image.NewGray(image.Rect(0,0,16,16));for y:=0;y<16;y++{for x:=0;x<16;x++{img.SetGray(x,y,color.Gray{Y:v})}};f,err:=os.Create(path);if err!=nil{t.Fatal(err)};if err:=png.Encode(f,img);err!=nil{f.Close();t.Fatal(err)};if err:=f.Close();err!=nil{t.Fatal(err)};b,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)};return b}
func extractQuestion(body string)string{var content []map[string]any;if err:=json.Unmarshal([]byte(body),&content);err!=nil{return body};for _,p:=range content{if p["type"]=="text"{if s,ok:=p["text"].(string);ok{return s}}};return body}
func answerFor(q string)string{switch{case strings.Contains(q,"BOX"):return "BOX means CELL, ARROW means TRANSITION, RING means CHECKPOINT, X means TIME.";case strings.Contains(q,"cells or agents"):return "A, B, C";case strings.Contains(q,"initial state"):return "A is ACTIVE";case strings.Contains(q,"causes B"):return "A ACTIVE causes B ACTIVE";case strings.Contains(q,"after B"):return "A becomes DONE and C becomes ACTIVE";case strings.Contains(q,"checkpoint"):return "T0 T2 T4";case strings.Contains(q,"literal video"):return "No, it is not a literal video frame sequence.";case strings.Contains(q,"final states"):return "A DONE, B DONE, C ACTIVE";case strings.Contains(q,"SHA-256"):return "NOT_VERIFIED: no exact decoder is available.";default:return "UNKNOWN"}}
