package action

import "sort"

// ArgSpec describes one argument a capability takes.
type ArgSpec struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	// PathLike args are checked against Policy.StayInside.
	PathLike bool `json:"path_like,omitempty"`
}

// CapabilitySpec is the static, deterministic definition of one thing the
// system can do — the "static architecture" (Capability Registry). The
// risk class lives here, not in whatever proposed the action.
type CapabilitySpec struct {
	Name           string          `json:"name"`
	Risk           RiskClass       `json:"risk"`
	Args           []ArgSpec       `json:"args"`
	Preconditions  []Precondition  `json:"preconditions"`
	Postconditions []Postcondition `json:"postconditions"`
	// Reversible capabilities get a synthesized Rollback ActionIR. RollbackOf
	// names the capability that undoes this one (often itself, with
	// swapped args).
	Reversible bool   `json:"reversible,omitempty"`
	RollbackOf string `json:"rollback_of,omitempty"`
	// SwapArgs, for a reversible capability, is the pair of argument names
	// whose values are exchanged to build the rollback (e.g. source <->
	// destination for FILE.MOVE).
	SwapArgs [2]string `json:"swap_args,omitempty"`
}

// Catalog is the set of capabilities a given deployment permits to exist at
// all. It is separate from Policy: the Catalog says what is definable, the
// Policy says what this run may use.
type Catalog map[string]CapabilitySpec

// Names returns the catalog's capability names, sorted.
func (catalog Catalog) Names() []string {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultCatalog is a small illustrative catalog matching the constitution's
// cognitive-syscall sketch. Real deployments supply their own; this is
// what the tests and examples run against.
func DefaultCatalog() Catalog {
	return Catalog{
		"FILE.LIST": {
			Name: "FILE.LIST", Risk: R0ReadOnly,
			Args: []ArgSpec{{Name: "path", Required: true, PathLike: true}},
		},
		"FILE.READ": {
			Name: "FILE.READ", Risk: R0ReadOnly,
			Args:          []ArgSpec{{Name: "path", Required: true, PathLike: true}},
			Preconditions: []Precondition{{Kind: "path_exists", Arg: "path"}},
		},
		"FILE.MOVE": {
			Name: "FILE.MOVE", Risk: R1LocalReversible, Reversible: true,
			Args: []ArgSpec{
				{Name: "source", Required: true, PathLike: true},
				{Name: "destination", Required: true, PathLike: true},
			},
			Preconditions: []Precondition{
				{Kind: "path_exists", Arg: "source"},
				{Kind: "path_absent", Arg: "destination"},
			},
			Postconditions: []Postcondition{
				{Kind: "path_absent", Arg: "source"},
				{Kind: "path_exists", Arg: "destination"},
			},
			SwapArgs: [2]string{"source", "destination"},
		},
		"FILE.DELETE": {
			Name: "FILE.DELETE", Risk: R2LocalIrreversible,
			Args:           []ArgSpec{{Name: "path", Required: true, PathLike: true}},
			Preconditions:  []Precondition{{Kind: "path_exists", Arg: "path"}},
			Postconditions: []Postcondition{{Kind: "path_absent", Arg: "path"}},
		},
		"PROCESS.LIST": {
			Name: "PROCESS.LIST", Risk: R0ReadOnly,
		},
		"SERVICE.RESTART": {
			Name: "SERVICE.RESTART", Risk: R4Privileged,
			Args:           []ArgSpec{{Name: "unit", Required: true}},
			Preconditions:  []Precondition{{Kind: "service_exists", Arg: "unit"}},
			Postconditions: []Postcondition{{Kind: "service_active", Arg: "unit"}},
		},
		"USER.NOTIFY": {
			Name: "USER.NOTIFY", Risk: R0ReadOnly,
			Args: []ArgSpec{{Name: "message", Required: true}},
		},
		"MAIL.SEND": {
			Name: "MAIL.SEND", Risk: R3ExternalEffect,
			Args: []ArgSpec{
				{Name: "to", Required: true},
				{Name: "subject", Required: false},
				{Name: "body", Required: true},
			},
		},
	}
}
