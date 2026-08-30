package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tlaloc.local/behaviorlab/internal/promotion"
	"tlaloc.local/behaviorlab/internal/target"
)

type ModelConfig struct {
	Name      string  `json:"name"`
	BaseURL   string  `json:"base_url"`
	Model     string  `json:"model"`
	APIKeyEnv string  `json:"api_key_env,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type Config struct {
	Schema           string        `json:"schema"`
	Carrier          string        `json:"carrier"`
	MasterPrompt     string        `json:"master_prompt"`
	OrigamiEvaluator string        `json:"origami_evaluator"`
	OrigamiBinary    string        `json:"origami_binary,omitempty"`
	StoreDir         string        `json:"store_dir,omitempty"`
	Models           []ModelConfig `json:"models"`
	TrialsPerModel   int           `json:"trials_per_model"`
	RunToolLoop      bool          `json:"run_tool_loop"`
	ToolQuestion     string        `json:"tool_question,omitempty"`
	RoutingEvidence  string        `json:"routing_evidence"`
	OutputDir        string        `json:"output_dir"`
}

type TrialArtifact struct {
	Execution  promotion.TrialExecution `json:"execution"`
	Evaluation promotion.EvaluatorReport `json:"evaluation"`
	ToolLoop   *target.HybridResult       `json:"tool_loop,omitempty"`
}

func main(){
	configPath:=flag.String("config","","campaign config JSON");flag.Parse();if *configPath==""{die(fmt.Errorf("-config is required"))}
	body,err:=os.ReadFile(*configPath);die(err);var cfg Config;die(json.Unmarshal(body,&cfg));if cfg.Schema!="tlaloc.perception-campaign.r1.config"{die(fmt.Errorf("unexpected config schema %q",cfg.Schema))};if cfg.TrialsPerModel<=0{cfg.TrialsPerModel=3};if len(cfg.Models)<1{die(fmt.Errorf("at least one model required"))};if cfg.OutputDir==""{cfg.OutputDir="runs/perception-promotion-r1"};die(os.MkdirAll(cfg.OutputDir,0o755))
	carrier,err:=os.ReadFile(cfg.Carrier);die(err);master,err:=os.ReadFile(cfg.MasterPrompt);die(err);variants,err:=promotion.BuildTransportVariants(carrier);die(err);var routing promotion.RoutingEvidence;rb,err:=os.ReadFile(cfg.RoutingEvidence);die(err);die(json.Unmarshal(rb,&routing))
	jsonl,err:=os.Create(filepath.Join(cfg.OutputDir,"trials.jsonl"));die(err);defer jsonl.Close();writer:=bufio.NewWriter(jsonl);defer writer.Flush()
	evaluator:=promotion.OrigamiEvaluator{Binary:cfg.OrigamiEvaluator,Carrier:cfg.Carrier};records:=[]promotion.TrialRecord{}
	for _,mc:=range cfg.Models{
		if mc.Name==""{mc.Name=mc.Model};client:=target.OpenAICompat{BaseURL:mc.BaseURL,Model:mc.Model,Temperature:mc.Temperature};if mc.APIKeyEnv!=""{client.APIKey=os.Getenv(mc.APIKeyEnv)}
		toolLoopDone:=false
		for trial:=1;trial<=cfg.TrialsPerModel;trial++{
			for _,variant:=range variants{
				ctx,cancel:=context.WithTimeout(context.Background(),5*time.Minute);execution,err:=promotion.RunPerceptionTrial(ctx,client,mc.Name,trial,variant,string(master),true);cancel();if err!=nil{fmt.Fprintf(os.Stderr,"trial %s/%d/%s failed: %v\n",mc.Name,trial,variant.Name,err);continue};ctx,cancel=context.WithTimeout(context.Background(),time.Minute);evaluation,err:=evaluator.Evaluate(ctx,execution.Observation);cancel();if err!=nil{fmt.Fprintf(os.Stderr,"evaluate %s/%d/%s failed: %v\n",mc.Name,trial,variant.Name,err);continue}
				artifact:=TrialArtifact{Execution:execution,Evaluation:evaluation};record:=promotion.TrialRecord{Model:mc.Name,Trial:trial,Transport:variant.Name,EvidenceKind:"REAL_MODEL",Evaluation:evaluation}
				if cfg.RunToolLoop&&!toolLoopDone&&variant.Name=="original"&&cfg.OrigamiBinary!=""&&cfg.StoreDir!=""{
					question:=cfg.ToolQuestion;if question==""{question="Boot the supplied Origami carrier using the visual probes, use the declared Origami tools to verify the bound store, then answer with a concise confirmation and one verified fact from the memory plane."};executor:=target.FixedOrigamiExecutor{OrigamiBinary:cfg.OrigamiBinary,Carrier:cfg.Carrier,StoreDir:cfg.StoreDir};ctx,cancel:=context.WithTimeout(context.Background(),8*time.Minute);hybrid,toolErr:=client.CompleteHybrid(ctx,target.HybridInput{SystemPrompt:string(master),Question:question,ImagePNG:carrier,Tools:target.OrigamiFixedTools(),Executor:executor,MaxTurns:16});cancel();if toolErr==nil{artifact.ToolLoop=&hybrid;record.ToolLoopPassed=hybrid.ToolCalls>0&&hybrid.Answer!="";record.ToolCalls=hybrid.ToolCalls;record.AnswerPresent=hybrid.Answer!="";if record.ToolLoopPassed{toolLoopDone=true}}else{fmt.Fprintf(os.Stderr,"tool loop %s failed: %v\n",mc.Name,toolErr)}
				}
				record.Evaluation=evaluation;records=append(records,record);line,_:=json.Marshal(artifact);_,_ = writer.Write(append(line,'\n'));_ = writer.Flush()
			}
		}
	}
	report,err:=promotion.Evaluate(records,routing,promotion.DefaultPolicy());die(err);out,err:=json.MarshalIndent(report,"","  ");die(err);out=append(out,'\n');die(os.WriteFile(filepath.Join(cfg.OutputDir,"campaign-report.json"),out,0o644));fmt.Print(string(out))
}

func die(err error){if err!=nil{fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}}
