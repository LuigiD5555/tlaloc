package closedloop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/adaptivesearch"
	"tlaloc.local/behaviorlab/internal/learningmemory"
	"tlaloc.local/behaviorlab/internal/target"
	"tlaloc.local/behaviorlab/internal/temporalbench"
	"tlaloc.local/behaviorlab/internal/visualsearch"
)

type prepared struct {
	cfg        Config
	master     string
	conditions []string
	store      learningmemory.Store
}

type specimenRun struct {
	report SpecimenReport
	events []learningmemory.Event
}

func Validate(cfg Config) error {
	_, err := prepare(cfg, false)
	return err
}

func Run(ctx context.Context, cfg Config) (Report, error) {
	p, err := prepare(cfg, true)
	if err != nil { return Report{}, err }
	if err := os.MkdirAll(p.cfg.OutputDir, 0o755); err != nil { return Report{}, err }
	report := Report{Schema:ReportSchema,RunID:p.cfg.RunID,OutputDir:p.cfg.OutputDir,MemoryRoot:p.store.Root,Authority:"CLOSED_LOOP_EXECUTES_EXPERIMENTS_ONLY_ORIGAMI_PROMOTION_REMAINS_EXTERNAL_AND_EVIDENCE_GATED"}
	tested := map[string]bool{}
	for gen:=1; gen<=p.cfg.MaxGenerations; gen++ {
		genDir:=filepath.Join(p.cfg.OutputDir,fmt.Sprintf("generation-%03d",gen)); if err:=os.MkdirAll(genDir,0o755);err!=nil{return report,err}
		baseRun, errs, err:=p.runSpecimen(ctx,gen,genDir,p.cfg.Baseline,"",nil); report.ExecutionErrors=append(report.ExecutionErrors,errs...); if err!=nil{return report,err}
		events,err:=p.store.LoadAll();if err!=nil{return report,err}
		planBefore:=adaptivesearch.BuildPlan(p.store.Root,events);planBeforePath:=filepath.Join(genDir,"plan-before.json");if err:=writeJSON(planBeforePath,planBefore);err!=nil{return report,err}
		visualCandidates,cfgByID:=p.availableCandidates(events,tested)
		queue:=adaptivesearch.Prioritize(planBefore,visualCandidates,p.cfg.CandidatesPerGeneration)
		queuePath:=filepath.Join(genDir,"candidate-queue.json");if err:=writeJSON(queuePath,queue);err!=nil{return report,err}
		changeIDs:=map[string][]string{}
		changeEvents:=adaptivesearch.ChangeAttemptEvents(queue,visualCandidates)
		if len(changeEvents)==0 && len(queue.CandidateOrder)>0 { changeEvents=p.explorationChangeEvents(queue,baseRun.events,cfgByID) }
		if len(changeEvents)>0 { _,_,stored,putErr:=p.store.PutAll(changeEvents);if putErr!=nil{return report,putErr};for _,e:=range stored{changeIDs[e.CandidateID]=append(changeIDs[e.CandidateID],e.EventID)} }
		g:=GenerationReport{Generation:gen,PlanBeforePath:planBeforePath,QueuePath:queuePath,Baseline:baseRun.report}
		baselineMetric:=metricValue(baseRun.report.Scores,p.cfg.OutcomeMetric)
		for _,item:=range queue.CandidateOrder {
			cc,ok:=cfgByID[item.CandidateID];if !ok{continue};tested[cc.ID]=true;g.SelectedIDs=append(g.SelectedIDs,cc.ID)
			if err:=p.ensureCandidatePNG(ctx,cc);err!=nil{report.ExecutionErrors=append(report.ExecutionErrors,ExecutionError{Generation:gen,SpecimenID:cc.ID,CandidateID:cc.ID,Error:"candidate build: "+err.Error()});continue}
			run,runErrs,runErr:=p.runSpecimen(ctx,gen,genDir,SpecimenConfig{ID:cc.ID,PNG:cc.PNG},cc.ID,cc.Mutations);report.ExecutionErrors=append(report.ExecutionErrors,runErrs...);if runErr!=nil{return report,runErr};g.Candidates=append(g.Candidates,run.report)
			after:=metricValue(run.report.Scores,p.cfg.OutcomeMetric);out:=CandidateOutcome{CandidateID:cc.ID,Metric:p.cfg.OutcomeMetric,Before:baselineMetric,After:after,Delta:after-baselineMetric}
			parents:=append([]string(nil),changeIDs[cc.ID]...);parents=append(parents,observationIDs(run.events)...);parents=dedupeLimit(parents,30)
			if len(parents)>=2 && len(changeIDs[cc.ID])>0 {
				beforeCopy,afterCopy,deltaCopy:=baselineMetric,after,after-baselineMetric
				ev:=learningmemory.Event{Schema:learningmemory.EventSchema,EventType:learningmemory.EventOutcome,EvidenceClass:learningmemory.EvidenceManual,CandidateID:cc.ID,ParentEventIDs:parents,BeforeScore:&beforeCopy,AfterScore:&afterCopy,Delta:&deltaCopy,Tags:[]string{"closed-loop","metric:"+p.cfg.OutcomeMetric,fmt.Sprintf("generation:%d",gen)}}
				_,stored,putErr:=p.store.Put(ev);if putErr!=nil{return report,putErr};out.EventID=stored.EventID
			}
			g.Outcomes=append(g.Outcomes,out)
		}
		events,err=p.store.LoadAll();if err!=nil{return report,err};planAfter:=adaptivesearch.BuildPlan(p.store.Root,events);planAfterPath:=filepath.Join(genDir,"plan-after.json");if err:=writeJSON(planAfterPath,planAfter);err!=nil{return report,err};g.PlanAfterPath=planAfterPath;report.FinalPlanPath=planAfterPath
		remaining, _:=p.availableCandidates(events,tested);g.RemainingBank=len(remaining);report.Generations=append(report.Generations,g)
		if len(remaining)==0 { report.StopReason="CANDIDATE_BANK_EXHAUSTED";break }
		if gen==p.cfg.MaxGenerations { report.StopReason="MAX_GENERATIONS_REACHED" }
	}
	if report.StopReason==""{report.StopReason="COMPLETED"}
	if err:=writeJSON(filepath.Join(p.cfg.OutputDir,"closed-loop-report.json"),report);err!=nil{return report,err}
	return report,nil
}

