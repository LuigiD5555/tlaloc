package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/experimentpolicy"
	"tlaloc.local/behaviorlab/internal/learningcycle"
	"tlaloc.local/behaviorlab/internal/learningmemory"
)

func main(){
	if len(os.Args)<2{usage();os.Exit(2)}
	switch os.Args[1]{
	case "status":status(os.Args[2:])
	case "plan":plan(os.Args[2:])
	case "validate-parity":validateParity(os.Args[2:])
	default:usage();os.Exit(2)
	}
}

func status(args []string){
	fs:=flag.NewFlagSet("status",flag.ExitOnError);root:=fs.String("store","","learning memory root");fs.Parse(args)
	store:=learningmemory.New(*root);events,err:=store.LoadAll();die(err);writeJSON(learningcycle.BuildStatus(store.Root,events))
}

func plan(args []string){
	fs:=flag.NewFlagSet("plan",flag.ExitOnError);root:=fs.String("store","","learning memory root");baseline:=fs.String("baseline","baseline","baseline candidate id");program:=fs.String("program-sha","","canonical program sha256");payload:=fs.String("payload-sha","","exact payload sha256");budget:=fs.Int("budget",3,"candidate budget");fs.Parse(args)
	store:=learningmemory.New(*root);events,err:=store.LoadAll();die(err);p:=learningcycle.BuildPlan(store.Root,events,*baseline,*program,*payload,*budget);die(learningcycle.ValidatePlan(p));writeJSON(p)
}

func validateParity(args []string){
	fs:=flag.NewFlagSet("validate-parity",flag.ExitOnError);candidatePath:=fs.String("candidate","","candidate manifest json");expectedPath:=fs.String("expected","","expected semantic manifest json");buildPath:=fs.String("build","","Origami build manifest json");fs.Parse(args)
	if *candidatePath==""||*expectedPath==""||*buildPath==""{die(fmt.Errorf("-candidate, -expected and -build are required"))}
	var c experimentpolicy.CandidateManifest;readJSON(*candidatePath,&c);var e experimentpolicy.SemanticManifest;readJSON(*expectedPath,&e);var b experimentpolicy.BuildManifest;readJSON(*buildPath,&b)
	if err:=experimentpolicy.ValidateBuild(c,b);err!=nil{writeJSON(experimentpolicy.ParityReport{Schema:experimentpolicy.ParitySchemaR1,CandidateID:c.ID,Pass:false,FailureCode:"INVALID_BUILD_MANIFEST"});os.Exit(1)}
	r:=experimentpolicy.CheckParity(c,e,b.VisibleSemantics);writeJSON(r);if !r.Pass{os.Exit(1)}
}

func readJSON(path string,v any){b,err:=os.ReadFile(path);die(err);die(json.Unmarshal(b,v))}
func writeJSON(v any){b,err:=json.MarshalIndent(v,"","  ");die(err);fmt.Println(string(b))}
func die(err error){if err!=nil{fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}}
func usage(){fmt.Fprintln(os.Stderr,"usage: tlaloc-learn <status|plan|validate-parity> [flags]")}
