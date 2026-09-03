package exocortex

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// VerifyTlaloqueID is the CapabilityDescriptor.ID this Tlaloque registers
// under. It is the only Tlaloque that may write a blackboard.Fact (E0.12):
// a model may never promote its own Observation.
const VerifyTlaloqueID = "verify-tlaloque"

// VerifyInput names the Observation to check and the contract it must
// satisfy. VerifyTlaloque never widens or invents this contract at
// runtime; it is fixed per Step by the T0 recipe.
type VerifyInput struct {
	TargetKey      string   `json:"target_key"`
	FactID         string   `json:"fact_id"`
	ExpectedType   string   `json:"expected_type"` // number | text | choice
	MinValue       *float64 `json:"min_value,omitempty"`
	MaxValue       *float64 `json:"max_value,omitempty"`
	AllowedChoices []string `json:"allowed_choices,omitempty"`
}

// VerifyTlaloque promotes an Observation to a Fact only when it satisfies
// an output-type/range/provenance contract; otherwise it reports
// UNSUPPORTED rather than fabricating success (E0.12, section 12 "Verify
// Tlaloque").
type VerifyTlaloque struct{}

func (VerifyTlaloque) Descriptor() tlaloque.CapabilityDescriptor {
	d, _ := tlaloque.CapabilityDescriptor{
		ID: VerifyTlaloqueID, Capability: OpVerify, Engine: tlaloque.EngineDeterministic,
		InputSchema: "exocortex.verify-input.r0", OutputSchema: "tlaloc.blackboard.fact.r0",
		Deterministic: true, MaxConcurrency: 8,
	}.Normalize()
	return d
}

func (VerifyTlaloque) Execute(_ context.Context, req tlaloque.CapabilityRequest) (tlaloque.CapabilityResponse, error) {
	var in VerifyInput
	if err := json.Unmarshal(req.Input, &in); err != nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("verify tlaloque: decode input: %w", err)
	}
	in.TargetKey = strings.TrimSpace(in.TargetKey)
	in.FactID = strings.TrimSpace(in.FactID)
	if in.TargetKey == "" {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("verify tlaloque: target_key is required")
	}
	if in.FactID == "" {
		in.FactID = in.TargetKey
	}
	if req.Blackboard == nil {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("verify tlaloque: requires a blackboard snapshot")
	}

	source, ok := latestObservationEntry(*req.Blackboard, in.TargetKey)
	if !ok {
		return tlaloque.CapabilityResponse{}, fmt.Errorf("verify tlaloque: no OBSERVATION found for target_key %q", in.TargetKey)
	}

	fact := blackboard.Fact{FactID: in.FactID}
	switch strings.ToLower(strings.TrimSpace(in.ExpectedType)) {
	case TargetTypeNumber:
		value, ok := extractNumericValue(source.Value)
		switch {
		case !ok:
			fact.Status = blackboard.FactUnsupported
			fact.VerificationNotes = "observation did not contain a parseable number"
		case in.MinValue != nil && value < *in.MinValue:
			fact.Status = blackboard.FactUnsupported
			fact.VerificationNotes = fmt.Sprintf("value %v below min_value %v", value, *in.MinValue)
		case in.MaxValue != nil && value > *in.MaxValue:
			fact.Status = blackboard.FactUnsupported
			fact.VerificationNotes = fmt.Sprintf("value %v above max_value %v", value, *in.MaxValue)
		default:
			fact.Status = blackboard.FactVerified
			fact.Value, _ = json.Marshal(value)
		}
		if fact.Value == nil {
			fact.Value, _ = json.Marshal(nil)
		}
	case TargetTypeChoice:
		text, ok := extractTextValue(source.Value)
		if !ok || text == "" {
			fact.Status = blackboard.FactUnsupported
			fact.VerificationNotes = "observation did not contain text"
			fact.Value, _ = json.Marshal(nil)
		} else if len(in.AllowedChoices) > 0 && !containsString(upperAll(in.AllowedChoices), strings.ToUpper(text)) {
			fact.Status = blackboard.FactUnsupported
			fact.VerificationNotes = fmt.Sprintf("choice %q is not one of %v", text, in.AllowedChoices)
			fact.Value, _ = json.Marshal(nil)
		} else {
			fact.Status = blackboard.FactVerified
			fact.Value, _ = json.Marshal(text)
		}
	case TargetTypeText, "":
		text, ok := extractTextValue(source.Value)
		if !ok || text == "" {
			fact.Status = blackboard.FactUnsupported
			fact.VerificationNotes = "observation did not contain text"
			fact.Value, _ = json.Marshal(nil)
		} else {
			fact.Status = blackboard.FactVerified
			fact.Value, _ = json.Marshal(text)
		}
	default:
		return tlaloque.CapabilityResponse{}, fmt.Errorf("verify tlaloque: unsupported expected_type %q", in.ExpectedType)
	}

	obs, err := blackboard.FactObservation([]blackboard.Entry{source}, fact)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	body, err := json.Marshal(fact)
	if err != nil {
		return tlaloque.CapabilityResponse{}, err
	}
	return tlaloque.CapabilityResponse{WorkerID: VerifyTlaloqueID, Output: body, Confidence: 1, Observations: []blackboard.Observation{obs}}, nil
}

func latestObservationEntry(snapshot blackboard.Snapshot, key string) (blackboard.Entry, bool) {
	var latest blackboard.Entry
	found := false
	for _, e := range snapshot.Entries {
		if e.Type == blackboard.EntryObservation && e.Key == key {
			if !found || e.RecordedAt > latest.RecordedAt {
				latest = e
				found = true
			}
		}
	}
	return latest, found
}

func extractNumericValue(raw json.RawMessage) (float64, bool) {
	var direct float64
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
			return v, true
		}
		return 0, false
	}
	var normalized NormalizeOutput
	if err := json.Unmarshal(raw, &normalized); err == nil && normalized.IsNumber {
		return normalized.AsNumber, true
	}
	return 0, false
}

func extractTextValue(raw json.RawMessage) (string, bool) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text), true
	}
	var normalized NormalizeOutput
	if err := json.Unmarshal(raw, &normalized); err == nil && normalized.Trimmed != "" {
		return normalized.Trimmed, true
	}
	return "", false
}

func upperAll(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strings.ToUpper(strings.TrimSpace(v))
	}
	return out
}
