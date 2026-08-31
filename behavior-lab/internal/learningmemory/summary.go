package learningmemory

import (
	"sort"
	"strings"
)

func BuildSummary(root string, events []Event) Summary {
	s:=Summary{Schema:"tlaloc.learning-memory.r0.summary",StoreRoot:root,TotalEvents:len(events)}
	type acc struct{ p FailurePattern; models,specimens,questions map[string]bool }
	patterns:=map[string]*acc{}
	type outAcc struct{ n int; sum,best,worst float64; init bool }
	outcomes:=map[string]*outAcc{}
	for _,e:=range events{
		switch e.EventType{
		case EventObservation:
			s.ObservationEvents++
			if e.EvidenceClass==EvidenceRealModel{s.RealModelObservations++}
			if e.EvidenceClass==EvidenceSynthetic{s.SyntheticObservations++}
			if e.Pass!=nil&&*e.Pass{s.PassedObservations++}else{s.FailedObservations++}
			if e.EvidenceClass!=EvidenceRealModel||e.Pass==nil||*e.Pass{continue}
			stage:=strings.TrimSpace(e.LastCompletedStage);if stage==""{stage="UNKNOWN_STAGE"}
			failure:=strings.TrimSpace(e.FailureCode);if failure==""{failure="BENCHMARK_ASSERTION_FAILED"}
			layer:=strings.TrimSpace(e.ScoreLayer);if layer==""{layer="UNKNOWN_LAYER"}
			key:=stage+"|"+failure+"|"+layer
			a:=patterns[key];if a==nil{a=&acc{p:FailurePattern{Key:key,Stage:stage,FailureCode:failure,ScoreLayer:layer,SuggestedTarget:suggestedTarget(failure,stage)},models:map[string]bool{},specimens:map[string]bool{},questions:map[string]bool{}};patterns[key]=a}
			a.p.Count++;if e.ModelID!=""{a.models[e.ModelID]=true};if e.SpecimenID!=""{a.specimens[e.SpecimenID]=true};if e.QuestionID!=""{a.questions[e.QuestionID]=true}
		case EventChange:s.ChangeAttempts++
		case EventOutcome:
			s.OutcomeLinks++
			if e.Delta==nil&&e.BeforeScore!=nil&&e.AfterScore!=nil{d:=*e.AfterScore-*e.BeforeScore;e.Delta=&d}
			if e.Delta==nil||e.CandidateID==""{continue}
			a:=outcomes[e.CandidateID];if a==nil{a=&outAcc{};outcomes[e.CandidateID]=a};v:=*e.Delta;a.n++;a.sum+=v;if !a.init||v>a.best{a.best=v};if !a.init||v<a.worst{a.worst=v};a.init=true
		}
	}
	for _,a:=range patterns{a.p.Models=sortedSet(a.models);a.p.Specimens=sortedSet(a.specimens);a.p.Questions=sortedSet(a.questions);s.TopRealFailurePatterns=append(s.TopRealFailurePatterns,a.p)}
	sort.Slice(s.TopRealFailurePatterns,func(i,j int)bool{if s.TopRealFailurePatterns[i].Count!=s.TopRealFailurePatterns[j].Count{return s.TopRealFailurePatterns[i].Count>s.TopRealFailurePatterns[j].Count};return s.TopRealFailurePatterns[i].Key<s.TopRealFailurePatterns[j].Key})
	if len(s.TopRealFailurePatterns)>0{s.NextDebugTarget=s.TopRealFailurePatterns[0].SuggestedTarget}
	for id,a:=range outcomes{s.CandidateOutcomes=append(s.CandidateOutcomes,CandidateOutcome{CandidateID:id,Outcomes:a.n,MeanDelta:a.sum/float64(a.n),BestDelta:a.best,WorstDelta:a.worst})}
	sort.Slice(s.CandidateOutcomes,func(i,j int)bool{return s.CandidateOutcomes[i].CandidateID<s.CandidateOutcomes[j].CandidateID})
	return s
}

func suggestedTarget(failure,stage string)string{
	switch strings.ToUpper(failure){
	case "NO_VISUAL_SIGNAL":return "VISUAL_SIGNAL"
	case "BOOT_NOT_FOUND":return "BOOT"
	case "ROSETTA_NOT_FOUND":return "ROSETTA"
	case "CODEC_NOT_FOUND":return "CODEC_REGISTRY"
	case "CAPABILITY_MISMATCH":return "CAPABILITY_FALLBACK"
	case "T2_NOT_FOUND":return "T2_NAVIGATION"
	case "SEMANTIC_EVIDENCE_INSUFFICIENT":return "SEMANTIC_LAYOUT"
	case "TEMPORAL_RULE_AMBIGUOUS":return "TEMPORAL_GRAMMAR"
	case "TEMPORAL_EXECUTION_INCOMPLETE":return "EXECUTION_POLICY_COMPLIANCE"
	case "RULE_FIRING_PRECONDITION_VIOLATION", "EXECUTION_SEMANTICS_CONTRADICTION", "CROSS_MODEL_EXECUTION_FIDELITY_FAILED":return "SYNCHRONOUS_EXECUTION_FIDELITY"
	case "CELL_IDENTITY_CONFUSION":return "CELL_IDENTITY_ENCODING"
	case "FROM_STATE_PRECONDITION_CONFUSION":return "FROM_STATE_PRECONDITION_VISIBILITY"
	case "CONDITION_TARGET_BINDING_CONFUSION":return "RULE_ROLE_BINDING"
	case "CHECKPOINT_NOT_FOUND":return "TEMPORAL_ROUTING"
	case "ARTIFACT_GENERATION_REGRESSION", "UNAUTHORIZED_SEMANTIC_DRIFT":return "SEMANTIC_PARITY_GATE"
	case "UNSUPPORTED_OPERATION":return "CAPABILITY_PROFILE"
	}
	if strings.EqualFold(stage,"ROSETTA"){return "ROSETTA_TO_T2_ROUTE"}
	if strings.EqualFold(stage,"TEMPORAL_ROUTE")||strings.EqualFold(stage,"TEMPORAL_STEP"){return "TEMPORAL_PROGRAM"}
	return "BENCHMARK_FAILURE_ANALYSIS"
}

func sortedSet(m map[string]bool)[]string{out:=make([]string,0,len(m));for k:=range m{out=append(out,k)};sort.Strings(out);return out}
