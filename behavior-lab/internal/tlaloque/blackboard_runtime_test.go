package tlaloque

import (
	"context"
	"encoding/json"
	"testing"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

type bbWorker struct{ desc CapabilityDescriptor; exec func(CapabilityRequest)(CapabilityResponse,error) }
func(w bbWorker)Descriptor()CapabilityDescriptor{return w.desc}
func(w bbWorker)Execute(_ context.Context,req CapabilityRequest)(CapabilityResponse,error){return w.exec(req)}

func TestSwarmBlackboardPropagatesRunSnapshot(t *testing.T){
	store:=blackboard.New(t.TempDir());registry:=NewRegistry();seen:=false
	first:=bbWorker{desc:CapabilityDescriptor{ID:"observer",Capability:"OBSERVE",Scope:ScopeGeneral,Engine:EngineDeterministic,InputSchema:"json",OutputSchema:"json",Deterministic:true},exec:func(req CapabilityRequest)(CapabilityResponse,error){if req.Blackboard==nil||len(req.Blackboard.Entries)!=0{t.Fatalf("first snapshot=%+v",req.Blackboard)};return CapabilityResponse{WorkerID:"observer",Output:json.RawMessage(`{"ok":true}`),Confidence:.8,Observations:[]blackboard.Observation{{Key:"state",Value:json.RawMessage(`"ACTIVE"`),Confidence:.8}}},nil}}
	second:=bbWorker{desc:CapabilityDescriptor{ID:"consumer",Capability:"CONSUME",Scope:ScopeGeneral,Engine:EngineDeterministic,InputSchema:"json",OutputSchema:"json",Deterministic:true},exec:func(req CapabilityRequest)(CapabilityResponse,error){if req.Blackboard==nil{t.Fatal("missing blackboard")};for _,e:=range req.Blackboard.Entries{if e.Type==blackboard.EntryObservation&&e.Key=="state"{seen=true}};return CapabilityResponse{WorkerID:"consumer",Output:json.RawMessage(`{"done":true}`)},nil}}
	if err:=registry.Register(first);err!=nil{t.Fatal(err)};if err:=registry.Register(second);err!=nil{t.Fatal(err)}
	plan:=SwarmPlan{ID:"bb",Nodes:[]SwarmNode{{ID:"observe",Capability:"OBSERVE",WorkerID:"observer"},{ID:"consume",Capability:"CONSUME",WorkerID:"consumer",DependsOn:[]string{"observe"}}}}
	report,err:=(SwarmRunner{Registry:registry,Blackboard:&BlackboardRuntime{Store:store,RunID:"run-a"}}).Run(context.Background(),plan,"task",json.RawMessage(`{"x":1}`));if err!=nil{t.Fatal(err)};if!report.Succeeded||report.RunID!="run-a"{t.Fatalf("report=%+v",report)};if!seen{t.Fatal("dependent worker did not receive prior observation")}
	snap,err:=store.Snapshot("run-a");if err!=nil{t.Fatal(err)};obs,metrics:=0,0;for _,e:=range snap.Entries{switch e.Type{case blackboard.EntryObservation:obs++;case blackboard.EntryMetric:metrics++}};if obs!=1||metrics!=2{t.Fatalf("observations=%d metrics=%d entries=%+v",obs,metrics,snap.Entries)}
}

func TestSwarmConsolidatorObservationsBecomeDecisions(t *testing.T){
	store:=blackboard.New(t.TempDir());registry:=NewRegistry()
	worker:=bbWorker{desc:CapabilityDescriptor{ID:"c",Capability:"CONSOLIDATE_BLACKBOARD",Scope:ScopeGeneral,Engine:EngineDeterministic,InputSchema:"json",OutputSchema:"json",Deterministic:true},exec:func(req CapabilityRequest)(CapabilityResponse,error){return CapabilityResponse{WorkerID:"c",Output:json.RawMessage(`{"ok":true}`),Observations:[]blackboard.Observation{{Key:"decision.Q0",Value:json.RawMessage(`{"status":"UNKNOWN"}`)}}},nil}}
	if err:=registry.Register(worker);err!=nil{t.Fatal(err)}
	_,err:=(SwarmRunner{Registry:registry,Blackboard:&BlackboardRuntime{Store:store,RunID:"run"}}).Run(context.Background(),SwarmPlan{ID:"p",Nodes:[]SwarmNode{{ID:"consolidate",Capability:"CONSOLIDATE_BLACKBOARD",WorkerID:"c"}}},"task",json.RawMessage(`{}`));if err!=nil{t.Fatal(err)}
	snap,err:=store.Snapshot("run");if err!=nil{t.Fatal(err)};found:=false;for _,e:=range snap.Entries{if e.Type==blackboard.EntryDecision&&e.Key=="decision.Q0"{found=true}};if!found{t.Fatalf("decision not persisted: %+v",snap.Entries)}
}
