package protocoleval

import (
	"sort"
	"strings"
)

const SchemaR0 = "tlaloc.origami-protocol-interop.r0"

type Mode string

const (
	ModeRead      Mode = "READ"
	ModeWrite     Mode = "WRITE"
	ModeRoundTrip Mode = "ROUNDTRIP"
	ModeMultiHop  Mode = "MULTIHOP"
)

type Relation struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

type SemanticState struct {
	Entities    []string   `json:"entities,omitempty"`
	Relations   []Relation `json:"relations,omitempty"`
	Hierarchy   []Relation `json:"hierarchy,omitempty"`
	Evidence    []string   `json:"evidence,omitempty"`
	Uncertainty []string   `json:"uncertainty,omitempty"`
}

type Hop struct {
	ModelID      string        `json:"model_id"`
	DecoderCodec string        `json:"decoder_codec,omitempty"`
	EncoderCodec string        `json:"encoder_codec,omitempty"`
	Decoded      SemanticState `json:"decoded,omitempty"`
	Reencoded    SemanticState `json:"reencoded,omitempty"`
	ModelOutput  string        `json:"model_output,omitempty"`
}

type Trial struct {
	Schema          string        `json:"schema"`
	ID              string        `json:"id"`
	Mode            Mode          `json:"mode"`
	RequiredDecoder string        `json:"required_decoder,omitempty"`
	RequiredEncoder string        `json:"required_encoder,omitempty"`
	Source          SemanticState `json:"source"`
	Hops            []Hop         `json:"hops"`
}

type Preservation struct {
	Entities    float64 `json:"entities"`
	Relations   float64 `json:"relations"`
	Hierarchy   float64 `json:"hierarchy"`
	Evidence    float64 `json:"evidence"`
	Uncertainty float64 `json:"uncertainty"`
}

type HopResult struct {
	ModelID                           string       `json:"model_id"`
	DecoderDiscovered                 bool         `json:"decoder_discovered"`
	EncoderDiscovered                 bool         `json:"encoder_discovered"`
	ReadPreservation                  Preservation `json:"read_preservation"`
	WritePreservation                 Preservation `json:"write_preservation"`
	ReadSemanticDrift                 float64      `json:"read_semantic_drift"`
	WriteSemanticDrift                float64      `json:"write_semantic_drift"`
	HopToHopDrift                     float64      `json:"hop_to_hop_drift"`
	InventedReadFactRate              float64      `json:"invented_read_fact_rate"`
	InventedWriteFactRate             float64      `json:"invented_write_fact_rate"`
	UndeclaredExternalCodecDependency bool         `json:"undeclared_external_codec_dependency"`
	SemanticToExactEscalation         bool         `json:"semantic_to_exact_escalation"`
	Pass                              bool         `json:"pass"`
}

type Result struct {
	Schema                         string      `json:"schema"`
	TrialID                        string      `json:"trial_id"`
	Mode                           Mode        `json:"mode"`
	Hops                           []HopResult `json:"hops"`
	FinalSemanticDrift             float64     `json:"final_semantic_drift"`
	MeanSemanticDrift              float64     `json:"mean_semantic_drift"`
	CrossModelReadSuccess          float64     `json:"cross_model_read_success"`
	CrossModelWriteSuccess         float64     `json:"cross_model_write_success"`
	UndeclaredCodecDependencyCount int         `json:"undeclared_codec_dependency_count"`
	SemanticToExactEscalationCount int         `json:"semantic_to_exact_escalation_count"`
	Pass                           bool        `json:"pass"`
}

