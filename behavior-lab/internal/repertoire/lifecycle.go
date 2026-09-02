// Package repertoire is the lifecycle machinery for Tlaloque specialists.
// Tlaloc's existing promotion path only handles admission; a swarm that
// only ever adds specialists accumulates stale and harmful ones (Thinking
// Swarms ch.14). This package adds the other direction — demotion,
// retirement, archival — while never deleting the evidence history.
//
// It also fixes the canonical shape all of Tlaloc's learning subsystems
// should share (they need not merge, only agree on this):
//
//	Observation -> Experience -> Assessment -> Candidate -> Shadow ->
//	Promotion -> Monitoring -> Demotion
//
// learningmemory produces Observations/Experiences; learningcycle /
// experimentpolicy produce Candidates; promotion + calibration gate
// Shadow->Active; this package owns the phases and the transitions between
// them.
package repertoire

import (
	"fmt"
	"sort"
	"time"
)

const Schema = "tlaloc.repertoire-entry.r0"

// Phase is where a specialist sits in the repertoire. The privileged
// runtime only ever dispatches ACTIVE specialists; everything else is
// pipeline or archive.
type Phase string

const (
	Experimental Phase = "EXPERIMENTAL" // being tried, not measured against gates
	Candidate    Phase = "CANDIDATE"    // proposed for the repertoire, under assessment
	Shadow       Phase = "SHADOW"       // runs alongside ACTIVE, results compared, not trusted
	Active       Phase = "ACTIVE"       // dispatched for real work
	Degraded     Phase = "DEGRADED"     // still ACTIVE-eligible but flagged; recoverable
	Retired      Phase = "RETIRED"      // removed from the active repertoire
	Archived     Phase = "ARCHIVED"     // terminal; kept only as history
)

// allowedTransitions is the whole state machine. Anything not listed is
// rejected.
var allowedTransitions = map[Phase]map[Phase]bool{
	Experimental: {Candidate: true, Archived: true},
	Candidate:    {Shadow: true, Experimental: true, Archived: true},
	Shadow:       {Active: true, Candidate: true, Archived: true},
	Active:       {Degraded: true, Retired: true},
	Degraded:     {Active: true, Retired: true},
	Retired:      {Archived: true},
	Archived:     {},
}

// Event is one immutable entry in a specialist's history.
type Event struct {
	At          time.Time `json:"at"`
	From        Phase     `json:"from"`
	To          Phase     `json:"to"`
	Reason      string    `json:"reason"`
	EvidenceRef string    `json:"evidence_ref,omitempty"`
}

// Entry is a specialist's position in the repertoire plus its full,
// append-only history. History is never truncated — a retired specialist
// keeps every event that got it there.
type Entry struct {
	Schema    string  `json:"schema"`
	WorkerID  string  `json:"worker_id"`
	Version   string  `json:"version"`
	Phase     Phase   `json:"phase"`
	History   []Event `json:"history"`
	UpdatedAt string  `json:"updated_at"`
}

// New starts a specialist in EXPERIMENTAL with a single genesis event.
func New(workerID, version, reason string, at time.Time) Entry {
	return Entry{
		Schema:   Schema,
		WorkerID: workerID,
		Version:  version,
		Phase:    Experimental,
		History: []Event{{
			At: at.UTC(), From: "", To: Experimental, Reason: reason,
		}},
		UpdatedAt: at.UTC().Format(time.RFC3339Nano),
	}
}

// Transition moves an entry to a new phase if the state machine allows it,
// returning a new Entry with the event appended. The input entry is not
// mutated and its history slice is copied, so callers cannot corrupt an
// audit trail by aliasing.
func (entry Entry) Transition(to Phase, reason, evidenceRef string, at time.Time) (Entry, error) {
	if entry.Phase == to {
		return Entry{}, fmt.Errorf("repertoire: %s is already %s", entry.WorkerID, to)
	}
	if !allowedTransitions[entry.Phase][to] {
		return Entry{}, fmt.Errorf("repertoire: %s -> %s is not an allowed transition for %s", entry.Phase, to, entry.WorkerID)
	}
	if reason == "" {
		return Entry{}, fmt.Errorf("repertoire: a transition to %s needs a reason", to)
	}

	next := entry
	next.History = make([]Event, len(entry.History), len(entry.History)+1)
	copy(next.History, entry.History)
	next.History = append(next.History, Event{
		At: at.UTC(), From: entry.Phase, To: to, Reason: reason, EvidenceRef: evidenceRef,
	})
	next.Phase = to
	next.UpdatedAt = at.UTC().Format(time.RFC3339Nano)
	return next, nil
}

// EnteredActiveAt returns when the specialist most recently became ACTIVE,
// or the zero time if it never has. Used by monitoring to age out unused
// ACTIVE specialists.
func (entry Entry) EnteredActiveAt() time.Time {
	var last time.Time
	for _, event := range entry.History {
		if event.To == Active {
			last = event.At
		}
	}
	return last
}

// PhaseCounts summarizes a set of entries by phase — the repertoire's
// health at a glance.
func PhaseCounts(entries []Entry) map[Phase]int {
	counts := map[Phase]int{}
	for _, entry := range entries {
		counts[entry.Phase]++
	}
	return counts
}

// ActiveWorkerIDs returns the sorted ids of every ACTIVE specialist — the
// only ones a runtime should dispatch.
func ActiveWorkerIDs(entries []Entry) []string {
	ids := []string{}
	for _, entry := range entries {
		if entry.Phase == Active {
			ids = append(ids, entry.WorkerID)
		}
	}
	sort.Strings(ids)
	return ids
}
