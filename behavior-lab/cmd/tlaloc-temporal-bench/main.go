package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/temporalbench"
)

func main() {
	in := flag.String("in", "-", "benchmark campaign JSON or - for stdin")
	out := flag.String("out", "-", "benchmark result JSON or - for stdout")
	printDebug := flag.Bool("print-debug-instruction", false, "print the test-only observable diagnostic suffix and exit")
	printExample := flag.Bool("print-debug-example", false, "print one valid ORIGAMI_DEBUG_R0 footer example and exit")
	memoryRoot := flag.String("memory-store", "", "learning-memory root; defaults to XDG state directory")
	noMemory := flag.Bool("no-memory", false, "disable automatic persistence of real-model evidence")
	origamiVersion := flag.String("origami-version", "", "Origami version/profile label stored with evidence")
	tlalocVersion := flag.String("tlaloc-version", "", "Tlaloc version label stored with evidence")
	candidateID := flag.String("candidate-id", "", "candidate/change identifier stored with evidence")
	flag.Parse()
	if *printDebug { fmt.Println(temporalbench.DiagnosticInstruction()); return }
	if *printExample { fmt.Println(temporalbench.FormatDebugExample()); return }
	body, err := read(*in); die(err)
	var campaign temporalbench.Campaign
	dec := json.NewDecoder(bytes.NewReader(body)); dec.DisallowUnknownFields(); die(dec.Decode(&campaign))
	if campaign.Schema != temporalbench.CampaignSchema { die(fmt.Errorf("unexpected schema %q", campaign.Schema)) }
	if campaign.BenchmarkID == "" { die(fmt.Errorf("benchmark_id is required")) }
	result := temporalbench.EvaluateCampaign(campaign)
	encoded, err := json.MarshalIndent(result, "", "  "); die(err); encoded=append(encoded,'\n')
	if result.RealEvidence && !*noMemory {
		events,err:=learningmemory.ImportTemporalBenchmark(body,encoded,campaign,result,learningmemory.ImportOptions{OrigamiVersion:*origamiVersion,TlalocVersion:*tlalocVersion,CandidateID:*candidateID});die(err)
		if len(events)>0 { store:=learningmemory.New(*memoryRoot); _,_,_,err=store.PutAll(events);die(err) }
	}
	if *out=="-" { _,err=os.Stdout.Write(encoded); die(err); return }
	die(os.WriteFile(*out,encoded,0o644))
}

func read(path string)([]byte,error){ if path=="-"{return os.ReadFile("/dev/stdin")}; return os.ReadFile(path) }
func die(err error){ if err!=nil{fmt.Fprintln(os.Stderr,"error:",err); os.Exit(1)} }
