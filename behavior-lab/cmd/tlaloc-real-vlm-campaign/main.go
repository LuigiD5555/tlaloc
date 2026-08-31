package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tlaloc.local/behaviorlab/internal/realcampaign"
)

func main(){
	if len(os.Args)<2{usage();os.Exit(2)}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	switch os.Args[1]{
	case "doctor": doctor(ctx,os.Args[2:])
	case "prepare": prepare(ctx,os.Args[2:])
	case "run": run(ctx,os.Args[2:])
	case "example": example()
	default: usage();os.Exit(2)
	}
}

func flags(name string,args []string)(realcampaign.Spec,*flag.FlagSet){
	fs:=flag.NewFlagSet(name,flag.ExitOnError)
	var s realcampaign.Spec
	fs.StringVar(&s.CampaignID,"id","origami-temporal-real-vlm-r0","campaign id")
	fs.StringVar(&s.Phase,"phase","SMOKE","SMOKE or EVIDENCE")
	fs.StringVar(&s.Endpoint,"endpoint","http://127.0.0.1:1234/v1","OpenAI-compatible base URL")
	fs.StringVar(&s.Model,"model","","model id; auto-selected only when endpoint reports exactly one model")
	fs.StringVar(&s.APIKeyEnv,"api-key-env","","environment variable containing API key")
	fs.StringVar(&s.Program,"program","","canonical Origami signal-chain TemporalProgram JSON")
	fs.StringVar(&s.TemporalCarrier,"carrier","origami-temporal-carrier","Origami temporal carrier executable")
	fs.StringVar(&s.CandidateBuilder,"builder","origami-candidate-build","Origami candidate builder executable")
	fs.StringVar(&s.OutputDir,"out","runs/real-vlm/origami-temporal-r0","campaign output directory")
	fs.StringVar(&s.MasterPrompt,"master-prompt","","optional Origami Master Prompt for R4_ASSISTED evidence phase")
	fs.Float64Var(&s.Temperature,"temperature",0,"target model temperature")
	fs.IntVar(&s.TimeoutSeconds,"timeout",180,"per-call timeout seconds")
	fs.IntVar(&s.TransportRetries,"transport-retries",1,"transport retries per call")
	fs.IntVar(&s.TrialsPerModel,"trials",0,"trials per model; defaults to 1 smoke / 3 evidence")
	fs.IntVar(&s.CandidatesPerGen,"candidates-per-generation",0,"candidate budget per generation")
	fs.IntVar(&s.MaxGenerations,"generations",0,"maximum experimental generations")
	fs.Parse(args)
	return s,fs
}

func doctor(ctx context.Context,args []string){s,_:=flags("doctor",args);r,err:=realcampaign.Doctor(ctx,s);die(err);write(r)}
func prepare(ctx context.Context,args []string){s,_:=flags("prepare",args);r,err:=realcampaign.Prepare(ctx,s);die(err);write(r)}
func run(ctx context.Context,args []string){s,_:=flags("run",args);prepared,report,err:=realcampaign.Run(ctx,s);die(err);write(map[string]any{"prepared":prepared,"report":report})}

func example(){
	write(realcampaign.Spec{Schema:realcampaign.SpecSchema,CampaignID:"origami-temporal-real-vlm-r0",Phase:realcampaign.PhaseSmoke,Endpoint:"http://127.0.0.1:1234/v1",Program:"/path/to/origami/experiments/temporal-automaton-r0/signal-chain.json",TemporalCarrier:"origami-temporal-carrier",CandidateBuilder:"origami-candidate-build",OutputDir:"runs/real-vlm/origami-temporal-r0",TimeoutSeconds:180,TransportRetries:1})
}
func write(v any){b,err:=json.MarshalIndent(v,"","  ");die(err);fmt.Println(string(b))}
func die(err error){if err!=nil{fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}}
func usage(){fmt.Fprintln(os.Stderr,"usage: tlaloc-real-vlm-campaign <doctor|prepare|run|example> [flags]")}
