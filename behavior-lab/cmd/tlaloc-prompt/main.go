package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tlaloc.local/behaviorlab/internal/promptgenome"
)

func main(){
	if len(os.Args)<2||os.Args[1]!="compile"{usage();os.Exit(2)}
	fs:=flag.NewFlagSet("compile",flag.ExitOnError);genomePath:=fs.String("genome","behavior-lab/profiles/prompt-genome-r1.json","prompt genome json");model:=fs.String("model","","model id/family");relevant:=fs.String("relevant","","comma-separated optional module ids");maxChars:=fs.Int("max-chars",0,"compiled prompt character budget; 0 means unlimited");outText:=fs.String("out-text","","optional text output path");outJSON:=fs.String("out-json","","optional manifest output path");fs.Parse(os.Args[2:])
	var g promptgenome.Genome;readJSON(*genomePath,&g);compiled,err:=promptgenome.Compile(promptgenome.CompileRequest{Genome:g,Model:*model,RelevantModules:splitCSV(*relevant),MaxChars:*maxChars});die(err)
	if *outText!=""{die(os.WriteFile(*outText,[]byte(compiled.Prompt+"\n"),0o600))}
	if *outJSON!=""{b,err:=json.MarshalIndent(compiled,"","  ");die(err);die(os.WriteFile(*outJSON,append(b,'\n'),0o600))}
	b,err:=json.MarshalIndent(compiled,"","  ");die(err);fmt.Println(string(b))
}

func splitCSV(s string)[]string{out:=[]string{};for _,v:=range strings.Split(s,","){if v=strings.TrimSpace(v);v!=""{out=append(out,v)}};return out}
func readJSON(path string,v any){b,err:=os.ReadFile(path);die(err);die(json.Unmarshal(b,v))}
func die(err error){if err!=nil{fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}}
func usage(){fmt.Fprintln(os.Stderr,"usage: tlaloc-prompt compile [-genome file] [-model id] [-relevant a,b] [-max-chars N] [-out-text file] [-out-json file]")}
