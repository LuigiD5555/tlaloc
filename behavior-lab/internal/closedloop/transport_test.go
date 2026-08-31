package closedloop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"tlaloc.local/behaviorlab/internal/learningmemory"
)

func TestBaselineTransportFailureStopsWithoutSemanticMemory(t *testing.T){
	dir:=t.TempDir();pngPath:=filepath.Join(dir,"baseline.png");writeTestPNG(t,pngPath,64)
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Error(w,"temporary transport failure",http.StatusServiceUnavailable)}));defer server.Close()
	cfg:=Config{Schema:ConfigSchema,RunID:"transport-stop",OutputDir:filepath.Join(dir,"run"),MemoryRoot:filepath.Join(dir,"memory"),TrialsPerModel:1,MaxGenerations:2,DiagnosticRetries:true,Conditions:[]string{"NATIVE_PNG_ONLY"},Models:[]ModelConfig{{Name:"offline",Provider:"OPENAI_COMPAT",BaseURL:server.URL,Model:"fake",TimeoutSeconds:2}},Baseline:SpecimenConfig{ID:"baseline",PNG:pngPath}}
	report,err:=Run(context.Background(),cfg);if err!=nil{t.Fatal(err)};if report.StopReason!="BASELINE_EXECUTION_UNAVAILABLE"{t.Fatalf("stop=%s",report.StopReason)};if len(report.ExecutionErrors)==0{t.Fatal("expected execution error")};if len(report.Generations)!=1||report.Generations[0].Baseline.Scores.CleanTrials!=0{t.Fatalf("unexpected baseline report %+v",report.Generations)}
	store:=learningmemory.New(cfg.MemoryRoot);events,err:=store.LoadAll();if err!=nil{t.Fatal(err)};for _,e:=range events{if e.EventType==learningmemory.EventObservation{t.Fatalf("transport failure leaked into semantic memory: %+v",e)}}
}
