package experimentpolicy

import (
	"fmt"
	"sort"
	"strings"
)

const executeVisibleRulesToStableTextR1 = "EXEC: INIT > APPLY ALL SAME PRE-STEP > NEXT > REPEAT UNTIL UNCHANGED > REPORT STABLE"
const synchronousRuleTextR1 = "EACH STEP: TEST ALL RULES ON SAME PRE-STEP SNAPSHOT"
const synchronousConditionTextR1 = "EACH STEP: TEST ALL CONDITIONS ON SAME PRE-STEP SNAPSHOT"
const visibleFromStatePreconditionR1 = "VISIBLE_FROM_STATE_PRECONDITION_R1"
const visibleRuleRoleBindingR1 = "VISIBLE_RULE_ROLE_BINDING_R1"
const executeDontSummarizeToStableR1 = "EXECUTE_DONT_SUMMARIZE_TO_STABLE_R1"
const freezeSelectApplyTogetherR1 = "FREEZE_SELECT_APPLY_TOGETHER_R1"

var executionComplianceVisibleFactsR1 = map[string]string{
	"EXECUTION_POLICY_COMPLIANCE.MODE_TEXT":  "MODE: EXECUTE PROGRAM | DO NOT SUMMARIZE OR DESCRIBE",
	"EXECUTION_POLICY_COMPLIANCE.STEP1_TEXT": "1 READ INITIAL CELL STATES",
	"EXECUTION_POLICY_COMPLIANCE.STEP2_TEXT": "2 TEST ALL RULES",
	"EXECUTION_POLICY_COMPLIANCE.STEP3_TEXT": "3 USE SAME PRE-STEP",
	"EXECUTION_POLICY_COMPLIANCE.STEP4_TEXT": "4 APPLY FIRED SETS",
	"EXECUTION_POLICY_COMPLIANCE.STEP5_TEXT": "5 REPEAT UNTIL UNCHANGED",
	"EXECUTION_POLICY_COMPLIANCE.STEP6_TEXT": "6 REPORT FINAL A/B/C",
	"EXECUTION_POLICY_COMPLIANCE.STOP_TEXT":  "STOP ONLY WHEN UNCHANGED",
}

var synchronousExecutionFidelityVisibleFactsR1 = map[string]string{
	"SYNCHRONOUS_EXECUTION_FIDELITY.HEADER_TEXT": "SYNC: FREEZE > SELECT > APPLY",
	"SYNCHRONOUS_EXECUTION_FIDELITY.STEP1_TEXT": "1 FREEZE PRE-STEP",
	"SYNCHRONOUS_EXECUTION_FIDELITY.STEP2_TEXT": "2 FIRE IFF WHEN+TARGET=REQUIRE",
	"SYNCHRONOUS_EXECUTION_FIDELITY.STEP3_TEXT": "3 SELECT ALL | APPLY TOGETHER",
	"SYNCHRONOUS_EXECUTION_FIDELITY.NO_ORDER_TEXT": "4 NO ORDER | NO CASCADE",
	"SYNCHRONOUS_EXECUTION_FIDELITY.STOP_TEXT": "5 NEXT | STOP IF ZERO FIRE",
}

