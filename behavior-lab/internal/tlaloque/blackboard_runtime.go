package tlaloque

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

type BlackboardRuntime struct { Store blackboard.Store; RunID string }
type runBlackboardWriter struct { mu sync.Mutex; store blackboard.Store; runID string; taskID string }
func newRunBlackboardWriter(cfg *BlackboardRuntime,taskID string)*runBlackboardWriter{if cfg==nil{return nil};runID:=strings.TrimSpace(cfg.RunID);if runID==""{runID=strings.TrimSpace(taskID)};return &runBlackboardWriter{store:cfg.Store,runID:runID,taskID:taskID}}
func(w *runBlackboardWriter)Snapshot()(*blackboard.Snapshot,error){if w==nil{return nil,nil};w.mu.Lock();defer w.mu.Unlock();s,err:=w.store.Snapshot(w.runID);if err!=nil{return nil,err};return &s,nil}
func(w *runBlackboardWriter)append(e blackboard.Entry)error{if w==nil{return nil};_,_,err:=w.store.Append(e);return err}
func(w *runBlackboardWriter)RecordSelectionFailure(node SwarmNode,err error)error{if w==nil||err==nil{return nil};w.mu.Lock();defer w.mu.Unlock();value,_:=json.Marshal(map[string]any{"error":err.Error()});return w.append(blackboard.Entry{Type:blackboard.EntryFailure,RunID:w.runID,TaskID:w.taskID,NodeID:node.ID,WorkerID:"UNRESOLVED",Key:"node.selection_failure",Value:value})}
func(w *runBlackboardWriter)RecordNode(ex NodeExecution,observations []blackboard.Observation)error{if w==nil{return nil};w.mu.Lock();defer w.mu.Unlock();workerID:=ex.WorkerID;if strings.TrimSpace(workerID)==""{workerID="UNRESOLVED"};for _,raw:=range observations{o,err:=blackboard.NormalizeObservation(raw);if err!=nil{return fmt.Errorf("node %s worker %s: %w",ex.NodeID,workerID,err)};entryType:=blackboard.EntryObservation;if strings.Contains(strings.ToUpper(ex.Capability),"CONSOLIDATE")&&strings.HasPrefix(strings.ToLower(o.Key),"decision."){entryType=blackboard.EntryDecision};if err:=w.append(blackboard.Entry{Type:entryType,RunID:w.runID,TaskID:w.taskID,NodeID:ex.NodeID,WorkerID:workerID,Key:o.Key,Value:o.Value,Confidence:o.Confidence,References:o.References,Provenance:o.Provenance});err!=nil{return err}};if ex.Error!=""{value,_:=json.Marshal(map[string]any{"error":ex.Error});if err:=w.append(blackboard.Entry{Type:blackboard.EntryFailure,RunID:w.runID,TaskID:w.taskID,NodeID:ex.NodeID,WorkerID:workerID,Key:"node.failure",Value:value});err!=nil{return err}};metric:=struct{DurationMS int64 `json:"duration_ms"`;Error string `json:"error,omitempty"`;Confidence float64 `json:"confidence,omitempty"`;Output json.RawMessage `json:"output,omitempty"`}{DurationMS:ex.DurationMS,Error:ex.Error,Confidence:ex.Confidence,Output:ex.Output};value,err:=json.Marshal(metric);if err!=nil{return err};return w.append(blackboard.Entry{Type:blackboard.EntryMetric,RunID:w.runID,TaskID:w.taskID,NodeID:ex.NodeID,WorkerID:workerID,Key:"node.execution",Value:value,Confidence:ex.Confidence})}
