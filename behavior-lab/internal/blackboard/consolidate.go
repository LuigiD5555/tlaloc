package blackboard

import (
	"encoding/json"
	"fmt"
	"sort"
)

const (
	ConsensusConfirmed = "CONFIRMED"
	ConsensusUnknown   = "UNKNOWN"
)

type ValueContract func(json.RawMessage) bool

type Consensus struct {
	Key           string          `json:"key"`
	Status        string          `json:"status"`
	Value         json.RawMessage `json:"value,omitempty"`
	Votes         int             `json:"votes"`
	RequiredVotes int             `json:"required_votes"`
	ObservationIDs []string       `json:"observation_ids,omitempty"`
}

func canonicalJSON(raw json.RawMessage) (string, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Consolidate keeps contradictory observations and derives a decision without
// deleting any of them. A contract violation, tie, or lack of a >=2/3 majority
// is UNKNOWN.
func Consolidate(snapshot Snapshot, key string, contract ValueContract) (Consensus, error) {
	if snapshot.Schema != SnapshotSchema {
		return Consensus{}, fmt.Errorf("unexpected snapshot schema %q", snapshot.Schema)
	}
	type bucket struct {
		raw json.RawMessage
		ids []string
	}
	buckets := map[string]*bucket{}
	total := 0
	invalid := false
	for _, e := range snapshot.Entries {
		if e.Type != EntryObservation || e.Key != key {
			continue
		}
		total++
		if contract != nil && !contract(e.Value) {
			invalid = true
			continue
		}
		canon, err := canonicalJSON(e.Value)
		if err != nil {
			invalid = true
			continue
		}
		b := buckets[canon]
		if b == nil {
			b = &bucket{raw: append(json.RawMessage(nil), e.Value...)}
			buckets[canon] = b
		}
		b.ids = append(b.ids, e.ID)
	}
	required := 0
	if total > 0 {
		required = (2*total + 2) / 3
	}
	unknown := Consensus{Key: key, Status: ConsensusUnknown, RequiredVotes: required}
	if total == 0 || invalid {
		return unknown, nil
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	bestVotes := 0
	bestKey := ""
	tied := false
	for _, k := range keys {
		n := len(buckets[k].ids)
		if n > bestVotes {
			bestVotes, bestKey, tied = n, k, false
		} else if n == bestVotes {
			tied = true
		}
	}
	unknown.Votes = bestVotes
	if tied || bestVotes < required {
		return unknown, nil
	}
	winner := buckets[bestKey]
	sort.Strings(winner.ids)
	return Consensus{Key: key, Status: ConsensusConfirmed, Value: winner.raw, Votes: bestVotes, RequiredVotes: required, ObservationIDs: winner.ids}, nil
}
