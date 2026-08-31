package learningmemory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct { Root string }

func DefaultRoot() string {
	if x := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); x != "" {
		return filepath.Join(x, "tlaloc", "learning-memory")
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home)=="" { return filepath.Join(".", ".tlaloc-learning-memory") }
	return filepath.Join(home, ".local", "state", "tlaloc", "learning-memory")
}

func New(root string) Store {
	if strings.TrimSpace(root)=="" { root=DefaultRoot() }
	return Store{Root:root}
}

func (s Store) Put(e Event) (bool, Event, error) {
	if e.Schema=="" { e.Schema=EventSchema }
	if e.RecordedAt=="" { e.RecordedAt=time.Now().UTC().Format(time.RFC3339Nano) }
	id, err := ContentID(e); if err!=nil { return false,e,err }
	if e.EventID!="" && e.EventID!=id { return false,e,fmt.Errorf("event_id mismatch: %s != %s",e.EventID,id) }
	e.EventID=id
	if err:=Validate(e); err!=nil { return false,e,err }
	dir:=filepath.Join(s.Root,"events")
	if err:=os.MkdirAll(dir,0o700); err!=nil { return false,e,err }
	final:=filepath.Join(dir,id+".json")
	if _,err:=os.Stat(final); err==nil { return false,e,nil } else if !errors.Is(err,os.ErrNotExist) { return false,e,err }
	body,err:=json.MarshalIndent(e,"","  "); if err!=nil{return false,e,err}; body=append(body,'\n')
	tmp,err:=os.CreateTemp(dir,".event-*.tmp"); if err!=nil{return false,e,err}
	tmpName:=tmp.Name(); defer os.Remove(tmpName)
	if err:=tmp.Chmod(0o600); err!=nil { tmp.Close(); return false,e,err }
	if _,err:=tmp.Write(body); err!=nil { tmp.Close(); return false,e,err }
	if err:=tmp.Sync(); err!=nil { tmp.Close(); return false,e,err }
	if err:=tmp.Close(); err!=nil { return false,e,err }
	if err:=os.Link(tmpName,final); err!=nil {
		if _,statErr:=os.Stat(final); statErr==nil { return false,e,nil }
		return false,e,err
	}
	return true,e,nil
}

func (s Store) PutAll(events []Event) (added, skipped int, out []Event, err error) {
	for _,e:=range events{
		ok,stored,putErr:=s.Put(e); if putErr!=nil{return added,skipped,out,putErr}
		out=append(out,stored); if ok{added++}else{skipped++}
	}
	return
}

func (s Store) LoadAll() ([]Event,error) {
	dir:=filepath.Join(s.Root,"events")
	entries,err:=os.ReadDir(dir)
	if errors.Is(err,os.ErrNotExist){return []Event{},nil}; if err!=nil{return nil,err}
	names:=[]string{}; for _,e:=range entries{if !e.IsDir()&&strings.HasSuffix(e.Name(),".json"){names=append(names,e.Name())}}; sort.Strings(names)
	out:=make([]Event,0,len(names))
	for _,name:=range names{
		body,err:=os.ReadFile(filepath.Join(dir,name)); if err!=nil{return nil,err}
		var e Event; dec:=json.NewDecoder(strings.NewReader(string(body))); dec.DisallowUnknownFields(); if err:=dec.Decode(&e);err!=nil{return nil,fmt.Errorf("%s: %w",name,err)}
		if err:=Validate(e);err!=nil{return nil,fmt.Errorf("%s: %w",name,err)}
		id,err:=ContentID(e); if err!=nil{return nil,err}; if e.EventID!=id{return nil,fmt.Errorf("%s: content id mismatch",name)}
		out=append(out,e)
	}
	return out,nil
}

func ContentID(e Event)(string,error){
	e.EventID=""; e.RecordedAt=""
	body,err:=json.Marshal(e); if err!=nil{return "",err}
	sum:=sha256.Sum256(body); return hex.EncodeToString(sum[:]),nil
}

func Validate(e Event) error {
	if e.Schema!=EventSchema{return fmt.Errorf("unexpected schema %q",e.Schema)}
	switch e.EventType{
	case EventObservation:
		if e.BenchmarkID==""||e.TrialID==""||e.QuestionID==""{return fmt.Errorf("observation requires benchmark_id, trial_id and question_id")}
		if e.Pass==nil{return fmt.Errorf("observation requires pass")}
	case EventChange:
		if e.CandidateID==""||strings.TrimSpace(e.ChangeSummary)==""{return fmt.Errorf("change requires candidate_id and change_summary")}
		if len(e.ParentEventIDs)==0{return fmt.Errorf("change requires parent_event_ids")}
	case EventOutcome:
		if e.CandidateID==""{return fmt.Errorf("outcome requires candidate_id")}
		if len(e.ParentEventIDs)<2{return fmt.Errorf("outcome requires change and post-change evidence parent ids")}
		if e.BeforeScore==nil||e.AfterScore==nil{return fmt.Errorf("outcome requires before_score and after_score")}
	default:return fmt.Errorf("unknown event_type %q",e.EventType)
	}
	switch e.EvidenceClass{case EvidenceRealModel,EvidenceSynthetic,EvidenceCI,EvidenceManual:default:return fmt.Errorf("unknown evidence_class %q",e.EvidenceClass)}
	return nil
}
