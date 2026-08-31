package realcampaign

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/closedloop"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/temporalbench"
)

const parentProfile = "origami.temporal-carrier.r0.profile-1"

func Normalize(spec Spec) (Spec, error) {
	if spec.Schema != "" && spec.Schema != SpecSchema { return Spec{}, fmt.Errorf("unexpected schema %q",spec.Schema) }
	spec.Schema = SpecSchema
	spec.Phase = strings.ToUpper(strings.TrimSpace(spec.Phase))
	if spec.Phase=="" { spec.Phase=PhaseSmoke }
	if spec.Phase!=PhaseSmoke && spec.Phase!=PhaseEvidence { return Spec{},fmt.Errorf("unsupported phase %q",spec.Phase) }
	if strings.TrimSpace(spec.CampaignID)=="" { return Spec{},fmt.Errorf("campaign_id is required") }
	if strings.TrimSpace(spec.Endpoint)=="" { spec.Endpoint="http://127.0.0.1:1234/v1" }
	spec.Endpoint=strings.TrimRight(spec.Endpoint,"/")
	for name,value:=range map[string]string{"program":spec.Program,"temporal_carrier":spec.TemporalCarrier,"candidate_builder":spec.CandidateBuilder,"output_dir":spec.OutputDir}{ if strings.TrimSpace(value)=="" { return Spec{},fmt.Errorf("%s is required",name) } }
	if spec.TimeoutSeconds<=0 { spec.TimeoutSeconds=180 }
	if spec.TransportRetries<0 { spec.TransportRetries=0 }
	if spec.Phase==PhaseSmoke {
		if spec.TrialsPerModel<=0 { spec.TrialsPerModel=1 }
		if spec.CandidatesPerGen<=0 { spec.CandidatesPerGen=1 }
		if spec.MaxGenerations<=0 { spec.MaxGenerations=1 }
		if len(spec.Conditions)==0 { spec.Conditions=[]string{"NATIVE_PNG_ONLY"} }
	} else {
		if spec.TrialsPerModel<=0 { spec.TrialsPerModel=3 }
		if spec.TrialsPerModel<3 { return Spec{},fmt.Errorf("EVIDENCE requires trials_per_model >= 3") }
		if spec.CandidatesPerGen<=0 { spec.CandidatesPerGen=2 }
		if spec.MaxGenerations<=0 { spec.MaxGenerations=3 }
		if len(spec.Conditions)==0 { spec.Conditions=[]string{"NATIVE_PNG_ONLY"}; if spec.MasterPrompt!="" { spec.Conditions=append(spec.Conditions,"R4_ASSISTED") } }
	}
	if err:=validateRealModelID(spec.Model); err!=nil && spec.Model!="" { return Spec{},err }
	if err:=validateCanonicalSignalChain(spec.Program);err!=nil{return Spec{},err}
	return spec,nil
}