func prepare(cfg Config, checkFiles bool)(prepared,error){
	if cfg.Schema!=""&&cfg.Schema!=ConfigSchema{return prepared{},fmt.Errorf("unexpected schema %q",cfg.Schema)};cfg.Schema=ConfigSchema
	if strings.TrimSpace(cfg.RunID)==""{return prepared{},fmt.Errorf("run_id is required")};if strings.TrimSpace(cfg.OutputDir)==""{return prepared{},fmt.Errorf("output_dir is required")};if cfg.BenchmarkID==""{cfg.BenchmarkID="origami-temporal-native-r0"}
	if cfg.TrialsPerModel<=0{cfg.TrialsPerModel=1};if cfg.CandidatesPerGeneration<=0{cfg.CandidatesPerGeneration=2};if cfg.MaxGenerations<=0{cfg.MaxGenerations=1};if cfg.OutcomeMetric==""{cfg.OutcomeMetric=OutcomeNative};if cfg.OutcomeMetric!=OutcomeNative&&cfg.OutcomeMetric!=OutcomeOverall{return prepared{},fmt.Errorf("unsupported outcome_metric %q",cfg.OutcomeMetric)}
	if len(cfg.Conditions)==0{cfg.Conditions=[]string{"NATIVE_PNG_ONLY"};if cfg.MasterPrompt!=""{cfg.Conditions=append(cfg.Conditions,"R4_ASSISTED")}}
	conditions:=[]string{};seenCondition:=map[string]bool{};for _,c:=range cfg.Conditions{c=strings.ToUpper(strings.TrimSpace(c));if c!="NATIVE_PNG_ONLY"&&c!="R4_ASSISTED"{return prepared{},fmt.Errorf("unsupported condition %q",c)};if !seenCondition[c]{conditions=append(conditions,c);seenCondition[c]=true}}
	if cfg.OutcomeMetric==OutcomeNative&&!seenCondition["NATIVE_PNG_ONLY"]{return prepared{},fmt.Errorf("NATIVE_SCORE requires NATIVE_PNG_ONLY condition")}
	if strings.TrimSpace(cfg.Baseline.ID)==""||strings.TrimSpace(cfg.Baseline.PNG)==""{return prepared{},fmt.Errorf("baseline id and png are required")};if len(cfg.Models)==0{return prepared{},fmt.Errorf("at least one model is required")}
	modelNames:=map[string]bool{};for i:=range cfg.Models{m:=&cfg.Models[i];if m.Name==""{m.Name=m.Model};if m.Provider==""{m.Provider="OPENAI_COMPAT"};if strings.ToUpper(m.Provider)!="OPENAI_COMPAT"{return prepared{},fmt.Errorf("model %s provider %q unsupported in R0",m.Name,m.Provider)};if m.Model==""{return prepared{},fmt.Errorf("model %d model is required",i)};if modelNames[m.Name]{return prepared{},fmt.Errorf("duplicate model name %q",m.Name)};modelNames[m.Name]=true;if m.TimeoutSeconds<=0{m.TimeoutSeconds=120};if m.TransportRetries<0{m.TransportRetries=0}}
	candidateIDs:=map[string]bool{};for i:=range cfg.Candidates{c:=&cfg.Candidates[i];if c.ID==""||c.PNG==""{return prepared{},fmt.Errorf("candidate %d requires id and png",i)};if candidateIDs[c.ID]{return prepared{},fmt.Errorf("duplicate candidate id %q",c.ID)};candidateIDs[c.ID]=true;if c.BaseProfileID==""{return prepared{},fmt.Errorf("candidate %q base_profile_id required",c.ID)};if len(c.Mutations)==0{return prepared{},fmt.Errorf("candidate %q requires mutations",c.ID)};for j,m:=range c.Mutations{if !m.Experimental{return prepared{},fmt.Errorf("candidate %q mutation %d must remain experimental",c.ID,j)}}}
	master:="";if seenCondition["R4_ASSISTED"]{if cfg.MasterPrompt==""{return prepared{},fmt.Errorf("R4_ASSISTED requires master_prompt")};if checkFiles{b,err:=os.ReadFile(cfg.MasterPrompt);if err!=nil{return prepared{},err};master=string(b)}}
	if checkFiles{if _,err:=readPNGMeta(cfg.Baseline.PNG);err!=nil{return prepared{},fmt.Errorf("baseline: %w",err)};for _,c:=range cfg.Candidates{if _,err:=os.Stat(c.PNG);err==nil{if _,err:=readPNGMeta(c.PNG);err!=nil{return prepared{},fmt.Errorf("candidate %s: %w",c.ID,err)}}else if len(c.BuildCommand)==0{return prepared{},fmt.Errorf("candidate %s png missing and no build_command: %w",c.ID,err)}}}
	return prepared{cfg:cfg,master:master,conditions:conditions,store:learningmemory.New(cfg.MemoryRoot)},nil
}

