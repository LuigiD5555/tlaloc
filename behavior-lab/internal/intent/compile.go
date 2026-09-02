package intent

import (
	"fmt"
	"strconv"
	"strings"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// CompiledIntent is a validated IntentIR turned into the inputs the
// capability planner accepts, with the non-planning parts (invariants,
// evidence, budget, risk) carried through for downstream enforcement.
type CompiledIntent struct {
	Goals []tlaloque.CapabilityGoal

	Invariants           []Invariant
	SuccessCriteria      []Criterion
	EvidenceRequirements []EvidenceRequirement
	Budget               Budget
	Risk                 RiskProfile

	// MaxLatencyMS is a "max_latency_ms" constraint if one was given; 0
	// means unconstrained. It is advisory to planning and enforced by
	// accounting.
	MaxLatencyMS int64

	// The following feed the deterministic envelope that builds an
	// action.Policy (see internal/envelope). They are carried as plain
	// strings so this package need not import internal/action.
	MaxActionRisk       string   // "" | R0_READ_ONLY .. R4_PRIVILEGED, from "max_action_risk"
	StayInside          []string // path sandbox prefixes, from repeated "stay_inside"
	AllowedCapabilities []string // action-capability allow-list, from repeated "allow_capability"
}

var validRiskLevels = map[string]bool{"low": true, "medium": true, "high": true}
var validActionRisk = map[string]bool{
	"R0_READ_ONLY": true, "R1_LOCAL_REVERSIBLE": true, "R2_LOCAL_IRREVERSIBLE": true,
	"R3_EXTERNAL_EFFECT": true, "R4_PRIVILEGED": true,
}
var validEvidenceLevels = map[EvidenceLevel]bool{EvidenceA: true, EvidenceB: true, EvidenceC: true, EvidenceD: true}

// Compile validates an IntentIR and translates it into CapabilityGoals —
// one per required output — carrying the shared constraints as planner
// hints. It does not select workers or build a DAG; PlanFor does that
// against a registry.
func Compile(ir IntentIR) (CompiledIntent, error) {
	if strings.TrimSpace(ir.Version) == "" {
		return CompiledIntent{}, fmt.Errorf("intent: version is required")
	}
	outputs := normalizeUpper(ir.RequiredOutputs)
	if len(outputs) == 0 {
		return CompiledIntent{}, fmt.Errorf("intent: at least one required_output is needed")
	}
	if ir.Risk.Level != "" && !validRiskLevels[strings.ToLower(ir.Risk.Level)] {
		return CompiledIntent{}, fmt.Errorf("intent: risk level %q is not low|medium|high", ir.Risk.Level)
	}
	for _, requirement := range ir.EvidenceRequirements {
		if !validEvidenceLevels[requirement.MinLevel] {
			return CompiledIntent{}, fmt.Errorf("intent: evidence level %q is not A|B|C|D", requirement.MinLevel)
		}
	}
	if ir.Budget.MaxTokens < 0 || ir.Budget.MaxWallMS < 0 || ir.Budget.MaxUpstreamCalls < 0 {
		return CompiledIntent{}, fmt.Errorf("intent: budget values must not be negative")
	}

	var (
		preferDeterministic bool
		maxParameters       int64
		domainHint          string
		scopeHint           string
		maxLatencyMS        int64
		maxActionRisk       string
		stayInside          []string
		allowedCapabilities []string
	)
	for _, constraint := range ir.Constraints {
		value := strings.TrimSpace(constraint.Value)
		switch strings.ToLower(strings.TrimSpace(constraint.Kind)) {
		case "max_action_risk":
			normalized := strings.ToUpper(value)
			if !validActionRisk[normalized] {
				return CompiledIntent{}, fmt.Errorf("intent: max_action_risk %q is not R0..R4", value)
			}
			maxActionRisk = normalized
		case "stay_inside":
			if value != "" {
				stayInside = append(stayInside, value)
			}
		case "allow_capability":
			if value != "" {
				allowedCapabilities = append(allowedCapabilities, strings.ToUpper(value))
			}
		case "prefer_deterministic":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return CompiledIntent{}, fmt.Errorf("intent: prefer_deterministic %q is not a bool", value)
			}
			preferDeterministic = parsed
		case "max_parameters":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsed < 0 {
				return CompiledIntent{}, fmt.Errorf("intent: max_parameters %q is not a non-negative integer", value)
			}
			maxParameters = parsed
		case "max_latency_ms":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsed < 0 {
				return CompiledIntent{}, fmt.Errorf("intent: max_latency_ms %q is not a non-negative integer", value)
			}
			maxLatencyMS = parsed
		case "domain":
			domainHint = strings.ToUpper(value)
		case "scope":
			scopeHint = strings.ToUpper(value)
		}
	}

	// A high-risk intent should not be silently satisfied by a large
	// probabilistic model when a bounded one exists.
	if strings.ToLower(ir.Risk.Level) == "high" {
		preferDeterministic = true
	}

	available := make([]string, 0, len(ir.Inputs))
	for _, in := range ir.Inputs {
		if kind := strings.TrimSpace(in.Kind); kind != "" {
			available = append(available, kind)
		}
	}

	goals := make([]tlaloque.CapabilityGoal, 0, len(outputs))
	for _, output := range outputs {
		goals = append(goals, tlaloque.CapabilityGoal{
			Capability:          output,
			ScopeHint:           scopeHint,
			DomainHint:          domainHint,
			PreferDeterministic: preferDeterministic,
			MaxParameters:       maxParameters,
			AvailableProducts:   available,
		})
	}

	return CompiledIntent{
		Goals:                goals,
		Invariants:           ir.Invariants,
		SuccessCriteria:      ir.SuccessCriteria,
		EvidenceRequirements: ir.EvidenceRequirements,
		Budget:               ir.Budget,
		Risk:                 ir.Risk,
		MaxLatencyMS:         maxLatencyMS,
		MaxActionRisk:        maxActionRisk,
		StayInside:           stayInside,
		AllowedCapabilities:  allowedCapabilities,
	}, nil
}

func normalizeUpper(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
