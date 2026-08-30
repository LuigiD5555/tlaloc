package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tlaloc.local/behaviorlab/internal/target"
)

type OrigamiObservation struct {
	Schema       string `json:"schema"`
	Model        string `json:"model"`
	Trial        int    `json:"trial"`
	Transport    string `json:"transport"`
	EvidenceKind string `json:"evidence_kind"`
	BootText     []string `json:"boot_text"`
	ProbeTop     string `json:"probe_top"`
	ProbeBottom  string `json:"probe_bottom"`
	ToolProtocol string `json:"tool_protocol"`
	AddressABI   string `json:"address_abi"`
	T3           any    `json:"t3,omitempty"`
}

type TrialExecution struct {
	Observation OrigamiObservation `json:"observation"`
	RawModelOutput string `json:"raw_model_output"`
	PromptTokensReported int `json:"prompt_tokens_reported"`
	CompletionTokensReported int `json:"completion_tokens_reported"`
}

func ObservationQuestion(includeT3 bool) string {
	t3 := "Set t3 to null."
	if includeT3 { t3 = "If you can read T3 natively from the image, include carrier_id, store_root_sha256, source_sha256, page_count, block_count, document_count and object_count. Otherwise set t3 to null." }
	return `Inspect only the supplied Origami image. Return JSON only with keys: boot_text (array of the T0 lines you can read), probe_top (8 bits), probe_bottom (8 bits), tool_protocol, address_abi, t3. Do not guess unreadable values; use empty strings or null. ` + t3
}

func RunPerceptionTrial(ctx context.Context, client target.OpenAICompat, modelName string, trial int, variant TransportVariant, systemPrompt string, includeT3 bool) (TrialExecution,error) {
	result,err:=client.CompletePerception(ctx,target.PerceptionInput{SystemPrompt:systemPrompt,Question:ObservationQuestion(includeT3),Image:variant.Bytes,MediaType:variant.MediaType});if err!=nil{return TrialExecution{},err}
	obs,err:=ParseObservation(result.Content);if err!=nil{return TrialExecution{},fmt.Errorf("parse model observation: %w",err)}
	obs.Schema="origami.perception-observation.r1";obs.Model=modelName;obs.Trial=trial;obs.Transport=variant.Name;obs.EvidenceKind="REAL_MODEL"
	return TrialExecution{Observation:obs,RawModelOutput:result.Content,PromptTokensReported:result.PromptTokensReported,CompletionTokensReported:result.CompletionTokensReported},nil
}

func ParseObservation(content string)(OrigamiObservation,error){
	content=strings.TrimSpace(content);if strings.HasPrefix(content,"```"){lines:=strings.Split(content,"\n");if len(lines)>=3{lines=lines[1:];if strings.TrimSpace(lines[len(lines)-1])=="```"{lines=lines[:len(lines)-1]};content=strings.Join(lines,"\n")}}
	start:=strings.Index(content,"{");end:=strings.LastIndex(content,"}");if start<0||end<start{return OrigamiObservation{},fmt.Errorf("no JSON object in model output")};content=content[start:end+1]
	var raw struct{BootText []string `json:"boot_text"`;ProbeTop string `json:"probe_top"`;ProbeBottom string `json:"probe_bottom"`;ToolProtocol string `json:"tool_protocol"`;AddressABI string `json:"address_abi"`;T3 json.RawMessage `json:"t3"`}
	dec:=json.NewDecoder(strings.NewReader(content));dec.DisallowUnknownFields();if err:=dec.Decode(&raw);err!=nil{return OrigamiObservation{},err};var t3 any;if len(raw.T3)>0&&string(raw.T3)!="null"{if err:=json.Unmarshal(raw.T3,&t3);err!=nil{return OrigamiObservation{},err}}
	return OrigamiObservation{BootText:raw.BootText,ProbeTop:strings.TrimSpace(raw.ProbeTop),ProbeBottom:strings.TrimSpace(raw.ProbeBottom),ToolProtocol:strings.TrimSpace(raw.ToolProtocol),AddressABI:strings.TrimSpace(raw.AddressABI),T3:t3},nil
}

type OrigamiEvaluator struct { Binary string; Carrier string }
func (e OrigamiEvaluator) Evaluate(ctx context.Context, observation OrigamiObservation)(EvaluatorReport,error){
	if e.Binary==""||e.Carrier==""{return EvaluatorReport{},fmt.Errorf("Origami evaluator binary and carrier are required")};body,err:=json.Marshal(observation);if err!=nil{return EvaluatorReport{},err};tmp,err:=os.CreateTemp("","origami-observation-*.json");if err!=nil{return EvaluatorReport{},err};path:=tmp.Name();defer os.Remove(path);if _,err:=tmp.Write(body);err!=nil{tmp.Close();return EvaluatorReport{},err};if err:=tmp.Close();err!=nil{return EvaluatorReport{},err}
	cmd:=exec.CommandContext(ctx,e.Binary,"-carrier",e.Carrier,"-observation",path,"-out","-");var stdout,stderr bytes.Buffer;cmd.Stdout=&stdout;cmd.Stderr=&stderr;if err:=cmd.Run();err!=nil{return EvaluatorReport{},fmt.Errorf("origami evaluator: %w: %s",err,strings.TrimSpace(stderr.String()))};var report EvaluatorReport;if err:=json.Unmarshal(stdout.Bytes(),&report);err!=nil{return EvaluatorReport{},fmt.Errorf("decode evaluator report: %w",err)};return report,nil
}
