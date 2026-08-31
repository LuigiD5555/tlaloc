package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/temporalbench"
)

func main() {
	in := flag.String("in", "-", "benchmark campaign JSON or - for stdin")
	out := flag.String("out", "-", "benchmark result JSON or - for stdout")
	flag.Parse()
	body, err := read(*in); die(err)
	var campaign temporalbench.Campaign
	dec := json.NewDecoder(bytes.NewReader(body)); dec.DisallowUnknownFields(); die(dec.Decode(&campaign))
	if campaign.Schema != temporalbench.CampaignSchema { die(fmt.Errorf("unexpected schema %q", campaign.Schema)) }
	if campaign.BenchmarkID == "" { die(fmt.Errorf("benchmark_id is required")) }
	result := temporalbench.EvaluateCampaign(campaign)
	encoded, err := json.MarshalIndent(result, "", "  "); die(err); encoded=append(encoded,'\n')
	if *out=="-" { _,err=os.Stdout.Write(encoded); die(err); return }
	die(os.WriteFile(*out,encoded,0o644))
}

func read(path string)([]byte,error){ if path=="-"{return os.ReadFile("/dev/stdin")}; return os.ReadFile(path) }
func die(err error){ if err!=nil{fmt.Fprintln(os.Stderr,"error:",err); os.Exit(1)} }
