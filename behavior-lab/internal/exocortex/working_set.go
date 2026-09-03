package exocortex

import (
	"encoding/json"
	"fmt"
	"sort"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

// Resolved holds out-of-band operand payloads a Step's Addresses point at
// (e.g. a region crop's file path or bytes) that do not live on the
// Blackboard as Observation/Fact values. The T0 runner populates this from
// the Region Tlaloque's own output; WorkingSetBuilder never reaches outside
// it or the Blackboard snapshot for anything else.
type Resolved map[Address]json.RawMessage

// WorkingSetBuilder deterministically reduces a Blackboard snapshot plus
// the current Step down to the minimal CapabilityRequest input an executor
// needs (E3B, E0.8): a tight crop and one instruction, never workflow
// history, never unrelated Facts, never a full page when a crop already
// exists.
type WorkingSetBuilder struct{}

// Build resolves exactly the addresses the Step declares as inputs, using
// only: (1) the most recent matching Observation/Fact on the snapshot for
// that address's key, or (2) an explicit out-of-band Resolved value. It
// never copies the rest of the snapshot into the result.
func (WorkingSetBuilder) Build(snapshot *blackboard.Snapshot, step Step, resolved Resolved) (map[string]json.RawMessage, error) {
	step, err := step.Normalize()
	if err != nil {
		return nil, err
	}
	out := make(map[string]json.RawMessage, len(step.Inputs))
	for _, addr := range step.Inputs {
		key := string(addr)
		if value, ok := resolved[addr]; ok {
			out[key] = value
			continue
		}
		value, ok := latestObservationOrFact(snapshot, key)
		if !ok {
			return nil, fmt.Errorf("exocortex: step %s: input %s has no resolved value and no matching blackboard entry", step.ID, addr)
		}
		out[key] = value
	}
	return out, nil
}

// latestObservationOrFact returns the most recently recorded OBSERVATION or
// FACT entry whose Key matches, preferring a FACT (verified, when present)
// over a raw Observation for the same key — the working set should hand an
// executor the most authoritative value it has, not force it to re-derive
// what Verify already settled.
func latestObservationOrFact(snapshot *blackboard.Snapshot, key string) (json.RawMessage, bool) {
	if snapshot == nil {
		return nil, false
	}
	entries := append([]blackboard.Entry(nil), snapshot.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].RecordedAt < entries[j].RecordedAt })

	var lastObservation json.RawMessage
	var lastFact json.RawMessage
	for _, e := range entries {
		switch e.Type {
		case blackboard.EntryObservation:
			if e.Key == key {
				lastObservation = e.Value
			}
		case blackboard.EntryFact:
			if e.Key == "fact."+key {
				lastFact = e.Value
			}
		}
	}
	if lastFact != nil {
		return lastFact, true
	}
	if lastObservation != nil {
		return lastObservation, true
	}
	return nil, false
}
