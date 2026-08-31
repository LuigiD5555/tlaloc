package experimentpolicy

import (
	"fmt"
	"sort"
)

func CheckParity(candidate CandidateManifest, expected, actual SemanticManifest) ParityReport {
	r := ParityReport{Schema:ParitySchemaR1, CandidateID:candidate.ID, Pass:true}
	if candidate.Schema != CandidateSchemaR1 { r.Pass=false; r.FailureCode="INVALID_CANDIDATE_MANIFEST"; return r }
	if expected.Schema != SemanticSchemaR1 || actual.Schema != SemanticSchemaR1 { r.Pass=false; r.FailureCode="INVALID_SEMANTIC_MANIFEST"; return r }
	if candidate.ProgramSHA256 != "" && actual.ProgramSHA256 != candidate.ProgramSHA256 {
		r.Differences=append(r.Differences,Difference{Key:"PROGRAM_SHA256",Expected:candidate.ProgramSHA256,Actual:actual.ProgramSHA256,Allowed:false})
	}
	if expected.ProgramSHA256 != actual.ProgramSHA256 {
		r.Differences=append(r.Differences,Difference{Key:"PROGRAM_SHA256",Expected:expected.ProgramSHA256,Actual:actual.ProgramSHA256,Allowed:false})
	}
	if expected.PayloadSHA256 != actual.PayloadSHA256 {
		r.Differences=append(r.Differences,Difference{Key:"PAYLOAD_SHA256",Expected:expected.PayloadSHA256,Actual:actual.PayloadSHA256,Allowed:false})
	}

	declared:=factMap(candidate.ExpectedSemanticChanges)
	exp:=factMap(expected.Facts); act:=factMap(actual.Facts)
	keys:=map[string]bool{}
	for k:=range exp{keys[k]=true};for k:=range act{keys[k]=true};for k:=range declared{keys[k]=true}
	ordered:=make([]string,0,len(keys));for k:=range keys{ordered=append(ordered,k)};sort.Strings(ordered)
	for _,k:=range ordered{
		want,changed:=declared[k]
		if changed {
			if act[k]==want { if exp[k]!=act[k]{r.Differences=append(r.Differences,Difference{Key:k,Expected:want,Actual:act[k],Allowed:true})};continue }
			r.Differences=append(r.Differences,Difference{Key:k,Expected:want,Actual:act[k],Allowed:false});continue
		}
		if exp[k]!=act[k]{r.Differences=append(r.Differences,Difference{Key:k,Expected:exp[k],Actual:act[k],Allowed:false})}
	}
	for _,d:=range r.Differences{if !d.Allowed{r.Pass=false}}
	if !r.Pass{r.FailureCode="UNAUTHORIZED_SEMANTIC_DRIFT"}
	return r
}

func ValidateBuild(candidate CandidateManifest, build BuildManifest) error {
	if build.Schema != BuildSchemaR1 { return fmt.Errorf("unexpected build schema %q", build.Schema) }
	if build.CandidateID != candidate.ID { return fmt.Errorf("candidate id mismatch") }
	if candidate.ProgramSHA256 != "" && build.ProgramSHA256 != candidate.ProgramSHA256 { return fmt.Errorf("program sha mismatch") }
	if candidate.PayloadSHA256 != "" && build.PayloadSHA256 != candidate.PayloadSHA256 { return fmt.Errorf("payload sha mismatch") }
	if build.ArtifactSHA256 == "" || build.ArtifactBytes <= 0 { return fmt.Errorf("artifact provenance incomplete") }
	return nil
}

func factMap(f []SemanticFact) map[string]string { m:=map[string]string{}; for _,x:=range f{m[x.Key]=x.Value}; return m }
