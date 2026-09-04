package exocortex

import (
	"fmt"
	"sort"
	"strings"
)

// AdapterR1 consults a frozen CapabilityProfileR1 BEFORE inference and
// returns a deterministic decision: which preventive input transformations
// to apply, or a pre-inference rejection. It never touches a model prompt,
// ground truth, a scorer verdict, or a benchmark base id (protocol §12-§15).
type AdapterR1 struct {
	Profile CapabilityProfileR1
}

// AdaptRequestR1 carries ONLY observable runtime input properties. There is
// deliberately no field for base_id, expected answer, dataset family, or a
// prior scorer verdict.
type AdaptRequestR1 struct {
	Opcode                  string
	HasVisualOperand        bool
	LineHeightPx            float64 // measured containing-line height in submitted px; 0 = unknown/NA
	VisualFieldName         string  // TARGET | LINE | BLOCK | LOCAL_256 | LOCAL_384 | VIEWPORT | FULL_PAGE
	CompetingNumericOperands int    // detected competing numeric operands near the operand
	ValueCueTighterThanPadded bool
	IdenticalToAPriorFailedCall bool // set by the caller's dedupe layer, never by a scorer
}

// AdaptTransform is one deterministic preventive transformation.
type AdaptTransform struct {
	Kind string `json:"kind"`
	From string `json:"from"`
	To   string `json:"to"`
}

// AdaptDecisionR1 is the deterministic decision trace for one invocation
// (protocol §13). It is emitted for every adapted call.
type AdaptDecisionR1 struct {
	RequestedOpcode         string            `json:"requested_opcode"`
	ProfileVersion          string            `json:"profile_version"`
	ProfileHash             string            `json:"profile_hash"`
	DetectedInputProperties map[string]any    `json:"detected_input_properties"`
	RulesConsidered         []string          `json:"rules_considered"`
	RulesApplied            []string          `json:"rules_applied"`
	Transformations         []AdaptTransform  `json:"transformations"`
	ResultingWorkingSet     map[string]any    `json:"resulting_working_set"`
	ModelCallCount          int               `json:"model_call_count"`
	Rejected                bool              `json:"rejected"`
	RejectionReason         string            `json:"rejection_reason,omitempty"`
	FallbackAction          string            `json:"fallback_action,omitempty"`
}

var visualOpcodesR1 = map[string]bool{
	OpExtractNumber:          true,
	"READ_ASSOCIATED_NUMBER": true,
}