func CheckVisibleTextFidelity(candidate CandidateManifest, semantic SemanticManifest, visible VisibleTextManifest) VisibleTextFidelityReport {
	r:=VisibleTextFidelityReport{Schema:VisibleTextParitySchemaR1,CandidateID:candidate.ID,Pass:true}
	if visible.Schema!=VisibleTextSchemaR1{r.Pass=false;r.FailureCode="INVALID_VISIBLE_TEXT_MANIFEST";return r}
	if semantic.Schema!=SemanticSchemaR1{r.Pass=false;r.FailureCode="INVALID_SEMANTIC_MANIFEST";return r}
	if visible.ProgramSHA256!=semantic.ProgramSHA256 || (candidate.ProgramSHA256!=""&&visible.ProgramSHA256!=candidate.ProgramSHA256){r.Differences=append(r.Differences,Difference{Key:"PROGRAM_SHA256",Expected:semantic.ProgramSHA256,Actual:visible.ProgramSHA256,Allowed:false})}
	sm:=factMap(semantic.Facts);vm:=factMap(visible.Facts)
	cellIDs:=[]string{};for k:=range sm{if strings.HasPrefix(k,"VISIBLE_CELL_ID_"){cellIDs=append(cellIDs,strings.TrimPrefix(k,"VISIBLE_CELL_ID_"))}};sort.Strings(cellIDs)
	for _,id:=range cellIDs{visibleID:=sm["VISIBLE_CELL_ID_"+id];requireVisibleFact(&r,vm,"CELL."+id+".LABEL","CELL "+visibleID);requireVisibleFact(&r,vm,"CELL."+id+".INITIAL_TEXT",sm["CELL."+id+".INITIAL"])}

	if sm["RULE_ROLE_BINDING"]==visibleRuleRoleBindingR1 {
		requireVisibleFact(&r,vm,"TEMPORAL_GRAMMAR.SYNC_TEXT",synchronousConditionTextR1)
		for i,id:=range semanticRuleIDs(sm){if i>=4{break};req:=sm["RULE."+id+".REQUIRES"];if req==""{req="TRUE"}else{req=visibleRequires(req,sm)};target:=sm["RULE."+id+".TARGET"];visibleTarget:=sm["VISIBLE_CELL_ID_"+target];if visibleTarget==""{visibleTarget=target};from:=sm["RULE."+id+".FROM"];if from==""{from="*"};requireVisibleFact(&r,vm,"RULE."+id+".WHEN_TEXT",strings.ToUpper(id)+" WHEN "+req);requireVisibleFact(&r,vm,"RULE."+id+".ROLE_TEXT","TARGET "+visibleTarget+" | REQUIRE "+from+" | SET "+sm["RULE."+id+".TO"])}
	} else if sm["FROM_STATE_PRECONDITION_VISIBILITY"]==visibleFromStatePreconditionR1 {
		requireVisibleFact(&r,vm,"TEMPORAL_GRAMMAR.SYNC_TEXT",synchronousConditionTextR1)
		for i,id:=range semanticRuleIDs(sm){if i>=6{break};req:=sm["RULE."+id+".REQUIRES"];if req==""{req="TRUE"}else{req=visibleRequires(req,sm)};target:=sm["RULE."+id+".TARGET"];visibleTarget:=sm["VISIBLE_CELL_ID_"+target];if visibleTarget==""{visibleTarget=target};from:=sm["RULE."+id+".FROM"];if from==""{from="*"};want:=fmt.Sprintf("IF %s AND %s=%s THEN %s -> %s",req,visibleTarget,from,visibleTarget,sm["RULE."+id+".TO"]);requireVisibleFact(&r,vm,"RULE."+id+".TEXT",want)}
	} else if sm["TEMPORAL_GRAMMAR"]=="VISIBLE_RULE_MICROGRAMMAR_R1" {
		requireVisibleFact(&r,vm,"TEMPORAL_GRAMMAR.SYNC_TEXT",synchronousRuleTextR1)
		for i,id:=range semanticRuleIDs(sm){if i>=6{break};req:=sm["RULE."+id+".REQUIRES"];if req==""{req="TRUE"}else{req=visibleRequires(req,sm)};target:=sm["RULE."+id+".TARGET"];visibleTarget:=sm["VISIBLE_CELL_ID_"+target];if visibleTarget==""{visibleTarget=target};from:=sm["RULE."+id+".FROM"];if from==""{from="*"};want:=fmt.Sprintf("IF %s => %s:%s>%s",req,visibleTarget,from,sm["RULE."+id+".TO"]);requireVisibleFact(&r,vm,"RULE."+id+".TEXT",want)}
	}
	if sm["EXECUTION_POLICY"]=="EXECUTE_VISIBLE_RULES_TO_STABLE_R1"{requireVisibleFact(&r,vm,"EXECUTION_POLICY.TEXT",executeVisibleRulesToStableTextR1)}
	if sm["SYNCHRONOUS_EXECUTION_FIDELITY"]==freezeSelectApplyTogetherR1 {
		requireVisibleFact(&r,vm,"EXECUTION_POLICY_COMPLIANCE.MODE_TEXT",executionComplianceVisibleFactsR1["EXECUTION_POLICY_COMPLIANCE.MODE_TEXT"])
		keys:=make([]string,0,len(synchronousExecutionFidelityVisibleFactsR1));for k:=range synchronousExecutionFidelityVisibleFactsR1{keys=append(keys,k)};sort.Strings(keys)
		for _,k:=range keys{requireVisibleFact(&r,vm,k,synchronousExecutionFidelityVisibleFactsR1[k])}
	} else if sm["EXECUTION_POLICY_COMPLIANCE"]==executeDontSummarizeToStableR1 {
		keys:=make([]string,0,len(executionComplianceVisibleFactsR1));for k:=range executionComplianceVisibleFactsR1{keys=append(keys,k)};sort.Strings(keys)
		for _,k:=range keys{requireVisibleFact(&r,vm,k,executionComplianceVisibleFactsR1[k])}
	}
	if !r.Pass&&r.FailureCode==""{r.FailureCode="VISIBLE_TEXT_FIDELITY_FAILED"};return r
}

func requireVisibleFact(r *VisibleTextFidelityReport,actual map[string]string,key,want string){got,ok:=actual[key];if !ok||got!=want{r.Pass=false;r.Differences=append(r.Differences,Difference{Key:key,Expected:want,Actual:got,Allowed:false})}}
func semanticRuleIDs(facts map[string]string)[]string{set:=map[string]bool{};for k:=range facts{if !strings.HasPrefix(k,"RULE.")||!strings.HasSuffix(k,".TARGET"){continue};id:=strings.TrimSuffix(strings.TrimPrefix(k,"RULE."),".TARGET");if id!=""{set[id]=true}};out:=make([]string,0,len(set));for id:=range set{out=append(out,id)};sort.Strings(out);return out}
func visibleRequires(raw string,semantic map[string]string)string{parts:=strings.Split(raw,"&");for i,p:=range parts{kv:=strings.SplitN(p,"=",2);if len(kv)!=2{continue};id:=kv[0];visible:=semantic["VISIBLE_CELL_ID_"+id];if visible==""{visible=id};parts[i]=visible+"="+kv[1]};return strings.Join(parts,"&")}
