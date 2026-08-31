package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tlaloc.local/behaviorlab/internal/candidateprepare"
	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningcycle"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/outcomelearner"
	"tlaloc.local/behaviorlab/internal/promptgenome"
)

func main(){
	if len(os.Args)<2{usage();os.Exit(2)}
	switch os.Args[1]{
	case "status":status(os.Args[2:])
	case "plan":plan(os.Args[2:])
	case "export-origami-spec":exportOrigamiSpec(os.Args[2:])
	case "prepare-candidate":prepareCandidate(os.Args[2:])
	case "validate-parity":validateParity(os.Args[2:])
	case "assess-outcome":assessOutcome(os.Args[2:])
	default:usage();os.Exit(2)
	}
}

func status(args []string){
	fs:=flag.NewFlagSet("status",flag.ExitOnError);root:=fs.String("store","","learning memory root");genomePath:=fs.String("genome","behavior-lab/profiles/prompt-genome-r1.json","prompt genome json");fs.Parse(args)
	store:=learningmemory.New(*root);events,err:=store.LoadAll();die(err);g:=loadGenome(*genomePath);writeJSON(learningcycle.BuildStatusWithGenome(store.Root,events,g))
}

func plan(args []string){
	fs:=flag.NewFlagSet("plan",flag.ExitOnError);root:=fs.String("store","","learning memory root");genomePath:=fs.String("genome","behavior-lab/profiles/prompt-genome-r1.json","prompt genome json");baseline:=fs.String("baseline","baseline","baseline candidate id");program:=fs.String("program-sha","","canonical program sha256");payload:=fs.String("payload-sha","","exact payload sha256");budget:=fs.Int("budget",3,"candidate budget");record:=fs.Bool("record-attempts",false,"persist CHANGE_ATTEMPT events for generated candidates");fs.Parse(args)
	store:=learningmemory.New(*root);events,err:=store.LoadAll();die(err);g:=loadGenome(*genomePath);p:=learningcycle.BuildPlanWithGenome(store.Root,events,g,*baseline,*program,*payload,*budget);die(learningcycle.ValidatePlan(p))
	if *record {
		for _,c:=range p.Candidates{
			parents:=c.ParentEvidenceIDs;if len(parents)==0{parents=p.Status.Policy.ParentEvidenceIDs};if len(parents)==0{continue}
			tags:=[]string{"learning-cycle","target:"+p.Intent.MutableModule,"module:"+p.Intent.MutableModule}
			for _,m:=range c.Mutations{tags=append(tags,"mutation:"+strings.ToUpper(m.Kind))}
			e:=learningmemory.Event{Schema:learningmemory.EventSchema,EventType:learningmemory.EventChange,EvidenceClass:learningmemory.EvidenceManual,CandidateID:c.ID,ParentEventIDs:parents,ChangeSummary:"Learning-cycle candidate generated for "+p.Intent.MutableModule,Tags:tags}
			_,_,err:=store.Put(e);die(err)
		}
	}
	writeJSON(p)
}

func exportOrigamiSpec(args []string){
	fs:=flag.NewFlagSet("export-origami-spec",flag.ExitOnError);candidatePath:=fs.String("candidate","","tlaloc candidate manifest json");parentSHA:=fs.String("parent-sha","","parent PNG sha256");out:=fs.String("out","","optional Origami CandidateSpec output path");fs.Parse(args)
	if *candidatePath==""{die(fmt.Errorf("-candidate is required"))}
	var c experimentpolicy.CandidateManifest;readJSON(*candidatePath,&c);spec,err:=experimentpolicy.ToOrigamiSpec(c,*parentSHA);die(err)
	if *out!=""{b,err:=json.MarshalIndent(spec,"","  ");die(err);die(os.WriteFile(*out,append(b,'\n'),0o600))}
	writeJSON(spec)
}

