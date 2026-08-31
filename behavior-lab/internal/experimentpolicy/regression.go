package experimentpolicy

import "strings"

// CheckRegressionPreconditions verifies that the candidate changed only the
// declared experimental surface before any real VLM sees the specimen.
func CheckRegressionPreconditions(candidate CandidateManifest, expected SemanticManifest, build BuildManifest) RegressionReport {
	r:=RegressionReport{Schema:RegressionSchemaR1,CandidateID:candidate.ID,Pass:true}
	add:=func(name string,pass bool,reason string){r.Checks=append(r.Checks,RegressionCheck{Name:name,Pass:pass,Reason:reason});if !pass{r.Pass=false}}
	add("ONE_PRIMARY_MUTATION",len(candidate.ChangedModules)==1,"candidate must declare exactly one changed module")
	add("PROGRAM_SHA_PRESERVED",expected.ProgramSHA256==build.VisibleSemantics.ProgramSHA256&&build.ProgramSHA256==build.VisibleSemantics.ProgramSHA256,"canonical TemporalProgram hash must remain unchanged")
	if candidate.PayloadSHA256!=""{add("PAYLOAD_SHA_PRESERVED",candidate.PayloadSHA256==build.PayloadSHA256,"payload hash must remain unchanged")}

	exp:=factMap(expected.Facts);act:=factMap(build.VisibleSemantics.Facts)
	frozen:=true
	for k,want:=range exp{
		if strings.HasPrefix(k,"VISIBLE_CELL_ID_"){continue}
		if act[k]!=want{frozen=false;break}
	}
	add("FROZEN_TEMPORAL_SEMANTICS",frozen,"rules, initial states, timing and execution policy inherited from the parent must not regress")
	return r
}