func Doctor(ctx context.Context, raw Spec) (DoctorResult,error) {
	spec,err:=Normalize(raw);if err!=nil{return DoctorResult{},err}
	carrier,err:=exec.LookPath(spec.TemporalCarrier);if err!=nil{return DoctorResult{},fmt.Errorf("temporal carrier: %w",err)}
	builder,err:=exec.LookPath(spec.CandidateBuilder);if err!=nil{return DoctorResult{},fmt.Errorf("candidate builder: %w",err)}
	caps,err:=queryBuilderCapabilities(ctx,builder);if err!=nil{return DoctorResult{},err}
	if caps.Schema!="origami.experimental-candidate.r0.capabilities" { return DoctorResult{},fmt.Errorf("unexpected builder capability schema %q",caps.Schema) }
	if caps.ExactPlaneMutation { return DoctorResult{},fmt.Errorf("candidate builder declares exact-plane mutation") }
	if !containsFold(caps.ParentProfiles,parentProfile) { return DoctorResult{},fmt.Errorf("candidate builder does not support parent profile %s",parentProfile) }
	models,err:=discoverModels(ctx,spec.Endpoint,apiKey(spec.APIKeyEnv));if err!=nil{return DoctorResult{},err}
	selected,err:=selectModel(spec.Model,models);if err!=nil{return DoctorResult{},err}
	if err:=validateRealModelID(selected);err!=nil{return DoctorResult{},err}

	tmp,err:=os.MkdirTemp("","tlaloc-real-vlm-doctor-*");if err!=nil{return DoctorResult{},err};defer os.RemoveAll(tmp)
	baseline:=filepath.Join(tmp,"baseline.png")
	if err:=buildBaseline(ctx,carrier,spec.Program,baseline);err!=nil{return DoctorResult{},err}
	image,err:=os.ReadFile(baseline);if err!=nil{return DoctorResult{},err}
	if len(image)!=8192{return DoctorResult{},fmt.Errorf("expected frozen 8192-byte temporal carrier, got %d",len(image))}
	questions:=temporalbench.CanonicalQuestions();if len(questions)==0{return DoctorResult{},fmt.Errorf("temporal benchmark has no questions")}
	client:=target.OpenAICompat{BaseURL:spec.Endpoint,Model:selected,Temperature:spec.Temperature,APIKey:apiKey(spec.APIKeyEnv)}
	probeCtx,cancel:=context.WithTimeout(ctx,time.Duration(spec.TimeoutSeconds)*time.Second);defer cancel()
	probe,err:=client.CompletePerception(probeCtx,target.PerceptionInput{Question:questions[0].Text,Image:image,MediaType:"image/png"});if err!=nil{return DoctorResult{},fmt.Errorf("multimodal probe failed: %w",err)}
	carrierSHA,_:=fileSHA(carrier);builderSHA,_:=fileSHA(builder);programSHA,_:=fileSHA(spec.Program)
	return DoctorResult{Schema:SpecSchema+".doctor",Endpoint:spec.Endpoint,DiscoveredModels:models,SelectedModel:selected,VisionTransport:true,ProbeResponse:probe.Content,TemporalCarrier:carrier,TemporalCarrierSHA256:carrierSHA,CandidateBuilder:builder,CandidateBuilderSHA256:builderSHA,BuilderCapabilities:caps,ProgramSHA256:programSHA,ParentProfile:parentProfile,Ready:true},nil
}

func Prepare(ctx context.Context,raw Spec)(Prepared,error){
	spec,err:=Normalize(raw);if err!=nil{return Prepared{},err}
	doc,err:=Doctor(ctx,spec);if err!=nil{return Prepared{},err}
	spec.Model=doc.SelectedModel
	phaseDir:=filepath.Join(spec.OutputDir,strings.ToLower(spec.Phase));if err:=os.MkdirAll(phaseDir,0o755);err!=nil{return Prepared{},err}
	baseline:=filepath.Join(phaseDir,"baseline.png");if err:=buildBaseline(ctx,doc.TemporalCarrier,spec.Program,baseline);err!=nil{return Prepared{},err}
	baselineBody,err:=os.ReadFile(baseline);if err!=nil{return Prepared{},err};if len(baselineBody)!=8192{return Prepared{},fmt.Errorf("baseline bytes=%d, want 8192",len(baselineBody))}
	memoryRoot:=filepath.Join(phaseDir,"learning-memory")
	closedOut:=filepath.Join(phaseDir,"closed-loop")
	cfg:=closedloop.Config{Schema:closedloop.ConfigSchema,RunID:spec.CampaignID+"-"+strings.ToLower(spec.Phase),BenchmarkID:"origami-temporal-native-r0",OutputDir:closedOut,MemoryRoot:memoryRoot,MasterPrompt:spec.MasterPrompt,OrigamiVersion:"6.0.0-alpha.15",TlalocVersion:detectTlalocVersion(),TrialsPerModel:spec.TrialsPerModel,CandidatesPerGeneration:spec.CandidatesPerGen,MaxGenerations:spec.MaxGenerations,MinIncumbentImprovement:0.01,DiagnosticRetries:true,Conditions:append([]string(nil),spec.Conditions...),OutcomeMetric:closedloop.OutcomeNative,Models:[]closedloop.ModelConfig{{Name:doc.SelectedModel,Provider:"OPENAI_COMPAT",BaseURL:spec.Endpoint,Model:doc.SelectedModel,APIKeyEnv:spec.APIKeyEnv,Temperature:spec.Temperature,TimeoutSeconds:spec.TimeoutSeconds,TransportRetries:spec.TransportRetries}},Baseline:closedloop.SpecimenConfig{ID:"signal-chain-r0",PNG:baseline},AutoCandidates:true,CandidateBuilder:[]string{doc.CandidateBuilder},AutoCandidateBaseProfileID:parentProfile,AutoCandidatesPerGeneration:4}
	configPath:=filepath.Join(phaseDir,"closed-loop.json");if err:=writeJSON(configPath,cfg);err!=nil{return Prepared{},err};configSHA,_:=fileSHA(configPath);baselineSHA,_:=fileSHA(baseline)
	status:="SMOKE_TRANSPORT_AND_BEHAVIOR_CHECK";policy:="REAL_MODEL_SINGLE_TRIAL_ISOLATED_MEMORY_NOT_PROMOTION_ELIGIBLE"
	if spec.Phase==PhaseEvidence{status="REAL_MODEL_REPEATED_EVIDENCE_READY";policy="REAL_MODEL_SINGLE_MODEL_REPEATED_TRIALS_REQUIRES_CROSS_MODEL_CONFIRMATION_FOR_PROMOTION"}
	manifest:=Manifest{Schema:ManifestSchema,CampaignID:spec.CampaignID,Phase:spec.Phase,Status:status,Endpoint:spec.Endpoint,ModelID:doc.SelectedModel,TlalocVersion:detectTlalocVersion(),OrigamiExpectedVersion:"6.0.0-alpha.15",ProgramPath:spec.Program,ProgramSHA256:doc.ProgramSHA256,BaselinePNG:baseline,BaselineSHA256:baselineSHA,BaselineBytes:len(baselineBody),TemporalCarrier:doc.TemporalCarrier,TemporalCarrierSHA256:doc.TemporalCarrierSHA256,CandidateBuilder:doc.CandidateBuilder,CandidateBuilderSHA256:doc.CandidateBuilderSHA256,BuilderCapabilities:doc.BuilderCapabilities,ClosedLoopConfig:configPath,ClosedLoopConfigSHA256:configSHA,MemoryRoot:memoryRoot,EvidencePolicy:policy,PromotionEligible:false,CrossModelEvidence:false}
	manifestPath:=filepath.Join(phaseDir,"manifest.json");if err:=writeJSON(manifestPath,manifest);err!=nil{return Prepared{},err}
	return Prepared{Spec:spec,Doctor:doc,Manifest:manifest,ManifestPath:manifestPath,ConfigPath:configPath},nil
}

