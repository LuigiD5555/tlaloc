package blackboard

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Fact status vocabulary (E5). A Fact starts life only through PromoteFact;
// there is no path that lets a model or a plain Observation become a Fact
// on its own (E0.6, E0.12).
const (
	FactVerified    = "VERIFIED"
	FactUnsupported = "UNSUPPORTED"
	FactUnknown     = "UNKNOWN"
)

// EntryFact is a fourth Entry.Type alongside OBSERVATION/DECISION/FAILURE/
// METRIC. A Fact is persisted as an ordinary content-addressed Entry, not a
// second store or schema (E0.15); ValidateEntry in model.go accepts it.
const EntryFact = "FACT"

// Fact is a typed, verified value derived from one or more Observations. It
// is the only Blackboard payload a Verify Tlaloque may produce (E0.12).
type Fact struct {
	FactID            string            `json:"fact_id"`
	Value             json.RawMessage   `json:"value"`
	Status            string            `json:"status"`
	DerivedFrom       []string          `json:"derived_from"`
	SourceProvenance  map[string]string `json:"source_provenance,omitempty"`
	VerificationNotes string            `json:"verification_notes,omitempty"`
}

// NormalizeFact validates a Fact before it is wrapped into an Entry.
func NormalizeFact(f Fact) (Fact, error) {
	f.FactID = strings.TrimSpace(f.FactID)
	if f.FactID == "" {
		return Fact{}, fmt.Errorf("fact id is required")
	}
	switch f.Status {
	case FactVerified, FactUnsupported, FactUnknown:
	default:
		return Fact{}, fmt.Errorf("fact %q: unknown status %q", f.FactID, f.Status)
	}
	if len(f.Value) == 0 || !json.Valid(f.Value) {
		return Fact{}, fmt.Errorf("fact %q: value must be valid JSON", f.FactID)
	}
	f.DerivedFrom = normalizeStrings(f.DerivedFrom)
	if f.Status == FactVerified && len(f.DerivedFrom) == 0 {
		return Fact{}, fmt.Errorf("fact %q: a VERIFIED fact requires at least one derived_from observation id", f.FactID)
	}
	return f, nil
}

// deriveFact stamps DerivedFrom from real, already-persisted Observation
// entries and validates the result. It is the shared choke point behind
// both PromoteFact and FactObservation: neither can produce a Fact without
// at least one real Observation entry as evidence (E0.6, E0.12).
func deriveFact(sourceObservations []Entry, fact Fact) (Fact, error) {
	if len(sourceObservations) == 0 {
		return Fact{}, fmt.Errorf("promote fact %q: at least one source observation is required", fact.FactID)
	}
	ids := make([]string, 0, len(sourceObservations))
	for _, o := range sourceObservations {
		if o.Type != EntryObservation {
			return Fact{}, fmt.Errorf("promote fact %q: entry %q is not an OBSERVATION", fact.FactID, o.ID)
		}
		if o.ID == "" {
			return Fact{}, fmt.Errorf("promote fact %q: source observation is missing its content id", fact.FactID)
		}
		ids = append(ids, o.ID)
	}
	fact.DerivedFrom = ids
	return NormalizeFact(fact)
}

// PromoteFact is the single choke point through which an Observation-backed
// result may become a Fact (E0.6, E0.12), for a caller writing directly to
// a blackboard.Store. It is called by the Verify Tlaloque only, never by
// the executor that produced the Observation: the caller must supply the
// actual persisted Observation entries being promoted, so a Fact can never
// be fabricated with no derivation.
func PromoteFact(runID, taskID, nodeID, workerID string, sourceObservations []Entry, fact Fact) (Entry, error) {
	normalized, err := deriveFact(sourceObservations, fact)
	if err != nil {
		return Entry{}, err
	}
	value, err := json.Marshal(normalized)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		Schema:   EntrySchema,
		Type:     EntryFact,
		RunID:    strings.TrimSpace(runID),
		TaskID:   strings.TrimSpace(taskID),
		NodeID:   strings.TrimSpace(nodeID),
		WorkerID: strings.TrimSpace(workerID),
		Key:      "fact." + normalized.FactID,
		Value:    value,
	}
	return entry, nil
}

// FactObservation is the Verify Tlaloque's path to producing a Fact through
// the ordinary CapabilityResponse.Observations/SwarmRunner write path
// rather than a direct Store write. BlackboardRuntime.RecordNode
// reclassifies a VERIFY-capability observation keyed "fact.*" as an
// EntryFact instead of an EntryObservation (see blackboard_runtime.go), so
// no second persistence path is introduced (E0.15).
func FactObservation(sourceObservations []Entry, fact Fact) (Observation, error) {
	normalized, err := deriveFact(sourceObservations, fact)
	if err != nil {
		return Observation{}, err
	}
	value, err := json.Marshal(normalized)
	if err != nil {
		return Observation{}, err
	}
	return Observation{Key: "fact." + normalized.FactID, Value: value, Provenance: normalized.SourceProvenance}, nil
}

// FactFromEntry decodes a Fact back out of a FACT entry.
func FactFromEntry(e Entry) (Fact, error) {
	if e.Type != EntryFact {
		return Fact{}, fmt.Errorf("entry %q is not a FACT", e.ID)
	}
	var f Fact
	if err := json.Unmarshal(e.Value, &f); err != nil {
		return Fact{}, fmt.Errorf("decode fact from entry %q: %w", e.ID, err)
	}
	return f, nil
}