func (p prepared) runSpecimen(ctx context.Context,generation int,genDir string,s SpecimenConfig,candidateID string,mutations []visualsearch.Mutation)(specimenRun,[]ExecutionError,error){
	meta,err:=readPNGMeta(s.PNG);if err!=nil{return specimenRun{},nil,err};imageBytes:=meta.bytes
	campaign:=temporalbench.Campaign{Schema:temporalbench.CampaignSchema,BenchmarkID:p.cfg.BenchmarkID};errs:=[]ExecutionError{};questions:=temporalbench.CanonicalQuestions()
	models:=map[string]ModelConfig{};for _,m:=range p.cfg.Models{models[m.Name]=m}
	for _,m:=range p.cfg.Models{for trialN:=1;trialN<=p.cfg.TrialsPerModel;trialN++{for _,condition:=range p.conditions{
		trial:=temporalbench.Trial{ID:fmt.Sprintf("g%03d-%s-%s-%s-%02d",generation,slug(s.ID),slug(m.Name),strings.ToLower(condition),trialN),ModelID:m.Name,Provider:m.Provider,Condition:condition,Specimen:temporalbench.Specimen{ID:s.ID,SHA256:meta.sha,Variant:"original",PNGBytes:len(imageBytes),Width:meta.width,Height:meta.height}}
		system:=p.systemFor(condition);complete:=true
		for _,q:=range questions{r,callErr:=p.call(ctx,m,system,q.Text,imageBytes);if callErr!=nil{errs=append(errs,ExecutionError{Generation:generation,SpecimenID:s.ID,CandidateID:candidateID,ModelID:m.Name,Condition:condition,QuestionID:q.ID,Error:callErr.Error()});complete=false;break};r.QuestionID=q.ID;trial.Responses=append(trial.Responses,r)}
		if complete{campaign.Trials=append(campaign.Trials,trial)}
	}}}
	clean:=temporalbench.EvaluateCampaign(campaign)
	if p.cfg.DiagnosticRetries{
		byTrial:=map[string]temporalbench.Trial{};for _,t:=range campaign.Trials{byTrial[t.ID]=t}
		for _,tr:=range clean.Trials{failed:=[]string{};for _,q:=range tr.Questions{if !q.Pass{failed=append(failed,q.QuestionID)}};if len(failed)==0{continue};source:=byTrial[tr.TrialID];m,ok:=models[source.ModelID];if !ok{continue};diag:=temporalbench.Trial{ID:source.ID+"-diag",ModelID:source.ModelID,Provider:source.Provider,Condition:source.Condition,DiagnosticMode:true,DiagnosticQuestionIDs:failed,Specimen:source.Specimen};system:=strings.TrimSpace(p.systemFor(source.Condition)+"\n\n"+temporalbench.DiagnosticInstruction())
			qByID:=map[string]string{};for _,q:=range questions{qByID[q.ID]=q.Text};for _,qid:=range failed{r,callErr:=p.call(ctx,m,system,qByID[qid],imageBytes);if callErr!=nil{errs=append(errs,ExecutionError{Generation:generation,SpecimenID:s.ID,CandidateID:candidateID,ModelID:m.Name,Condition:source.Condition,QuestionID:qid,Diagnostic:true,Error:callErr.Error()});continue};r.QuestionID=qid;diag.Responses=append(diag.Responses,r)};if len(diag.Responses)>0{campaign.Trials=append(campaign.Trials,diag)}
		}
	}
	result:=temporalbench.EvaluateCampaign(campaign);specDir:=filepath.Join(genDir,slug(s.ID));if err:=os.MkdirAll(specDir,0o755);err!=nil{return specimenRun{},errs,err};campaignPath:=filepath.Join(specDir,"campaign.json");resultPath:=filepath.Join(specDir,"result.json");campaignBody,_:=json.MarshalIndent(campaign,"","  ");campaignBody=append(campaignBody,'\n');resultBody,_:=json.MarshalIndent(result,"","  ");resultBody=append(resultBody,'\n');if err:=os.WriteFile(campaignPath,campaignBody,0o644);err!=nil{return specimenRun{},errs,err};if err:=os.WriteFile(resultPath,resultBody,0o644);err!=nil{return specimenRun{},errs,err}
	imported,err:=learningmemory.ImportTemporalBenchmark(campaignBody,resultBody,campaign,result,learningmemory.ImportOptions{OrigamiVersion:p.cfg.OrigamiVersion,TlalocVersion:p.cfg.TlalocVersion,CandidateID:candidateID});if err!=nil{return specimenRun{},errs,err};_,_,stored,err:=p.store.PutAll(imported);if err!=nil{return specimenRun{},errs,err};ids:=[]string{};for _,e:=range stored{ids=append(ids,e.EventID)}
	rep:=SpecimenReport{SpecimenID:s.ID,CandidateID:candidateID,PNG:s.PNG,SHA256:meta.sha,Scores:summarizeScores(result),CampaignPath:campaignPath,ResultPath:resultPath,MemoryEvents:len(stored),MemoryEventIDs:ids,ExecutionErrors:countSpecimenErrors(errs,s.ID,candidateID)}
	return specimenRun{report:rep,events:stored},errs,nil
}