func Run(ctx context.Context,raw Spec)(Prepared,closedloop.Report,error){
	prepared,err:=Prepare(ctx,raw);if err!=nil{return Prepared{},closedloop.Report{},err}
	var cfg closedloop.Config;if err:=readJSON(prepared.ConfigPath,&cfg);err!=nil{return prepared,closedloop.Report{},err}
	if err:=closedloop.ValidateAutoReady(ctx,cfg);err!=nil{return prepared,closedloop.Report{},err}
	report,err:=closedloop.RunAuto(ctx,cfg);if err!=nil{return prepared,report,err}
	return prepared,report,nil
}

func discoverModels(ctx context.Context,base,key string)([]string,error){
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,strings.TrimRight(base,"/")+"/models",nil);if err!=nil{return nil,err};if key!=""{req.Header.Set("Authorization","Bearer "+key)}
	resp,err:=http.DefaultClient.Do(req);if err!=nil{return nil,fmt.Errorf("model discovery: %w",err)};defer resp.Body.Close();body,err:=io.ReadAll(io.LimitReader(resp.Body,4<<20));if err!=nil{return nil,err};if resp.StatusCode/100!=2{return nil,fmt.Errorf("model discovery status %s: %s",resp.Status,strings.TrimSpace(string(body)))}
	var payload struct{Data []struct{ID string `json:"id"`} `json:"data"`};if err:=json.Unmarshal(body,&payload);err!=nil{return nil,fmt.Errorf("model discovery JSON: %w",err)}
	models:=[]string{};for _,m:=range payload.Data{if strings.TrimSpace(m.ID)!=""{models=append(models,m.ID)}};sort.Strings(models);if len(models)==0{return nil,fmt.Errorf("endpoint returned no models")};return models,nil
}

func selectModel(requested string,models []string)(string,error){
	if requested!=""{for _,m:=range models{if m==requested{return m,nil}};return "",fmt.Errorf("requested model %q not reported by endpoint; available: %s",requested,strings.Join(models,", "))}
	if len(models)==1{return models[0],nil};return "",fmt.Errorf("multiple models are available; select one explicitly: %s",strings.Join(models,", "))
}

func validateRealModelID(id string)error{u:=strings.ToUpper(strings.TrimSpace(id));if u==""{return nil};if strings.HasPrefix(u,"SYNTHETIC")||strings.Contains(u,"REPLACE_WITH")||strings.Contains(u,"PLACEHOLDER"){return fmt.Errorf("model id %q is not admissible for Real VLM Campaign R0",id)};return nil}

