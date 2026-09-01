package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/lfm2boundary"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" { usage(); os.Exit(2) }
	fs:=flag.NewFlagSet("run",flag.ExitOnError)
	endpoint:=fs.String("endpoint","http://127.0.0.1:1234/v1","LM Studio OpenAI-compatible endpoint")
	model:=fs.String("model",lfm2boundary.RequiredModel,"exact model identifier")
	carrier:=fs.String("carrier","","canonical Origami temporal carrier PNG")
	candidate:=fs.String("candidate","","candidate PNG (t2-temporal-grammar-visible-r1)")
	populations:=fs.String("populations","1,2,4,8,16","worker populations")
	parallelisms:=fs.String("parallelisms","1,2,4","global parallelism sweep")
	replicas:=fs.Int("replicas",3,"replicas per visual observation")
	out:=fs.String("out","lfm2-boundary-output","output directory")
	r4:=fs.String("r4-prompt","","Master Prompt R4 path; default uses installed Origami")
	worker:=fs.String("worker","","tlaloc-lfm2-worker path; default is sibling binary")
	maxTokens:=fs.Int("max-tokens",256,"bounded generation tokens")
	timeout:=fs.Duration("timeout",90*time.Second,"per specialist process timeout")
	_ = fs.Parse(os.Args[2:])
	pops,err:=parseInts(*populations);if err!=nil{fail(err)};pars,err:=parseInts(*parallelisms);if err!=nil{fail(err)}
	if *carrier==""||*candidate==""{fail(fmt.Errorf("--carrier and --candidate are required"))}
	workerPath:=*worker;if workerPath==""{exe,err:=os.Executable();if err!=nil{fail(err)};workerPath=filepath.Join(filepath.Dir(exe),"tlaloc-lfm2-worker")}
	r4Path:=*r4;if r4Path==""{r4Path=defaultR4Path()};r4Body,err:=os.ReadFile(r4Path);if err!=nil{fail(fmt.Errorf("read Master Prompt R4 %s: %w",r4Path,err))}
	summary,err:=lfm2boundary.Run(context.Background(),lfm2boundary.Config{Endpoint:*endpoint,Model:*model,CarrierPath:*carrier,CandidatePath:*candidate,R4Prompt:string(r4Body),WorkerBinary:workerPath,OutputDir:*out,Populations:pops,Parallelisms:pars,Replicas:*replicas,MaxTokens:*maxTokens,Timeout:*timeout});if err!=nil{fail(err)}
	fmt.Printf("LFM2_BOUNDARY_RUNS=%d\n",summary.Runs)
	fmt.Printf("MAX_USEFUL_PARALLEL=%d\n",summary.MaxUsefulParallel)
	fmt.Printf("MILESTONE_SUCCESS=%t\n",summary.MilestoneSuccess)
	fmt.Printf("OUTPUT_DIR=%s\n",*out)
}

func parseInts(s string)([]int,error){parts:=strings.Split(s,",");out:=[]int{};seen:=map[int]bool{};for _,raw:=range parts{n,err:=strconv.Atoi(strings.TrimSpace(raw));if err!=nil||n<=0{return nil,fmt.Errorf("invalid positive integer %q",raw)};if!seen[n]{seen[n]=true;out=append(out,n)}};return out,nil}
func defaultR4Path()string{data:=strings.TrimSpace(os.Getenv("XDG_DATA_HOME"));if data==""{home,_:=os.UserHomeDir();data=filepath.Join(home,".local","share")};return filepath.Join(data,"origami","current","generated","MASTER_PROMPT.md")}
func usage(){fmt.Fprintln(os.Stderr,"Usage: tlaloc-lfm2-boundary run --carrier BASE.png --candidate CANDIDATE.png [--endpoint http://127.0.0.1:1234/v1] [--model lfm2-vl-1.6b] [--populations 1,2,4,8,16] [--parallelisms 1,2,4] [--replicas 3] [--out DIR]")}
func fail(err error){fmt.Fprintln(os.Stderr,"ERROR:",err);os.Exit(1)}
