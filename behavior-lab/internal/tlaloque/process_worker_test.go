package tlaloque

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func processDesc()CapabilityDescriptor{return CapabilityDescriptor{ID:"proc",Capability:"TEST",Scope:ScopeGeneral,Engine:EngineProcess,InputSchema:"json",OutputSchema:"json",MaxConcurrency:1}}
func processReq()CapabilityRequest{return CapabilityRequest{TaskID:"t",NodeID:"n",Input:json.RawMessage(`{}`)}}

func TestProcessWorkerAcceptsSingleStrictResponse(t *testing.T){w:=ProcessWorker{Desc:processDesc(),Command:[]string{"sh","-c",`cat >/dev/null; printf '%s\n' '{"output":{"ok":true}}'`}};resp,err:=w.Execute(context.Background(),processReq());if err!=nil{t.Fatal(err)};if resp.WorkerID!="proc"||string(resp.Output)!=`{"ok":true}`{t.Fatalf("resp=%+v",resp)}}
func TestProcessWorkerRejectsIdentityMismatch(t *testing.T){w:=ProcessWorker{Desc:processDesc(),Command:[]string{"sh","-c",`cat >/dev/null; printf '%s\n' '{"worker_id":"other","output":{}}'`}};_,err:=w.Execute(context.Background(),processReq());if err==nil||!strings.Contains(err.Error(),"identity mismatch"){t.Fatalf("err=%v",err)}}
func TestProcessWorkerRejectsInvalidAndTrailingOutput(t *testing.T){for _,script:=range []string{`cat >/dev/null; printf 'not-json\n'`,`cat >/dev/null; printf '%s\n%s\n' '{"output":{}}' '{"extra":1}'`}{w:=ProcessWorker{Desc:processDesc(),Command:[]string{"sh","-c",script}};if _,err:=w.Execute(context.Background(),processReq());err==nil{t.Fatalf("expected error for %q",script)}}}
func TestProcessWorkerTimeoutAndPartialExit(t *testing.T){w:=ProcessWorker{Desc:processDesc(),Command:[]string{"sh","-c",`cat >/dev/null; sleep 1`},Timeout:20*time.Millisecond};if _,err:=w.Execute(context.Background(),processReq());err==nil{t.Fatal("expected timeout")};w=ProcessWorker{Desc:processDesc(),Command:[]string{"sh","-c",`cat >/dev/null; printf '{"output":'; exit 2`}};if _,err:=w.Execute(context.Background(),processReq());err==nil{t.Fatal("expected partial process failure")}}
