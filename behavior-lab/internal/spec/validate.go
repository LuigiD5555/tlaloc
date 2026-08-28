package spec

import (
	"errors"
	"fmt"
)

func Validate(s BehaviorSpec) error {
	if s.Version == "" || s.ID == "" { return errors.New("version and id are required") }
	if len(s.StateKinds) == 0 { return errors.New("at least one state kind is required") }
	if len(s.Operations) == 0 { return errors.New("at least one operation is required") }
	seen := map[InvariantCode]bool{}
	for _, inv := range s.Invariants { if inv.Code == "" { return errors.New("invariant code is required") }; if seen[inv.Code] { return fmt.Errorf("duplicate invariant: %s", inv.Code) }; seen[inv.Code] = true }
	if s.Output.Format == "" { return errors.New("output format is required") }
	return nil
}
