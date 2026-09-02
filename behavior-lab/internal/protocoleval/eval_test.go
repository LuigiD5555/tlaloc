package protocoleval

import "testing"

func sampleState() SemanticState {
	return SemanticState{
		Entities:    []string{"Sun", "Earth", "Moon"},
		Relations:   []Relation{{Subject: "Earth", Predicate: "orbits", Object: "Sun"}, {Subject: "Moon", Predicate: "orbits", Object: "Earth"}},
		Hierarchy:   []Relation{{Subject: "Moon", Predicate: "member_of", Object: "Earth system"}},
		Evidence:    []string{"observation-1"},
		Uncertainty: []string{"none declared"},
	}
}

func TestPerfectMultiHopPasses(t *testing.T) {
	s := sampleState()
	res := Evaluate(Trial{
		Schema: SchemaR0 + ".trial", ID: "perfect", Mode: ModeMultiHop,
		RequiredDecoder: "S2", RequiredEncoder: "E2", Source: s,
		Hops: []Hop{
			{ModelID: "model-a", DecoderCodec: "S2", EncoderCodec: "E2", Decoded: s, Reencoded: s, ModelOutput: "Used declared S2 then E2."},
			{ModelID: "model-b", DecoderCodec: "S2", EncoderCodec: "E2", Decoded: s, Reencoded: s, ModelOutput: "S2 read; E2 write."},
		},
	})
	if !res.Pass {
		t.Fatalf("perfect trial failed: %+v", res)
	}
	if res.FinalSemanticDrift != 0 || res.MeanSemanticDrift != 0 {
		t.Fatalf("unexpected drift: %+v", res)
	}
	if res.CrossModelReadSuccess != 1 || res.CrossModelWriteSuccess != 1 {
		t.Fatalf("success rate drift: %+v", res)
	}
}

func TestInventedRelationFails(t *testing.T) {
	s := sampleState()
	bad := sampleState()
	bad.Relations = append(bad.Relations, Relation{Subject: "Mars", Predicate: "orbits", Object: "Earth"})
	res := Evaluate(Trial{ID: "invent", Mode: ModeRoundTrip, RequiredDecoder: "S2", RequiredEncoder: "E2", Source: s, Hops: []Hop{{ModelID: "m", DecoderCodec: "S2", EncoderCodec: "E2", Decoded: bad, Reencoded: bad}}})
	if res.Pass {
		t.Fatal("invented relation should fail")
	}
	if res.Hops[0].InventedReadFactRate == 0 || res.Hops[0].InventedWriteFactRate == 0 {
		t.Fatalf("invention not measured: %+v", res.Hops[0])
	}
}

func TestUndeclaredExternalDecoderFails(t *testing.T) {
	s := sampleState()
	res := Evaluate(Trial{ID: "external", Mode: ModeRead, RequiredDecoder: "S2", Source: s, Hops: []Hop{{ModelID: "m", DecoderCodec: "S2", Decoded: s, ModelOutput: "I need an external decoder before I can read the payload."}}})
	if res.Pass {
		t.Fatal("external codec dependency should fail")
	}
	if !res.Hops[0].UndeclaredExternalCodecDependency {
		t.Fatal("dependency violation not detected")
	}
}

func TestSemanticToExactEscalationFails(t *testing.T) {
	s := sampleState()
	res := Evaluate(Trial{ID: "exact", Mode: ModeRead, RequiredDecoder: "S2", Source: s, Hops: []Hop{{ModelID: "m", DecoderCodec: "S2", Decoded: s, ModelOutput: "First decode the binary and decompress the residual."}}})
	if res.Pass {
		t.Fatal("semantic-to-exact escalation should fail")
	}
	if !res.Hops[0].SemanticToExactEscalation {
		t.Fatal("exact escalation not detected")
	}
}

func TestMissingSemanticFactMeasuresDrift(t *testing.T) {
	s := sampleState()
	partial := sampleState()
	partial.Relations = partial.Relations[:1]
	res := Evaluate(Trial{ID: "loss", Mode: ModeRead, RequiredDecoder: "S2", Source: s, Hops: []Hop{{ModelID: "m", DecoderCodec: "S2", Decoded: partial}}})
	if res.FinalSemanticDrift <= 0 {
		t.Fatalf("expected positive drift: %+v", res)
	}
	if res.Hops[0].ReadPreservation.Relations >= 1 {
		t.Fatalf("relation loss not measured: %+v", res.Hops[0])
	}
}
