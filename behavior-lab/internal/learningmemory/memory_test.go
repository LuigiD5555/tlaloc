package learningmemory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreIsIdempotentAndContentAddressed(t *testing.T){
	root:=t.TempDir();s:=New(root);pass:=false
	e:=Event{Schema:EventSchema,EventType:EventObservation,EvidenceClass:EvidenceRealModel,BenchmarkID:"b",TrialID:"t",QuestionID:"Q3",ModelID:"m",Pass:&pass,FailureCode:"T2_NOT_FOUND",LastCompletedStage:"ROSETTA",RecordedAt:"2026-01-01T00:00:00Z"}
	added,a,err:=s.Put(e);if err!=nil||!added{t.Fatalf("first put added=%v err=%v",added,err)}
	e.RecordedAt="2026-02-01T00:00:00Z";added,b,err:=s.Put(e);if err!=nil{t.Fatal(err)};if added{t.Fatal("same evidence must deduplicate")};if a.EventID!=b.EventID{t.Fatal("recorded_at must not change content id")}
	files,err:=os.ReadDir(filepath.Join(root,"events"));if err!=nil||len(files)!=1{t.Fatalf("files=%d err=%v",len(files),err)}
}

func TestSummaryUsesRealFailuresBeforeSynthetic(t *testing.T){
	fail:=false;pass:=true
	events:=[]Event{
		{Schema:EventSchema,EventType:EventObservation,EvidenceClass:EvidenceSynthetic,BenchmarkID:"b",TrialID:"s",QuestionID:"Q3",ModelID:"SYNTHETIC",Pass:&fail,FailureCode:"BOOT_NOT_FOUND",LastCompletedStage:"NONE",ScoreLayer:"R_PROTOCOL"},
		{Schema:EventSchema,EventType:EventObservation,EvidenceClass:EvidenceRealModel,BenchmarkID:"b",TrialID:"r1",QuestionID:"Q3",ModelID:"M1",SpecimenID:"O1",Pass:&fail,FailureCode:"T2_NOT_FOUND",LastCompletedStage:"ROSETTA",ScoreLayer:"S_SEMANTIC"},
		{Schema:EventSchema,EventType:EventObservation,EvidenceClass:EvidenceRealModel,BenchmarkID:"b",TrialID:"r2",QuestionID:"Q4",ModelID:"M2",SpecimenID:"O1",Pass:&fail,FailureCode:"T2_NOT_FOUND",LastCompletedStage:"ROSETTA",ScoreLayer:"S_SEMANTIC"},
		{Schema:EventSchema,EventType:EventObservation,EvidenceClass:EvidenceRealModel,BenchmarkID:"b",TrialID:"r3",QuestionID:"Q1",ModelID:"M2",Pass:&pass,ScoreLayer:"P_PERCEPTION"},
	}
	s:=BuildSummary("mem",events)
	if s.RealModelObservations!=3||s.SyntheticObservations!=1{t.Fatalf("unexpected counts %#v",s)}
	if len(s.TopRealFailurePatterns)!=1{t.Fatalf("patterns=%#v",s.TopRealFailurePatterns)}
	p:=s.TopRealFailurePatterns[0];if p.Count!=2||p.FailureCode!="T2_NOT_FOUND"||p.SuggestedTarget!="T2_NAVIGATION"{t.Fatalf("pattern=%#v",p)}
	if s.NextDebugTarget!="T2_NAVIGATION"{t.Fatalf("next=%s",s.NextDebugTarget)}
}

func TestCandidateOutcomeHistory(t *testing.T){before:=0.4;after:=0.8;delta:=0.4
events:=[]Event{{Schema:EventSchema,EventType:EventChange,EvidenceClass:EvidenceManual,CandidateID:"c1",ChangeSummary:"make T2 address visible",ParentEventIDs:[]string{"e1"}},{Schema:EventSchema,EventType:EventOutcome,EvidenceClass:EvidenceManual,CandidateID:"c1",ParentEventIDs:[]string{"change","post"},BeforeScore:&before,AfterScore:&after,Delta:&delta}}
s:=BuildSummary("mem",events);if s.ChangeAttempts!=1||s.OutcomeLinks!=1||len(s.CandidateOutcomes)!=1{t.Fatalf("summary=%#v",s)};if s.CandidateOutcomes[0].MeanDelta!=0.4{t.Fatalf("outcome=%#v",s.CandidateOutcomes[0])}}
