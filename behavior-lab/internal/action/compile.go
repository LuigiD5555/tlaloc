package action

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Policy is what a run is allowed to do — narrower than the Catalog. It is
// set by the deterministic envelope from the IntentIR's risk profile and
// constraints, never by a model.
type Policy struct {
	// MaxRisk is the ceiling; an action whose Catalog risk exceeds it is
	// refused.
	MaxRisk RiskClass
	// AllowedCapabilities, when non-empty, is an allow-list. Empty means
	// "any capability in the Catalog up to MaxRisk".
	AllowedCapabilities []string
	// StayInside, when set, requires every path-like argument to be within
	// one of these prefixes.
	StayInside []string
}

func (policy Policy) capabilityAllowed(name string) bool {
	if len(policy.AllowedCapabilities) == 0 {
		return true
	}
	for _, allowed := range policy.AllowedCapabilities {
		if allowed == name {
			return true
		}
	}
	return false
}

func (policy Policy) pathAllowed(path string) bool {
	if len(policy.StayInside) == 0 {
		return true
	}
	clean := strings.TrimSpace(path)
	for _, prefix := range policy.StayInside {
		if clean == prefix || strings.HasPrefix(clean, strings.TrimRight(prefix, "/")+"/") {
			return true
		}
	}
	return false
}

// Compile is the whole authorization boundary. It takes an untrusted
// ActionCandidate and either returns an authorized ActionIR or an error
// explaining the refusal. It never trusts the candidate for anything that
// matters: capability existence, risk class, required arguments, path
// scope, and the risk ceiling are all decided here against the Catalog and
// Policy.
func Compile(candidate ActionCandidate, catalog Catalog, policy Policy) (ActionIR, error) {
	name := strings.TrimSpace(candidate.Capability)
	spec, known := catalog[name]
	if !known {
		return ActionIR{}, fmt.Errorf("action: unknown capability %q", name)
	}
	if !spec.Risk.Valid() {
		return ActionIR{}, fmt.Errorf("action: capability %q has an invalid risk class in the catalog", name)
	}
	if !policy.capabilityAllowed(name) {
		return ActionIR{}, fmt.Errorf("action: capability %q is not on the policy allow-list", name)
	}
	if spec.Risk > policy.MaxRisk {
		return ActionIR{}, fmt.Errorf("action: capability %q is %s, above the policy ceiling %s", name, spec.Risk, policy.MaxRisk)
	}

	args := map[string]string{}
	for _, argSpec := range spec.Args {
		value, present := candidate.Arguments[argSpec.Name]
		value = strings.TrimSpace(value)
		if argSpec.Required && (!present || value == "") {
			return ActionIR{}, fmt.Errorf("action: capability %q requires argument %q", name, argSpec.Name)
		}
		if value == "" {
			continue
		}
		if argSpec.PathLike && !policy.pathAllowed(value) {
			return ActionIR{}, fmt.Errorf("action: argument %q=%q is outside the policy's allowed paths", argSpec.Name, value)
		}
		args[argSpec.Name] = value
	}
	// Reject arguments the capability does not declare — a candidate cannot
	// smuggle extra parameters past the schema.
	declared := map[string]bool{}
	for _, argSpec := range spec.Args {
		declared[argSpec.Name] = true
	}
	unexpected := []string{}
	for provided := range candidate.Arguments {
		if !declared[provided] {
			unexpected = append(unexpected, provided)
		}
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		return ActionIR{}, fmt.Errorf("action: capability %q got undeclared argument(s): %s", name, strings.Join(unexpected, ", "))
	}

	action := ActionIR{
		Schema:                 Schema,
		ActionID:               actionID(name, args),
		Capability:             name,
		Arguments:              args,
		Risk:                   spec.Risk,
		RiskName:               spec.Risk.String(),
		Preconditions:          append([]Precondition(nil), spec.Preconditions...),
		ExpectedPostconditions: append([]Postcondition(nil), spec.Postconditions...),
		ProposedBy:             strings.TrimSpace(candidate.ProposedBy),
	}
	if spec.Reversible {
		action.Rollback = synthesizeRollback(spec, args)
	}
	return action, nil
}

// synthesizeRollback builds the inverse ActionIR for a reversible
// capability by swapping the SwapArgs pair. The rollback is itself a fully
// formed ActionIR so the executor treats it like any other action.
func synthesizeRollback(spec CapabilitySpec, args map[string]string) *ActionIR {
	rollbackArgs := map[string]string{}
	for key, value := range args {
		rollbackArgs[key] = value
	}
	first, second := spec.SwapArgs[0], spec.SwapArgs[1]
	if first != "" && second != "" {
		rollbackArgs[first], rollbackArgs[second] = args[second], args[first]
	}
	return &ActionIR{
		Schema:     Schema,
		ActionID:   actionID(spec.Name+"#rollback", rollbackArgs),
		Capability: spec.Name,
		Arguments:  rollbackArgs,
		Risk:       spec.Risk,
		RiskName:   spec.Risk.String(),
	}
}

func actionID(name string, args map[string]string) string {
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	builder := strings.Builder{}
	builder.WriteString(name)
	for _, key := range keys {
		builder.WriteString("\x00")
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(args[key])
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:8])
}