func (p prepared) call(parent context.Context,m ModelConfig,system,question string,image []byte)(temporalbench.Response,error){
	client:=target.OpenAICompat{BaseURL:m.BaseURL,Model:m.Model,Temperature:m.Temperature};if m.APIKeyEnv!=""{client.APIKey=os.Getenv(m.APIKeyEnv)};attempts:=m.TransportRetries+1;var last error
	for i:=0;i<attempts;i++{ctx,cancel:=context.WithTimeout(parent,time.Duration(m.TimeoutSeconds)*time.Second);start:=time.Now();r,err:=client.CompletePerception(ctx,target.PerceptionInput{SystemPrompt:system,Question:question,Image:image,MediaType:"image/png"});cancel();if err==nil{return temporalbench.Response{Text:r.Content,LatencyMS:time.Since(start).Milliseconds()},nil};last=err}
	return temporalbench.Response{},last
}

func (p prepared) systemFor(condition string)string{if condition=="R4_ASSISTED"{return p.master};return ""}

func (p prepared) availableCandidates(events []learningmemory.Event,tested map[string]bool)([]visualsearch.Candidate,map[string]CandidateConfig){
	done:=map[string]bool{};for _,e:=range events{if e.EventType==learningmemory.EventOutcome&&e.CandidateID!=""{done[e.CandidateID]=true}}
	out:=[]visualsearch.Candidate{};by:=map[string]CandidateConfig{};for _,c:=range p.cfg.Candidates{if tested[c.ID]||done[c.ID]{continue};vc:=c.VisualCandidate();out=append(out,vc);by[c.ID]=c};return out,by
}

