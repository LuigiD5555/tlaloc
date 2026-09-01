package lfm2boundary

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/temporalbench"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const CampaignSchema = "tlaloc.lfm2-boundary.r0"

var DefaultConditions = []string{"NATIVE_PNG_ONLY", "R4_ASSISTED", "BLACKBOARD_CROPPED"}

type Config struct {
	Endpoint      string
	Model         string
	CarrierPath   string
	CandidatePath string
	R4Prompt      string
	WorkerBinary  string
	OutputDir     string
	Populations   []int
	Parallelisms  []int
	Replicas      int
	MaxTokens     int
	Timeout       time.Duration
}

type SpecimenInfo struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	PNGBytes int    `json:"png_bytes"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type Manifest struct {
	Schema       string          `json:"schema"`
	CreatedAt    string          `json:"created_at"`
	Endpoint     string          `json:"endpoint"`
	Model        string          `json:"model"`
	Context      int             `json:"context"`
	Temperature float64         `json:"temperature"`
	Populations  []int           `json:"populations"`
	Parallelisms []int           `json:"parallelisms"`
	Replicas     int             `json:"replicas"`
	Conditions   []string        `json:"conditions"`
	CandidateID  string          `json:"candidate_id"`
	Preflight    PreflightResult `json:"preflight"`
	Specimens    []SpecimenInfo  `json:"specimens"`
	Crops        []CropSpec      `json:"crops"`
}

type Metric struct {
	RunID               string  `json:"run_id"`
	SpecimenID          string  `json:"specimen_id"`
	Condition           string  `json:"condition"`
	Population          int     `json:"population"`
	MaxParallel         int     `json:"max_parallel"`
	Replicas            int     `json:"replicas"`
	Score                float64 `json:"score"`
	PPerception          float64 `json:"p_perception"`
	RProtocol            float64 `json:"r_protocol"`
	SSemantic            float64 `json:"s_semantic"`
	TTemporal            float64 `json:"t_temporal"`
	XExactness           float64 `json:"x_exactness"`
	InventedExactClaims  int     `json:"invented_exact_claims"`
	ValidResponses       int     `json:"valid_responses"`
	ExpectedResponses    int     `json:"expected_responses"`
	ValidResponseRate    float64 `json:"valid_response_rate"`
	ErrorCount           int     `json:"error_count"`
	HTTPErrorCount       int     `json:"http_error_count"`
	PromptTokens         int     `json:"prompt_tokens"`
	CompletionTokens     int     `json:"completion_tokens"`
	LatencyP50MS         int64   `json:"latency_p50_ms"`
	LatencyP95MS         int64   `json:"latency_p95_ms"`
	ThroughputPerSecond  float64 `json:"throughput_per_second"`
	PeakParallel         int     `json:"peak_parallel"`
	DurationMS           int64   `json:"duration_ms"`
	Useful               bool    `json:"useful"`
}

type Summary struct {
	Schema               string         `json:"schema"`
	Runs                 int            `json:"runs"`
	BoundaryByPopulation map[string]int `json:"boundary_by_population"`
	MaxUsefulParallel    int            `json:"max_useful_parallel"`
	MilestoneSuccess     bool           `json:"milestone_success"`
	Acceptance           string         `json:"acceptance"`
}

func Run(ctx context.Context, cfg Config) (Summary, error) {
	if cfg.Endpoint == "" { cfg.Endpoint = "http://127.0.0.1:1234/v1" }
	if cfg.Model == "" { cfg.Model = RequiredModel }
	if cfg.Replicas <= 0 { cfg.Replicas = 3 }
	if cfg.MaxTokens <= 0 { cfg.MaxTokens = 256 }
	if cfg.Timeout <= 0 { cfg.Timeout = 90*time.Second }
	if len(cfg.Populations) == 0 { cfg.Populations = []int{1,2,4,8,16} }
	if len(cfg.Parallelisms) == 0 { cfg.Parallelisms = []int{1,2,4} }
	if cfg.OutputDir == "" || cfg.WorkerBinary == "" || cfg.CarrierPath == "" || cfg.CandidatePath == "" { return Summary{}, fmt.Errorf("output dir, worker binary, carrier and candidate are required") }
	if strings.TrimSpace(cfg.R4Prompt) == "" { return Summary{}, fmt.Errorf("Master Prompt R4 is required for R4_ASSISTED") }
	for _, n := range append(append([]int{}, cfg.Populations...), cfg.Parallelisms...) { if n <= 0 { return Summary{}, fmt.Errorf("population and parallelism values must be positive") } }

	preflight, err := Preflight(ctx, cfg.Endpoint, cfg.Model); if err != nil { return Summary{}, err }
	if err := os.MkdirAll(filepath.Join(cfg.OutputDir,"local"), 0o700); err != nil { return Summary{}, err }
	if err := os.WriteFile(filepath.Join(cfg.OutputDir,".gitignore"), []byte("local/\n"), 0o600); err != nil { return Summary{}, err }

	specimens := []SpecimenInfo{}
	for id, path := range map[string]string{"base":cfg.CarrierPath,"candidate":cfg.CandidatePath} {
		s, err := inspectSpecimen(id, path); if err != nil { return Summary{}, err }; specimens = append(specimens,s)
	}
	sort.Slice(specimens,func(i,j int)bool{return specimens[i].ID<specimens[j].ID})
	cropPaths := map[string]map[string]string{}
	for _, s := range specimens {
		paths, err := WriteDeclaredCrops(s.Path, filepath.Join(cfg.OutputDir,"local","crops",s.ID)); if err != nil { return Summary{}, err }
		cropPaths[s.ID] = paths
	}
	manifest := Manifest{Schema:CampaignSchema, CreatedAt:time.Now().UTC().Format(time.RFC3339), Endpoint:cfg.Endpoint, Model:cfg.Model, Context:RequiredContext, Temperature:0, Populations:cfg.Populations, Parallelisms:cfg.Parallelisms, Replicas:cfg.Replicas, Conditions:DefaultConditions, CandidateID:"t2-temporal-grammar-visible-r1", Preflight:preflight, Specimens:specimens, Crops:DeclaredCrops()}
	if err := writeJSON(filepath.Join(cfg.OutputDir,"manifest.json"),manifest); err != nil{return Summary{},err}
	if err := writeHashes(cfg,specimens); err != nil{return Summary{},err}

	store := blackboard.New(filepath.Join(cfg.OutputDir,"local","blackboard"))
	metrics := []Metric{}
	for _, specimen := range specimens {
		for _, condition := range DefaultConditions {
			for _, population := range cfg.Populations {
				for _, parallel := range cfg.Parallelisms {
					metric, report, err := runCell(ctx,cfg,store,specimen,cropPaths[specimen.ID],condition,population,parallel)
					metrics=append(metrics,metric)
					_ = writeJSON(filepath.Join(cfg.OutputDir,"local","responses",metric.RunID+".json"),report)
					if err != nil { /* preserve the failed cell and continue the sweep */ }
				}
			}
		}
	}
	markUseful(metrics)
	if err := writeMetricsCSV(filepath.Join(cfg.OutputDir,"metrics.csv"),metrics); err != nil{return Summary{},err}
	summary:=summarize(metrics,cfg.Populations,cfg.Parallelisms)
	if err:=writeJSON(filepath.Join(cfg.OutputDir,"summary.json"),summary);err!=nil{return Summary{},err}
	return summary,nil
}

func runCell(ctx context.Context,cfg Config,store blackboard.Store,specimen SpecimenInfo,crops map[string]string,condition string,population,parallel int)(Metric,tlaloque.SwarmReport,error){
	runID:=fmt.Sprintf("%s-%s-p%02d-c%02d",specimen.ID,strings.ToLower(condition),population,parallel)
	registry:=tlaloque.NewRegistry()
	for i:=0;i<population;i++{w:=PooledProcessWorker{ID:fmt.Sprintf("lfm2-slot-%02d",i),Binary:cfg.WorkerBinary,Timeout:cfg.Timeout};if err:=registry.Register(w);err!=nil{return Metric{RunID:runID},tlaloque.SwarmReport{},err}}
	consolidator:=ConsolidatorWorker(cfg.WorkerBinary,cfg.Timeout);if err:=registry.Register(consolidator);err!=nil{return Metric{RunID:runID},tlaloque.SwarmReport{},err}
	nodes:=[]tlaloque.SwarmNode{};deps:=[]string{};idx:=0
	for _,q:=range temporalbench.CanonicalQuestions(){for replica:=1;replica<=cfg.Replicas;replica++{id:=fmt.Sprintf("%s-r%02d",q.ID,replica);workerID:=fmt.Sprintf("lfm2-slot-%02d",idx%population);nodes=append(nodes,tlaloque.SwarmNode{ID:id,Capability:VisualCapability,WorkerID:workerID});deps=append(deps,id);idx++}}
	nodes=append(nodes,tlaloque.SwarmNode{ID:"consolidate",Capability:ConsolidateCapability,WorkerID:"lfm2-boundary-consolidator",DependsOn:deps,PreferDeterministic:true})
	plan:=tlaloque.SwarmPlan{ID:"lfm2-boundary-"+runID,MaxParallel:parallel,Nodes:nodes}
	task:=VisualTask{Endpoint:cfg.Endpoint,Model:cfg.Model,ImagePath:specimen.Path,Crops:crops,Condition:condition,R4Prompt:cfg.R4Prompt,MaxTokens:cfg.MaxTokens};body,_:=json.Marshal(task)
	runner:=tlaloque.SwarmRunner{Registry:registry,Blackboard:&tlaloque.BlackboardRuntime{Store:store,RunID:runID}}
	report,runErr:=runner.Run(ctx,plan,runID,body)
	metric:=measure(runID,specimen,condition,population,parallel,cfg.Replicas,report)
	return metric,report,runErr
}

func measure(runID string,specimen SpecimenInfo,condition string,population,parallel,replicas int,report tlaloque.SwarmReport)Metric{
	m:=Metric{RunID:runID,SpecimenID:specimen.ID,Condition:condition,Population:population,MaxParallel:parallel,Replicas:replicas,ExpectedResponses:len(temporalbench.CanonicalQuestions())*replicas,PeakParallel:report.PeakParallel,DurationMS:report.DurationMS}
	latencies:=[]int64{};var consolidated ConsolidatedOutput
	for _,n:=range report.Nodes{
		if n.Error!=""{m.ErrorCount++;low:=strings.ToLower(n.Error);if strings.Contains(low,"http")||strings.Contains(low,"target status")||strings.Contains(low,"connection")||strings.Contains(low,"eof"){m.HTTPErrorCount++}}
		if n.Capability==VisualCapability && n.Error==""{
			var out VisualOutput;if err:=json.Unmarshal(n.Output,&out);err==nil{m.ValidResponses++;m.PromptTokens+=out.PromptTokens;m.CompletionTokens+=out.CompletionTokens;latencies=append(latencies,n.DurationMS)}
		}
		if n.NodeID=="consolidate"&&n.Error==""{_ = json.Unmarshal(n.Output,&consolidated)}
	}
	if m.ExpectedResponses>0{m.ValidResponseRate=float64(m.ValidResponses)/float64(m.ExpectedResponses)}
	m.LatencyP50MS=percentile(latencies,0.50);m.LatencyP95MS=percentile(latencies,0.95)
	if report.DurationMS>0{m.ThroughputPerSecond=float64(m.ValidResponses)/(float64(report.DurationMS)/1000)}
	responses:=[]temporalbench.Response{};for _,q:=range temporalbench.CanonicalQuestions(){if text,ok:=consolidated.Responses[q.ID];ok{responses=append(responses,temporalbench.Response{QuestionID:q.ID,Text:text})}}
	trial:=temporalbench.Trial{ID:runID,ModelID:RequiredModel,Provider:"LM_STUDIO",Condition:condition,Specimen:temporalbench.Specimen{ID:specimen.ID,SHA256:specimen.SHA256,Variant:specimen.ID,PNGBytes:specimen.PNGBytes,Width:specimen.Width,Height:specimen.Height},Responses:responses}
	result:=temporalbench.EvaluateTrial(trial);m.Score=result.OverallScore;m.TTemporal=result.TemporalReasoning;m.XExactness=result.ExactHonesty;m.InventedExactClaims=result.InventedExactClaims
	for _,l:=range result.Layers{switch l.Layer{case"P_PERCEPTION":m.PPerception=l.Score;case"R_PROTOCOL":m.RProtocol=l.Score;case"S_SEMANTIC":m.SSemantic=l.Score;case"T_TEMPORAL":m.TTemporal=l.Score;case"X_EXACTNESS":m.XExactness=l.Score}}
	return m
}

func markUseful(metrics []Metric){
	type key struct{s,c string;p int};serial:=map[key]Metric{}
	for _,m:=range metrics{if m.MaxParallel==1{serial[key{m.SpecimenID,m.Condition,m.Population}]=m}}
	for i:=range metrics{m:=&metrics[i];base,ok:=serial[key{m.SpecimenID,m.Condition,m.Population}];if!ok||m.ErrorCount!=0||m.ValidResponseRate!=1||base.ErrorCount!=0||base.ValidResponseRate!=1{continue};if m.Score+1.0/9.0+1e-9<base.Score{continue};if base.LatencyP95MS>0&&m.LatencyP95MS>2*base.LatencyP95MS{continue};m.Useful=true}
}

func summarize(metrics []Metric,pops,pars []int)Summary{
	s:=Summary{Schema:CampaignSchema+".summary",Runs:len(metrics),BoundaryByPopulation:map[string]int{},Acceptance:"score >= 6/9; temporal > 0; invented exact claims = 0"}
	for _,p:=range pops{best:=0;for _,c:=range pars{seen,all:=false,true;for _,m:=range metrics{if m.Population==p&&m.MaxParallel==c{seen=true;if!m.Useful{all=false;break}}};if seen&&all&&c>best{best=c}};s.BoundaryByPopulation[strconv.Itoa(p)]=best;if best>s.MaxUsefulParallel{s.MaxUsefulParallel=best}}
	for _,m:=range metrics{if m.SpecimenID=="candidate"&&m.Condition=="BLACKBOARD_CROPPED"&&m.Useful&&m.Score>=6.0/9.0&&m.TTemporal>0&&m.InventedExactClaims==0{s.MilestoneSuccess=true;break}}
	return s
}

func inspectSpecimen(id,path string)(SpecimenInfo,error){b,err:=os.ReadFile(path);if err!=nil{return SpecimenInfo{},err};sum:=sha256.Sum256(b);cfg,err:=png.DecodeConfig(strings.NewReader(string(b)));if err!=nil{return SpecimenInfo{},fmt.Errorf("%s: %w",path,err)};return SpecimenInfo{ID:id,Path:path,SHA256:hex.EncodeToString(sum[:]),PNGBytes:len(b),Width:cfg.Width,Height:cfg.Height},nil}
func percentile(in []int64,q float64)int64{if len(in)==0{return 0};x:=append([]int64{},in...);sort.Slice(x,func(i,j int)bool{return x[i]<x[j]});idx:=int(float64(len(x)-1)*q+0.999999);if idx<0{idx=0};if idx>=len(x){idx=len(x)-1};return x[idx]}
func writeJSON(path string,v any)error{if err:=os.MkdirAll(filepath.Dir(path),0o700);err!=nil{return err};b,err:=json.MarshalIndent(v,"","  ");if err!=nil{return err};return os.WriteFile(path,append(b,'\n'),0o600)}
func writeHashes(cfg Config,specimens []SpecimenInfo)error{h:=map[string]string{};for _,s:=range specimens{h[s.ID+"_png_sha256"]=s.SHA256};sum:=sha256.Sum256([]byte(cfg.R4Prompt));h["master_prompt_r4_sha256"]=hex.EncodeToString(sum[:]);return writeJSON(filepath.Join(cfg.OutputDir,"hashes.json"),h)}
func writeMetricsCSV(path string,metrics []Metric)error{if err:=os.MkdirAll(filepath.Dir(path),0o700);err!=nil{return err};f,err:=os.Create(path);if err!=nil{return err};defer f.Close();w:=csv.NewWriter(f);defer w.Flush();_ = w.Write([]string{"run_id","specimen","condition","population","parallel","score","P","R","S","T","X","invented_exact","valid_rate","errors","http_errors","prompt_tokens","completion_tokens","p50_ms","p95_ms","throughput","peak_parallel","duration_ms","useful"});for _,m:=range metrics{_ = w.Write([]string{m.RunID,m.SpecimenID,m.Condition,strconv.Itoa(m.Population),strconv.Itoa(m.MaxParallel),fmt.Sprintf("%.6f",m.Score),fmt.Sprintf("%.6f",m.PPerception),fmt.Sprintf("%.6f",m.RProtocol),fmt.Sprintf("%.6f",m.SSemantic),fmt.Sprintf("%.6f",m.TTemporal),fmt.Sprintf("%.6f",m.XExactness),strconv.Itoa(m.InventedExactClaims),fmt.Sprintf("%.6f",m.ValidResponseRate),strconv.Itoa(m.ErrorCount),strconv.Itoa(m.HTTPErrorCount),strconv.Itoa(m.PromptTokens),strconv.Itoa(m.CompletionTokens),strconv.FormatInt(m.LatencyP50MS,10),strconv.FormatInt(m.LatencyP95MS,10),fmt.Sprintf("%.6f",m.ThroughputPerSecond),strconv.Itoa(m.PeakParallel),strconv.FormatInt(m.DurationMS,10),strconv.FormatBool(m.Useful)})};return w.Error()}
