package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

const (
	CapabilitySchemaR0  = "tlaloc.tlaloque-capability.r0"
	ScopeGeneral        = "GENERAL"
	ScopeSpecific       = "SPECIFIC"
	EngineProcess       = "PROCESS"
	EngineDeterministic = "DETERMINISTIC"
	EngineModel         = "MODEL"
)

type CapabilityDescriptor struct {
	Schema string `json:"schema"`; ID string `json:"id"`; Capability string `json:"capability"`; Scope string `json:"scope"`; Domain string `json:"domain,omitempty"`; Engine string `json:"engine"`; InputSchema string `json:"input_schema"`; OutputSchema string `json:"output_schema"`; Deterministic bool `json:"deterministic"`; ParameterCount int64 `json:"parameter_count,omitempty"`; MaxConcurrency int `json:"max_concurrency,omitempty"`; Tags []string `json:"tags,omitempty"`; Dependencies []string `json:"dependencies,omitempty"`
}
func (d CapabilityDescriptor) Normalize()(CapabilityDescriptor,error){if d.Schema==""{d.Schema=CapabilitySchemaR0};if d.Schema!=CapabilitySchemaR0{return CapabilityDescriptor{},fmt.Errorf("unexpected capability schema %q",d.Schema)};d.ID=strings.TrimSpace(d.ID);d.Capability=strings.ToUpper(strings.TrimSpace(d.Capability));d.Scope=strings.ToUpper(strings.TrimSpace(d.Scope));d.Domain=strings.ToUpper(strings.TrimSpace(d.Domain));d.Engine=strings.ToUpper(strings.TrimSpace(d.Engine));if d.ID==""||d.Capability==""||d.InputSchema==""||d.OutputSchema==""{return CapabilityDescriptor{},fmt.Errorf("id, capability, input_schema and output_schema are required")};if d.Scope==""{d.Scope=ScopeGeneral};if d.Scope!=ScopeGeneral&&d.Scope!=ScopeSpecific{return CapabilityDescriptor{},fmt.Errorf("unsupported scope %q",d.Scope)};if d.Scope==ScopeSpecific&&d.Domain==""{return CapabilityDescriptor{},fmt.Errorf("specific worker %q requires domain",d.ID)};if d.Engine==""{d.Engine=EngineModel};if d.MaxConcurrency<=0{d.MaxConcurrency=1};return d,nil}

type CapabilityRequest struct { TaskID string `json:"task_id"`; NodeID string `json:"node_id"`; Input json.RawMessage `json:"input"`; Context map[string]json.RawMessage `json:"context,omitempty"`; Blackboard *blackboard.Snapshot `json:"blackboard,omitempty"` }
type CapabilityResponse struct { WorkerID string `json:"worker_id"`; Output json.RawMessage `json:"output"`; Confidence float64 `json:"confidence,omitempty"`; Notes string `json:"notes,omitempty"`; Observations []blackboard.Observation `json:"observations,omitempty"` }
type CapabilityWorker interface { Descriptor() CapabilityDescriptor; Execute(context.Context,CapabilityRequest)(CapabilityResponse,error) }
type SelectionRequest struct { Capability string; WorkerID string; ScopeHint string; DomainHint string; PreferDeterministic bool; MaxParameters int64 }
type Registry struct { mu sync.RWMutex; workers map[string]CapabilityWorker }
func NewRegistry()*Registry{return &Registry{workers:map[string]CapabilityWorker{}}}
func(r *Registry)Register(worker CapabilityWorker)error{if worker==nil{return fmt.Errorf("worker is nil")};d,err:=worker.Descriptor().Normalize();if err!=nil{return err};r.mu.Lock();defer r.mu.Unlock();if _,exists:=r.workers[d.ID];exists{return fmt.Errorf("worker %q already registered",d.ID)};r.workers[d.ID]=worker;return nil}
func(r *Registry)Get(id string)(CapabilityWorker,bool){r.mu.RLock();defer r.mu.RUnlock();w,ok:=r.workers[id];return w,ok}
func(r *Registry)Descriptors()[]CapabilityDescriptor{r.mu.RLock();defer r.mu.RUnlock();out:=make([]CapabilityDescriptor,0,len(r.workers));for _,w:=range r.workers{d,err:=w.Descriptor().Normalize();if err==nil{out=append(out,d)}};sort.Slice(out,func(i,j int)bool{return out[i].ID<out[j].ID});return out}
func(r *Registry)Select(req SelectionRequest)(CapabilityWorker,error){capability:=strings.ToUpper(strings.TrimSpace(req.Capability));if req.WorkerID!=""{w,ok:=r.Get(req.WorkerID);if !ok{return nil,fmt.Errorf("pinned worker %q not registered",req.WorkerID)};d,err:=w.Descriptor().Normalize();if err!=nil{return nil,err};if capability!=""&&d.Capability!=capability{return nil,fmt.Errorf("worker %q capability=%s, want %s",d.ID,d.Capability,capability)};return w,nil};scopeHint:=strings.ToUpper(strings.TrimSpace(req.ScopeHint));domainHint:=strings.ToUpper(strings.TrimSpace(req.DomainHint));if scopeHint!=""&&scopeHint!=ScopeGeneral&&scopeHint!=ScopeSpecific{return nil,fmt.Errorf("unsupported scope hint %q",scopeHint)};if scopeHint==ScopeSpecific&&domainHint==""{return nil,fmt.Errorf("SPECIFIC scope requires domain hint")};type candidate struct{worker CapabilityWorker;desc CapabilityDescriptor;score int};var candidates []candidate;r.mu.RLock();defer r.mu.RUnlock();for _,w:=range r.workers{d,err:=w.Descriptor().Normalize();if err!=nil||d.Capability!=capability{continue};if req.MaxParameters>0&&d.ParameterCount>req.MaxParameters{continue};if domainHint==""&&d.Scope==ScopeSpecific{continue};if scopeHint==ScopeGeneral&&d.Scope!=ScopeGeneral{continue};if scopeHint==ScopeSpecific&&(d.Scope!=ScopeSpecific||d.Domain!=domainHint){continue};if domainHint!=""&&d.Scope==ScopeSpecific&&d.Domain!=domainHint{continue};score:=0;if req.PreferDeterministic&&d.Deterministic{score+=100};if domainHint!=""&&d.Scope==ScopeSpecific&&d.Domain==domainHint{score+=50};if d.Scope==ScopeGeneral{score+=25};if d.ParameterCount==0{score+=10};candidates=append(candidates,candidate{worker:w,desc:d,score:score})};if len(candidates)==0{return nil,fmt.Errorf("no worker satisfies capability=%s scope=%s domain=%s",capability,scopeHint,domainHint)};sort.Slice(candidates,func(i,j int)bool{if candidates[i].score!=candidates[j].score{return candidates[i].score>candidates[j].score};pi,pj:=candidates[i].desc.ParameterCount,candidates[j].desc.ParameterCount;if pi!=pj{if pi==0{return true};if pj==0{return false};return pi<pj};return candidates[i].desc.ID<candidates[j].desc.ID});return candidates[0].worker,nil}