func prepareCandidate(args []string){
	fs:=flag.NewFlagSet("prepare-candidate",flag.ExitOnError);candidatePath:=fs.String("candidate","","tlaloc candidate manifest json");parent:=fs.String("parent","","validated parent PNG");builder:=fs.String("builder","origami-candidate-build","Origami candidate builder executable");inheritedPath:=fs.String("inherited-mutations","","optional JSON array of mutations already represented by parent");outDir:=fs.String("out-dir","","candidate preparation output directory");fs.Parse(args)
	if *candidatePath==""||*parent==""{die(fmt.Errorf("-candidate and -parent are required"))}
	var c experimentpolicy.CandidateManifest;readJSON(*candidatePath,&c)
	inherited:=[]experimentpolicy.OrigamiMutation{};if *inheritedPath!=""{readJSON(*inheritedPath,&inherited)}
	r,err:=candidateprepare.Prepare(candidateprepare.Request{Builder:*builder,ParentPNG:*parent,InheritedMutations:inherited,Candidate:c,OutputDir:*outDir});if err!=nil{writeJSON(r);die(err)};writeJSON(r)
}

func validateParity(args []string){
	fs:=flag.NewFlagSet("validate-parity",flag.ExitOnError);candidatePath:=fs.String("candidate","","candidate manifest json");expectedPath:=fs.String("expected","","expected semantic manifest json");buildPath:=fs.String("build","","Origami build manifest json");fs.Parse(args)
	if *candidatePath==""||*expectedPath==""||*buildPath==""{die(fmt.Errorf("-candidate, -expected and -build are required"))}
	var c experimentpolicy.CandidateManifest;readJSON(*candidatePath,&c);var e experimentpolicy.SemanticManifest;readJSON(*expectedPath,&e);var b experimentpolicy.BuildManifest;readJSON(*buildPath,&b)
	if err:=experimentpolicy.ValidateBuild(c,b);err!=nil{writeJSON(experimentpolicy.ParityReport{Schema:experimentpolicy.ParitySchemaR1,CandidateID:c.ID,Pass:false,FailureCode:"INVALID_BUILD_MANIFEST"});os.Exit(1)}
	r:=experimentpolicy.CheckParity(c,e,b.VisibleSemantics);writeJSON(r);if !r.Pass{os.Exit(1)}
}

func assessOutcome(args []string){
	fs:=flag.NewFlagSet("assess-outcome",flag.ExitOnError);requestPath:=fs.String("request","","outcome request json");root:=fs.String("store","","learning memory root");record:=fs.Bool("record",false,"persist OUTCOME_LINK when parent ids are supplied");parents:=fs.String("parents","","comma-separated change-event and post-change evidence ids");fs.Parse(args)
	if *requestPath==""{die(fmt.Errorf("-request is required"))}
	var req outcomelearner.Request;readJSON(*requestPath,&req);assessment,knowledge:=outcomelearner.Assess(req)
	if *record {
		ids:=splitCSV(*parents);if len(ids)<2{die(fmt.Errorf("-parents requires change-event and post-change evidence ids"))}
		before,after,delta:=assessment.BeforeScore,assessment.AfterScore,assessment.Delta
		e:=learningmemory.Event{Schema:learningmemory.EventSchema,EventType:learningmemory.EventOutcome,EvidenceClass:learningmemory.EvidenceManual,CandidateID:req.After.CandidateID,ParentEventIDs:ids,BeforeScore:&before,AfterScore:&after,Delta:&delta,Tags:[]string{"outcome:"+assessment.Classification,"knowledge:"+knowledge.Action}}
		store:=learningmemory.New(*root);_,_,err:=store.Put(e);die(err)
	}
	writeJSON(map[string]any{"assessment":assessment,"knowledge_update":knowledge})
}

func loadGenome(path string)promptgenome.Genome{var g promptgenome.Genome;readJSON(path,&g);return g}
func splitCSV(s string)[]string{out:=[]string{};for _,v:=range strings.Split(s,","){if v=strings.TrimSpace(v);v!=""{out=append(out,v)}};return out}
func readJSON(path string,v any){b,err:=os.ReadFile(path);die(err);die(json.Unmarshal(b,v))}
func writeJSON(v any){b,err:=json.MarshalIndent(v,"","  ");die(err);fmt.Println(string(b))}
func die(err error){if err!=nil{fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}}
func usage(){fmt.Fprintln(os.Stderr,"usage: tlaloc-learn <status|plan|export-origami-spec|prepare-candidate|validate-parity|assess-outcome> [flags]")}