func (p prepared) explorationChangeEvents(queue adaptivesearch.Queue,baseline []learningmemory.Event,by map[string]CandidateConfig)[]learningmemory.Event{
	parents:=observationIDs(baseline);parents=dedupeLimit(parents,20);if len(parents)==0{return nil};out:=[]learningmemory.Event{};for _,item:=range queue.CandidateOrder{c,ok:=by[item.CandidateID];if !ok{continue};tags:=[]string{"closed-loop","exploration"};for _,m:=range c.Mutations{tags=append(tags,"mutation:"+string(m.Kind))};sort.Strings(tags);out=append(out,learningmemory.Event{Schema:learningmemory.EventSchema,EventType:learningmemory.EventChange,EvidenceClass:learningmemory.EvidenceManual,CandidateID:c.ID,ParentEventIDs:parents,ChangeSummary:"Closed-loop exploration candidate selected before evidence scoring",Tags:tags})};return out
}

func (p prepared) ensureCandidatePNG(ctx context.Context,c CandidateConfig)error{if _,err:=os.Stat(c.PNG);err==nil{return nil};if len(c.BuildCommand)==0{return fmt.Errorf("png %s missing",c.PNG)};if err:=os.MkdirAll(filepath.Dir(c.PNG),0o755);err!=nil{return err};mutationJSON,_:=json.Marshal(c.Mutations);cmd:=exec.CommandContext(ctx,c.BuildCommand[0],c.BuildCommand[1:]...);cmd.Env=append(os.Environ(),"TLALOC_CANDIDATE_ID="+c.ID,"TLALOC_OUTPUT_PNG="+c.PNG,"TLALOC_MUTATIONS_JSON="+string(mutationJSON));output,err:=cmd.CombinedOutput();if err!=nil{return fmt.Errorf("%v: %s",err,strings.TrimSpace(string(output)))};_,err=readPNGMeta(c.PNG);return err}

type pngMeta struct{bytes []byte;sha string;width,height int}
func readPNGMeta(path string)(pngMeta,error){b,err:=os.ReadFile(path);if err!=nil{return pngMeta{},err};cfg,err:=png.DecodeConfig(bytes.NewReader(b));if err!=nil{return pngMeta{},fmt.Errorf("invalid PNG %s: %w",path,err)};sum:=sha256.Sum256(b);return pngMeta{bytes:b,sha:hex.EncodeToString(sum[:]),width:cfg.Width,height:cfg.Height},nil}
func summarizeScores(r temporalbench.Result)ScoreSummary{out:=ScoreSummary{};nativeN,assistN:=0,0;for _,t:=range r.Trials{if t.DiagnosticMode{continue};out.CleanTrials++;out.MeanOverall+=t.OverallScore;if t.Condition=="NATIVE_PNG_ONLY"{out.MeanNative+=t.OverallScore;nativeN++};if t.Condition=="R4_ASSISTED"{out.MeanAssisted+=t.OverallScore;assistN++}};if out.CleanTrials>0{out.MeanOverall/=float64(out.CleanTrials)};if nativeN>0{out.MeanNative/=float64(nativeN)};if assistN>0{out.MeanAssisted/=float64(assistN)};return out}
func metricValue(s ScoreSummary,metric string)float64{if metric==OutcomeOverall{return s.MeanOverall};return s.MeanNative}
func observationIDs(events []learningmemory.Event)[]string{out:=[]string{};for _,e:=range events{if e.EventType==learningmemory.EventObservation&&e.EventID!=""{out=append(out,e.EventID)}};sort.Strings(out);return out}
func dedupeLimit(in []string,n int)[]string{seen:=map[string]bool{};out:=[]string{};for _,x:=range in{if x==""||seen[x]{continue};seen[x]=true;out=append(out,x);if n>0&&len(out)>=n{break}};return out}
func countSpecimenErrors(in []ExecutionError,s,c string)int{n:=0;for _,e:=range in{if e.SpecimenID==s&&e.CandidateID==c{n++}};return n}
func slug(s string)string{s=strings.ToLower(strings.TrimSpace(s));var b strings.Builder;for _,r:=range s{if (r>='a'&&r<='z')||(r>='0'&&r<='9')||r=='-'||r=='_'{b.WriteRune(r)}else{b.WriteByte('-')}};return strings.Trim(b.String(),"-")}
func writeJSON(path string,v any)error{b,err:=json.MarshalIndent(v,"","  ");if err!=nil{return err};b=append(b,'\n');return os.WriteFile(path,b,0o644)}
