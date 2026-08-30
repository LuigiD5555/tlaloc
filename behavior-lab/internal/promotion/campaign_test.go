package promotion

import "testing"

func record(model string,trial int,transport string,real,hybrid,native,tool bool)TrialRecord{
	kind:="MOCK";if real{kind="REAL_MODEL"};return TrialRecord{Model:model,Trial:trial,Transport:transport,EvidenceKind:kind,Evaluation:EvaluatorReport{Model:model,Trial:trial,Transport:transport,EvidenceKind:kind,HybridTrialPromotionOK:hybrid&&real,NativeT3TrialPromotionOK:native&&real},ToolLoopPassed:tool,ToolCalls:boolInt(tool),AnswerPresent:tool}
}
func boolInt(v bool)int{if v{return 1};return 0}
func goodRouting()RoutingEvidence{return RoutingEvidence{Documents:5,Queries:6,PrimaryDocHitRate:1,VerifiedEvidenceRate:1,BudgetViolations:0,FalseExact:0}}

func TestMockCannotSatisfyCrossModelPromotion(t *testing.T){
	var records []TrialRecord
	for _,m:=range []string{"a","b","c"}{for i:=1;i<=3;i++{records=append(records,record(m,i,"original",false,true,true,true))}}
	for _,transport:=range []string{"resize-75","resize-50","jpeg-preview"}{records=append(records,record("a",10,transport,false,true,true,true))}
	report,err:=Evaluate(records,goodRouting(),DefaultPolicy());if err!=nil{t.Fatal(err)}
	if report.HybridSupportedCandidate||report.NativeVisualSupportedCandidate{t.Fatalf("mock evidence promoted: %+v",report)}
}

func TestHybridCanPromoteCandidateWithoutNativeT3(t *testing.T){
	var records []TrialRecord
	for _,m:=range []string{"a","b","c"}{for i:=1;i<=3;i++{records=append(records,record(m,i,"original",true,true,false,i==1))}}
	for i,transport:=range []string{"resize-75","resize-50","jpeg-preview"}{records=append(records,record([]string{"a","b","c"}[i],20+i,transport,true,true,false,false))}
	report,err:=Evaluate(records,goodRouting(),DefaultPolicy());if err!=nil{t.Fatal(err)}
	if !report.HybridSupportedCandidate{t.Fatalf("Hybrid candidate should pass: %+v",report)}
	if report.NativeVisualSupportedCandidate{t.Fatalf("Native must remain independent: %+v",report)}
}

func TestNativeRequiresThreeModelsThreeOriginalTrials(t *testing.T){
	var records []TrialRecord
	for _,m:=range []string{"a","b","c"}{for i:=1;i<=3;i++{records=append(records,record(m,i,"original",true,true,true,i==1))}}
	for i,transport:=range []string{"resize-75","resize-50","jpeg-preview"}{records=append(records,record([]string{"a","b","c"}[i],20+i,transport,true,true,false,false))}
	report,err:=Evaluate(records,goodRouting(),DefaultPolicy());if err!=nil{t.Fatal(err)}
	if !report.HybridSupportedCandidate||!report.NativeVisualSupportedCandidate{t.Fatalf("expected both candidate gates: %+v",report)}
}

func TestRoutingFalseExactBlocksPromotion(t *testing.T){
	var records []TrialRecord
	for _,m:=range []string{"a","b","c"}{for i:=1;i<=3;i++{records=append(records,record(m,i,"original",true,true,true,i==1))}}
	for i,transport:=range []string{"resize-75","resize-50","jpeg-preview"}{records=append(records,record([]string{"a","b","c"}[i],20+i,transport,true,true,true,false))}
	routing:=goodRouting();routing.FalseExact=1
	report,err:=Evaluate(records,routing,DefaultPolicy());if err!=nil{t.Fatal(err)}
	if report.HybridSupportedCandidate||report.NativeVisualSupportedCandidate{t.Fatalf("FALSE_EXACT must block stack candidates: %+v",report)}
}

func TestMissingTransportBlocksHybridCandidate(t *testing.T){
	var records []TrialRecord
	for _,m:=range []string{"a","b","c"}{for i:=1;i<=3;i++{records=append(records,record(m,i,"original",true,true,false,i==1))}}
	report,err:=Evaluate(records,goodRouting(),DefaultPolicy());if err!=nil{t.Fatal(err)}
	if report.HybridSupportedCandidate{t.Fatalf("transport gate unexpectedly bypassed: %+v",report)}
}
