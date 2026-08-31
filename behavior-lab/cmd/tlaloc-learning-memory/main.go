package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/temporalbench"
)

func main(){
	if len(os.Args)<2{usage();os.Exit(2)}
	switch os.Args[1]{
	case "ingest-benchmark":ingestBenchmark(os.Args[2:])
	case "summary":summary(os.Args[2:])
	case "record-change":recordChange(os.Args[2:])
	case "record-outcome":recordOutcome(os.Args[2:])
	case "events":events(os.Args[2:])
	default:usage();os.Exit(2)
	}
}

func ingestBenchmark(args []string){
	fs:=flag.NewFlagSet("ingest-benchmark",flag.ExitOnError);campaignPath:=fs.String("campaign","","benchmark campaign JSON");resultPath:=fs.String("result","","benchmark result JSON");root:=fs.String("store","","memory root");origami:=fs.String("origami-version","","Origami version/profile label");tlaloc:=fs.String("tlaloc-version","","Tlaloc version label");candidate:=fs.String("candidate-id","","candidate/change identifier");includeSynthetic:=fs.Bool("include-synthetic",false,"also store synthetic evaluator evidence");fs.Parse(args)
	if *campaignPath==""||*resultPath==""{die(fmt.Errorf("-campaign and -result are required"))}
	cb:=mustRead(*campaignPath);rb:=mustRead(*resultPath)
	var c temporalbench.Campaign;decodeStrict(cb,&c);var r temporalbench.Result;decodeStrict(rb,&r)
	evs,err:=learningmemory.ImportTemporalBenchmark(cb,rb,c,r,learningmemory.ImportOptions{OrigamiVersion:*origami,TlalocVersion:*tlaloc,CandidateID:*candidate,IncludeSynthetic:*includeSynthetic});die(err)
	store:=learningmemory.New(*root);added,skipped,_,err:=store.PutAll(evs);die(err)
	writeJSON(map[string]any{"store_root":store.Root,"events_considered":len(evs),"added":added,"already_present":skipped})
}

func summary(args []string){fs:=flag.NewFlagSet("summary",flag.ExitOnError);root:=fs.String("store","","memory root");fs.Parse(args);store:=learningmemory.New(*root);evs,err:=store.LoadAll();die(err);writeJSON(learningmemory.BuildSummary(store.Root,evs))}

func events(args []string){fs:=flag.NewFlagSet("events",flag.ExitOnError);root:=fs.String("store","","memory root");fs.Parse(args);store:=learningmemory.New(*root);evs,err:=store.LoadAll();die(err);writeJSON(evs)}

func recordChange(args []string){
	fs:=flag.NewFlagSet("record-change",flag.ExitOnError);root:=fs.String("store","","memory root");candidate:=fs.String("candidate-id","","candidate id");text:=fs.String("summary","","change/hypothesis summary");parents:=fs.String("parents","","comma-separated parent evidence event ids");recorded:=fs.String("recorded-at","","optional RFC3339 timestamp");fs.Parse(args)
	e:=learningmemory.Event{Schema:learningmemory.EventSchema,EventType:learningmemory.EventChange,EvidenceClass:learningmemory.EvidenceManual,RecordedAt:*recorded,CandidateID:*candidate,ChangeSummary:*text,ParentEventIDs:splitCSV(*parents)}
	store:=learningmemory.New(*root);added,stored,err:=store.Put(e);die(err);writeJSON(map[string]any{"added":added,"event":stored})
}

func recordOutcome(args []string){
	fs:=flag.NewFlagSet("record-outcome",flag.ExitOnError);root:=fs.String("store","","memory root");candidate:=fs.String("candidate-id","","candidate id");parents:=fs.String("parents","","comma-separated change event + post-change evidence ids");beforeS:=fs.String("before","","score before change");afterS:=fs.String("after","","score after change");recorded:=fs.String("recorded-at","","optional RFC3339 timestamp");fs.Parse(args)
	before,err:=strconv.ParseFloat(*beforeS,64);die(err);after,err:=strconv.ParseFloat(*afterS,64);die(err);delta:=after-before
	e:=learningmemory.Event{Schema:learningmemory.EventSchema,EventType:learningmemory.EventOutcome,EvidenceClass:learningmemory.EvidenceManual,RecordedAt:*recorded,CandidateID:*candidate,ParentEventIDs:splitCSV(*parents),BeforeScore:&before,AfterScore:&after,Delta:&delta}
	store:=learningmemory.New(*root);added,stored,err:=store.Put(e);die(err);writeJSON(map[string]any{"added":added,"event":stored})
}

func decodeStrict(body []byte,v any){dec:=json.NewDecoder(bytes.NewReader(body));dec.DisallowUnknownFields();die(dec.Decode(v))}
func mustRead(path string)[]byte{b,err:=os.ReadFile(path);die(err);return b}
func splitCSV(s string)[]string{out:=[]string{};for _,v:=range strings.Split(s,","){if v=strings.TrimSpace(v);v!=""{out=append(out,v)}};return out}
func writeJSON(v any){b,err:=json.MarshalIndent(v,"","  ");die(err);fmt.Println(string(b))}
func usage(){fmt.Fprintln(os.Stderr,"usage: tlaloc-learning-memory <ingest-benchmark|summary|events|record-change|record-outcome> [flags]")}
func die(err error){if err!=nil{fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}}
