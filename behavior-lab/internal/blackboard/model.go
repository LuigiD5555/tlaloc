package blackboard

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	EntrySchema    = "tlaloc.blackboard.r0.entry"
	SnapshotSchema = "tlaloc.blackboard.r0.snapshot"

	EntryObservation = "OBSERVATION"
	EntryDecision    = "DECISION"
	EntryFailure     = "FAILURE"
	EntryMetric      = "METRIC"
)

// Observation is the bounded structure a worker may return. Identity and run
// metadata are deliberately absent: SwarmRunner is the only authority allowed
// to turn observations into persisted blackboard entries.
type Observation struct {
	Key        string            `json:"key"`
	Value      json.RawMessage   `json:"value"`
	Confidence float64           `json:"confidence,omitempty"`
	References []string          `json:"references,omitempty"`
	Provenance map[string]string `json:"provenance,omitempty"`
}

// Entry is immutable once persisted. ID is derived from every semantic field
// except RecordedAt, so replaying the same observation is idempotent.
type Entry struct {
	Schema     string            `json:"schema"`
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	RunID      string            `json:"run_id"`
	TaskID     string            `json:"task_id"`
	NodeID     string            `json:"node_id"`
	WorkerID   string            `json:"worker_id"`
	Key        string            `json:"key"`
	Value      json.RawMessage   `json:"value"`
	Confidence float64           `json:"confidence,omitempty"`
	References []string          `json:"references,omitempty"`
	Provenance map[string]string `json:"provenance,omitempty"`
	RecordedAt string            `json:"recorded_at,omitempty"`
}

type Snapshot struct {
	Schema  string  `json:"schema"`
	RunID   string  `json:"run_id"`
	Entries []Entry `json:"entries"`
}

func NormalizeObservation(o Observation) (Observation, error) {
	o.Key = strings.TrimSpace(o.Key)
	if o.Key == "" {
		return Observation{}, fmt.Errorf("observation key is required")
	}
	if len(o.Value) == 0 || !json.Valid(o.Value) {
		return Observation{}, fmt.Errorf("observation %q value must be valid JSON", o.Key)
	}
	if o.Confidence < 0 || o.Confidence > 1 {
		return Observation{}, fmt.Errorf("observation %q confidence must be between 0 and 1", o.Key)
	}
	o.References = normalizeStrings(o.References)
	return o, nil
}

func ValidateEntry(e Entry) error {
	if e.Schema != EntrySchema {
		return fmt.Errorf("unexpected blackboard schema %q", e.Schema)
	}
	switch e.Type {
	case EntryObservation, EntryDecision, EntryFailure, EntryMetric:
	default:
		return fmt.Errorf("unknown blackboard entry type %q", e.Type)
	}
	if strings.TrimSpace(e.RunID) == "" || strings.TrimSpace(e.TaskID) == "" || strings.TrimSpace(e.NodeID) == "" || strings.TrimSpace(e.WorkerID) == "" || strings.TrimSpace(e.Key) == "" {
		return fmt.Errorf("blackboard entry requires run_id, task_id, node_id, worker_id and key")
	}
	if len(e.Value) == 0 || !json.Valid(e.Value) {
		return fmt.Errorf("blackboard entry %q value must be valid JSON", e.Key)
	}
	if e.Confidence < 0 || e.Confidence > 1 {
		return fmt.Errorf("blackboard entry %q confidence must be between 0 and 1", e.Key)
	}
	return nil
}

func normalizeStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		s := strings.TrimSpace(raw)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
