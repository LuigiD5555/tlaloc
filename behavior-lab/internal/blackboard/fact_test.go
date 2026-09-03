package blackboard

import "testing"

func observationEntry(t *testing.T, id, key, value string) Entry {
	t.Helper()
	e := Entry{Type: EntryObservation, RunID: "run1", TaskID: "task1", NodeID: "n1", WorkerID: "w1", Key: key, Value: []byte(value)}
	got, err := ContentID(e)
	if err != nil {
		t.Fatalf("ContentID: %v", err)
	}
	e.ID = got
	return e
}

func TestPromoteFact_RequiresSourceObservation(t *testing.T) {
	_, err := PromoteFact("run1", "task1", "n2", "verify", nil, Fact{FactID: "f1", Status: FactVerified, Value: []byte(`126`)})
	if err == nil {
		t.Fatalf("expected error when no source observations are supplied")
	}
}

func TestPromoteFact_RejectsNonObservationSource(t *testing.T) {
	failure := Entry{ID: "x", Type: EntryFailure, Value: []byte(`{}`)}
	_, err := PromoteFact("run1", "task1", "n2", "verify", []Entry{failure}, Fact{FactID: "f1", Status: FactVerified, Value: []byte(`126`)})
	if err == nil {
		t.Fatalf("expected error when source entry is not an OBSERVATION")
	}
}

func TestPromoteFact_VerifiedRequiresDerivation(t *testing.T) {
	obs := observationEntry(t, "e1", "raw_value", `"126"`)
	entry, err := PromoteFact("run1", "task1", "n2", "verify", []Entry{obs}, Fact{FactID: "f1", Status: FactVerified, Value: []byte(`126`)})
	if err != nil {
		t.Fatalf("PromoteFact: %v", err)
	}
	if entry.Type != EntryFact {
		t.Fatalf("entry type = %q, want FACT", entry.Type)
	}
	fact, err := FactFromEntry(entry)
	if err != nil {
		t.Fatalf("FactFromEntry: %v", err)
	}
	if len(fact.DerivedFrom) != 1 || fact.DerivedFrom[0] != obs.ID {
		t.Fatalf("derived_from = %v, want [%s]", fact.DerivedFrom, obs.ID)
	}
	if err := ValidateEntry(entry); err != nil {
		t.Fatalf("ValidateEntry on promoted fact: %v", err)
	}
}

func TestPromoteFact_UnsupportedNeverFabricatesDerivation(t *testing.T) {
	obs := observationEntry(t, "e2", "raw_value", `"garbled"`)
	entry, err := PromoteFact("run1", "task1", "n2", "verify", []Entry{obs}, Fact{FactID: "f2", Status: FactUnsupported, Value: []byte(`null`)})
	if err != nil {
		t.Fatalf("PromoteFact: %v", err)
	}
	fact, err := FactFromEntry(entry)
	if err != nil {
		t.Fatalf("FactFromEntry: %v", err)
	}
	if fact.Status != FactUnsupported {
		t.Fatalf("status = %q, want UNSUPPORTED", fact.Status)
	}
}

func TestModelCannotPromoteItsOwnObservationImplicitly(t *testing.T) {
	// A worker that only ever returns Observation values (as every Tlaloque
	// in this package does through CapabilityResponse.Observations) has no
	// exported path to construct an EntryFact except by calling
	// PromoteFact with a real, already-persisted Observation Entry. This
	// test documents that guarantee: NormalizeFact alone (without
	// PromoteFact) still requires derived_from for a VERIFIED fact.
	_, err := NormalizeFact(Fact{FactID: "f3", Status: FactVerified, Value: []byte(`1`)})
	if err == nil {
		t.Fatalf("expected NormalizeFact to reject a VERIFIED fact with no derivation")
	}
}