func Evaluate(t Trial) Result {
	result := Result{Schema: SchemaR0 + ".result", TrialID: t.ID, Mode: t.Mode, Pass: len(t.Hops) > 0}
	var prev SemanticState = t.Source
	var driftSum float64
	var readPass, writePass int

	for _, h := range t.Hops {
		hr := HopResult{ModelID: h.ModelID}
		hr.DecoderDiscovered = t.RequiredDecoder == "" || strings.EqualFold(strings.TrimSpace(h.DecoderCodec), strings.TrimSpace(t.RequiredDecoder))
		hr.EncoderDiscovered = t.RequiredEncoder == "" || strings.EqualFold(strings.TrimSpace(h.EncoderCodec), strings.TrimSpace(t.RequiredEncoder))
		hr.ReadPreservation = preservation(t.Source, h.Decoded)
		hr.WritePreservation = preservation(t.Source, h.Reencoded)
		hr.ReadSemanticDrift = semanticDrift(t.Source, h.Decoded)
		hr.WriteSemanticDrift = semanticDrift(t.Source, h.Reencoded)
		hr.HopToHopDrift = semanticDrift(prev, h.Decoded)
		hr.InventedReadFactRate = inventedRate(t.Source, h.Decoded)
		hr.InventedWriteFactRate = inventedRate(t.Source, h.Reencoded)
		upper := normalize(h.ModelOutput)
		hr.UndeclaredExternalCodecDependency = containsAny(upper, []string{
			"NEED AN EXTERNAL DECODER", "NEED EXTERNAL DECODER", "NEED A DECODER", "NEED THE ORIGINAL FILE",
			"NEED THE IMAGE FILE", "NEED ACCESS TO THE FILE", "NEED THE BINARY", "CANNOT READ THE PAYLOAD",
		})
		hr.SemanticToExactEscalation = containsAny(upper, []string{
			"DECODE THE BINARY", "EXTRACT THE BITS", "READ THE BITS", "DECOMPRESS", "BZIP2", "GZIP", "ZSTD",
			" X0 ", " X1 ", " X2 ", " X3 ", " X4 ", " X5 ", "MERKLE PROOF", "VERIFY HASH FIRST",
		})

		readOK := hr.DecoderDiscovered && hr.ReadSemanticDrift <= .05 && hr.InventedReadFactRate == 0
		writeOK := hr.EncoderDiscovered && hr.WriteSemanticDrift <= .05 && hr.InventedWriteFactRate == 0
		switch t.Mode {
		case ModeRead:
			hr.Pass = readOK
		case ModeWrite:
			hr.Pass = writeOK
		case ModeRoundTrip, ModeMultiHop:
			hr.Pass = readOK && writeOK
		default:
			hr.Pass = false
		}
		if hr.UndeclaredExternalCodecDependency || hr.SemanticToExactEscalation {
			hr.Pass = false
		}
		if readOK {
			readPass++
		}
		if writeOK {
			writePass++
		}
		if hr.UndeclaredExternalCodecDependency {
			result.UndeclaredCodecDependencyCount++
		}
		if hr.SemanticToExactEscalation {
			result.SemanticToExactEscalationCount++
		}
		if !hr.Pass {
			result.Pass = false
		}

		drift := hr.ReadSemanticDrift
		if t.Mode == ModeWrite {
			drift = hr.WriteSemanticDrift
		}
		if t.Mode == ModeRoundTrip || t.Mode == ModeMultiHop {
			drift = max(hr.ReadSemanticDrift, hr.WriteSemanticDrift)
		}
		driftSum += drift
		result.Hops = append(result.Hops, hr)
		if !emptyState(h.Reencoded) {
			prev = h.Reencoded
		} else {
			prev = h.Decoded
		}
	}

	if len(t.Hops) > 0 {
		result.MeanSemanticDrift = driftSum / float64(len(t.Hops))
		result.CrossModelReadSuccess = float64(readPass) / float64(len(t.Hops))
		result.CrossModelWriteSuccess = float64(writePass) / float64(len(t.Hops))
		last := t.Hops[len(t.Hops)-1]
		if t.Mode == ModeRead {
			result.FinalSemanticDrift = semanticDrift(t.Source, last.Decoded)
		} else {
			result.FinalSemanticDrift = semanticDrift(t.Source, last.Reencoded)
		}
	}
	if t.Mode == ModeMultiHop && len(t.Hops) < 2 {
		result.Pass = false
	}
	return result
}

func preservation(want, got SemanticState) Preservation {
	return Preservation{
		Entities:    ratio(setStrings(want.Entities), setStrings(got.Entities)),
		Relations:   ratio(setRelations(want.Relations), setRelations(got.Relations)),
		Hierarchy:   ratio(setRelations(want.Hierarchy), setRelations(got.Hierarchy)),
		Evidence:    ratio(setStrings(want.Evidence), setStrings(got.Evidence)),
		Uncertainty: ratio(setStrings(want.Uncertainty), setStrings(got.Uncertainty)),
	}
}

func semanticDrift(a, b SemanticState) float64 {
	as, bs := atomSet(a), atomSet(b)
	if len(as) == 0 && len(bs) == 0 {
		return 0
	}
	inter := 0
	for k := range as {
		if bs[k] {
			inter++
		}
	}
	union := len(as)
	for k := range bs {
		if !as[k] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return 1 - float64(inter)/float64(union)
}

func inventedRate(want, got SemanticState) float64 {
	w, g := atomSet(want), atomSet(got)
	if len(g) == 0 {
		return 0
	}
	invented := 0
	for k := range g {
		if !w[k] {
			invented++
		}
	}
	return float64(invented) / float64(len(g))
}

func atomSet(s SemanticState) map[string]bool {
	out := map[string]bool{}
	for k := range setStrings(s.Entities) {
		out["E|"+k] = true
	}
	for k := range setRelations(s.Relations) {
		out["R|"+k] = true
	}
	for k := range setRelations(s.Hierarchy) {
		out["H|"+k] = true
	}
	for k := range setStrings(s.Evidence) {
		out["V|"+k] = true
	}
	for k := range setStrings(s.Uncertainty) {
		out["U|"+k] = true
	}
	return out
}

func setStrings(values []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		if n := normalize(v); n != "" {
			out[n] = true
		}
	}
	return out
}

func setRelations(values []Relation) map[string]bool {
	out := map[string]bool{}
	for _, r := range values {
		k := normalize(r.Subject) + "|" + normalize(r.Predicate) + "|" + normalize(r.Object)
		if k != "||" {
			out[k] = true
		}
	}
	return out
}

func ratio(want, got map[string]bool) float64 {
	if len(want) == 0 {
		return 1
	}
	matched := 0
	for k := range want {
		if got[k] {
			matched++
		}
	}
	return float64(matched) / float64(len(want))
}

func emptyState(s SemanticState) bool { return len(atomSet(s)) == 0 }

func containsAny(text string, phrases []string) bool {
	for _, p := range phrases {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}

func normalize(value string) string { return strings.Join(strings.Fields(strings.ToUpper(value)), " ") }
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// CanonicalAtoms is useful to external adapters that want to inspect exactly
// what the deterministic evaluator compares without involving another model.
func CanonicalAtoms(s SemanticState) []string {
	m := atomSet(s)
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