// Prepare evaluates the frozen profile against the observable request and
// returns the decision. Hard invariants (protocol §14):
//  1. exactly one cognitive opcode (the request carries one);
//  2. no identical retry (rejected here, zero model calls);
//  3. visual opcode with missing visual input -> zero model calls;
//  4. low-scale detectable input -> preventive scale transform;
//  5. formatting/normalization is external (never in this path);
//  6. profile rules are read-only during the call;
//  7. no ground truth is consulted;
//  8. no benchmark base_id special cases.
func (a AdapterR1) Prepare(req AdaptRequestR1) (AdaptDecisionR1, error) {
	op := strings.ToUpper(strings.TrimSpace(req.Opcode))
	dec := AdaptDecisionR1{
		RequestedOpcode: op,
		ProfileVersion:  a.Profile.ProfileVersion,
		ProfileHash:     a.Profile.ProfileHash,
		DetectedInputProperties: map[string]any{
			"has_visual_operand":          req.HasVisualOperand,
			"line_height_px":              req.LineHeightPx,
			"visual_field_name":           strings.ToUpper(strings.TrimSpace(req.VisualFieldName)),
			"competing_numeric_operands":  req.CompetingNumericOperands,
			"value_cue_tighter_than_padded": req.ValueCueTighterThanPadded,
			"identical_to_a_prior_failed_call": req.IdenticalToAPriorFailedCall,
		},
		ResultingWorkingSet: map[string]any{},
		ModelCallCount:      1,
	}

	if op == "" {
		return dec, fmt.Errorf("adapter r1: opcode is required (one cognitive opcode per call)")
	}
	known := false
	for _, k := range a.Profile.KnownOpcodes() {
		if k == op {
			known = true
		}
	}
	if !known {
		dec.RulesConsidered = append(dec.RulesConsidered, "opcode_coverage")
		dec.Rejected = true
		dec.ModelCallCount = 0
		dec.RejectionReason = fmt.Sprintf("opcode %s has no R1 evidence for this executor", op)
		dec.FallbackAction = "UNKNOWN"
		return dec, nil
	}

	// Rule: EXACT_RETRY — never repeat an identical failed call.
	if r, ok := a.Profile.RecoveryRule("EXACT_RETRY"); ok {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
		if req.IdenticalToAPriorFailedCall {
			dec.RulesApplied = append(dec.RulesApplied, r.ID)
			dec.Rejected = true
			dec.ModelCallCount = 0
			dec.RejectionReason = "identical to a prior failed call; blind retry is proven inert (R1-F)"
			dec.FallbackAction = "UNKNOWN"
			return dec, nil
		}
	}

	// Rule: MISSING_VISUAL_OPERAND — visual opcode with no visual operand.
	if r, ok := a.Profile.RecoveryRule("MISSING_VISUAL_OPERAND"); ok && visualOpcodesR1[op] {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
		if !req.HasVisualOperand {
			dec.RulesApplied = append(dec.RulesApplied, r.ID)
			dec.Rejected = true
			dec.ModelCallCount = 0
			dec.RejectionReason = "visual opcode invoked with no visual operand"
			dec.FallbackAction = "UNSUPPORTED"
			return dec, nil
		}
	}

	// Rule: LOW_SCALE — upscale to the preferred scale before the call.
	targetLineHeight := req.LineHeightPx
	if r, ok := a.Profile.RecoveryRule("LOW_SCALE"); ok && op == OpExtractNumber {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
		safe := float64(a.Profile.ExtractNumber.FormalSafeScalePx)
		pref := a.Profile.ExtractNumber.PreferredScalePx
		if req.LineHeightPx > 0 && req.LineHeightPx < safe {
			dec.RulesApplied = append(dec.RulesApplied, r.ID)
			dec.Transformations = append(dec.Transformations, AdaptTransform{
				Kind: "UPSCALE_TO_PREFERRED", From: fmt.Sprintf("%.0fpx", req.LineHeightPx), To: fmt.Sprintf("%dpx", pref),
			})
			targetLineHeight = float64(pref)
		}
	}
	dec.ResultingWorkingSet["target_line_height_px"] = targetLineHeight

	// Rule: HIGH_CONTEXT — prefer the smallest sufficient working set
	// (PREVENTIVE_PRACTICE, not an earned recovery).
	field := strings.ToUpper(strings.TrimSpace(req.VisualFieldName))
	resultField := field
	if r, ok := a.Profile.RecoveryRule("HIGH_CONTEXT"); ok {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
		wide := map[string]bool{"BLOCK": true, "LOCAL_256": true, "LOCAL_384": true, "VIEWPORT": true, "FULL_PAGE": true}
		if wide[field] {
			dec.RulesApplied = append(dec.RulesApplied, r.ID)
			dec.Transformations = append(dec.Transformations, AdaptTransform{Kind: "CROP_TO_OPERAND_LINE", From: field, To: "LINE"})
			resultField = "LINE"
		}
	}
	if resultField != "" {
		dec.ResultingWorkingSet["visual_field"] = resultField
	}

	// Rule: NUMERIC_COMPETITORS — isolate the operand for association reads.
	if r, ok := a.Profile.RecoveryRule("NUMERIC_COMPETITORS"); ok && op == "READ_ASSOCIATED_NUMBER" {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
		if req.CompetingNumericOperands >= 1 {
			dec.RulesApplied = append(dec.RulesApplied, r.ID)
			dec.Transformations = append(dec.Transformations, AdaptTransform{
				Kind: "ISOLATE_OPERAND", From: fmt.Sprintf("%d competing numeric operands", req.CompetingNumericOperands), To: "0 competing numeric operands",
			})
			dec.ResultingWorkingSet["competing_numeric_operands"] = 0
		}
	}

	// Rule: VALUE_CUE — renderer must emit the frozen padded cue.
	if r, ok := a.Profile.RecoveryRule("VALUE_CUE"); ok {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
		if req.ValueCueTighterThanPadded {
			dec.RulesApplied = append(dec.RulesApplied, r.ID)
			dec.Transformations = append(dec.Transformations, AdaptTransform{Kind: "USE_PADDED_VALUE_CUE", From: "TIGHT", To: "PADDED"})
			dec.ResultingWorkingSet["value_cue"] = "PADDED"
		}
	}

	// CURRENT_TESSERACT_OCR is a DO_NOT_ROUTE rule, considered but never an action here.
	if r, ok := a.Profile.RecoveryRule("CURRENT_TESSERACT_OCR"); ok {
		dec.RulesConsidered = append(dec.RulesConsidered, r.ID)
	}

	sort.Strings(dec.RulesConsidered)
	dec.RulesConsidered = dedupeStrings(dec.RulesConsidered)
	dec.ModelCallCount = 1
	return dec, nil
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