func queryBuilderCapabilities(ctx context.Context,builder string)(BuilderCapabilities,error){
	cmd:=exec.CommandContext(ctx,builder,"capabilities");body,err:=cmd.CombinedOutput();if err!=nil{return BuilderCapabilities{},fmt.Errorf("candidate builder capabilities: %v: %s",err,strings.TrimSpace(string(body)))};var caps BuilderCapabilities;dec:=json.NewDecoder(bytes.NewReader(body));dec.DisallowUnknownFields();if err:=dec.Decode(&caps);err!=nil{return BuilderCapabilities{},err};return caps,nil
}

func buildBaseline(ctx context.Context,carrier,program,out string)error{cmd:=exec.CommandContext(ctx,carrier,"-mode","build","-in",program,"-out",out);body,err:=cmd.CombinedOutput();if err!=nil{return fmt.Errorf("build baseline: %v: %s",err,strings.TrimSpace(string(body)))};return nil}

func validateCanonicalSignalChain(path string)error{
	body,err:=os.ReadFile(path);if err!=nil{return err}
	var p struct{Schema string `json:"schema"`;ID string `json:"id"`;Automaton struct{Schema string `json:"schema"`;ID string `json:"id"`;Cells []struct{ID,Kind,InitialState string;Neighbors []string `json:"neighbors"`} `json:"cells"`;Rules []struct{ID,TargetCell,FromState,ToState string;Requires []struct{CellID,State string} `json:"requires"`} `json:"rules"`} `json:"automaton"`;MaxSteps int `json:"max_steps"`;CheckpointEvery int `json:"checkpoint_every"`}
	if err:=json.Unmarshal(body,&p);err!=nil{return fmt.Errorf("program JSON: %w",err)}
	if p.Schema!="origami.temporal-program.r0"||p.ID!="signal-chain-r0"||p.Automaton.ID!="signal-chain"||p.MaxSteps!=8||p.CheckpointEvery!=2{return fmt.Errorf("program is not the canonical signal-chain benchmark fixture")}
	cells:=map[string]string{};for _,c:=range p.Automaton.Cells{cells[c.ID]=c.InitialState};if cells["A"]!="ACTIVE"||cells["B"]!="IDLE"||cells["C"]!="IDLE"||len(cells)!=3{return fmt.Errorf("program signal-chain cells do not match benchmark ground truth")}
	type edge struct{from,to string};required:=map[edge]bool{{"A","B"}:false,{"B","A"}:false,{"B","C"}:false,{"C","B"}:false};for _,r:=range p.Automaton.Rules{if len(r.Requires)>0{e:=edge{r.Requires[0].CellID,r.TargetCell};if _,ok:=required[e];ok{required[e]=true}}};if !required[edge{"A","B"}]||!required[edge{"B","A"}]||!required[edge{"B","C"}]||!required[edge{"C","B"}]{return fmt.Errorf("program signal-chain transition structure does not match benchmark ground truth")};return nil
}

func detectTlalocVersion()string{if v:=strings.TrimSpace(os.Getenv("TLALOC_VERSION"));v!=""{return v};candidates:=[]string{"VERSION","../VERSION"};if exe,err:=os.Executable();err==nil{dir:=filepath.Dir(exe);candidates=append([]string{filepath.Join(dir,"..","VERSION")},candidates...)};for _,p:=range candidates{if b,err:=os.ReadFile(p);err==nil&&strings.TrimSpace(string(b))!=""{return strings.TrimSpace(string(b))}};return "UNKNOWN"}
func apiKey(env string)string{if strings.TrimSpace(env)==""{return ""};return os.Getenv(env)}
func containsFold(in []string,want string)bool{for _,x:=range in{if strings.EqualFold(strings.TrimSpace(x),want){return true}};return false}
func fileSHA(path string)(string,error){f,err:=os.Open(path);if err!=nil{return "",err};defer f.Close();h:=sha256.New();if _,err:=io.Copy(h,f);err!=nil{return "",err};return hex.EncodeToString(h.Sum(nil)),nil}
func writeJSON(path string,v any)error{body,err:=json.MarshalIndent(v,"","  ");if err!=nil{return err};body=append(body,'\n');return os.WriteFile(path,body,0o644)}
func readJSON(path string,v any)error{body,err:=os.ReadFile(path);if err!=nil{return err};dec:=json.NewDecoder(bytes.NewReader(body));dec.DisallowUnknownFields();return dec.Decode(v)}
