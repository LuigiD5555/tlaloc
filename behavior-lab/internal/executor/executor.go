// Package executor runs an authorized action.ActionIR against the world
// and verifies it actually happened — the "world" level of the
// Verification Spine, and the far side of the deterministic boundary.
//
// It is deliberately mechanical: check every precondition, run exactly one
// capability implementation, check every expected postcondition, and roll
// back if verification fails. It has no model, no planning, no discretion.
// Capability implementations and the world-state checker are injected, so
// this package is fully testable without touching a real OS and a real
// deployment supplies real ones.
package executor

import (
	"context"
	"fmt"

	"tlaloc.local/behaviorlab/internal/action"
)

// Impl performs one capability against the world. It must be deterministic
// in effect and must not partially apply — either it does the whole thing
// or it returns an error having changed nothing.
type Impl func(ctx context.Context, args map[string]string) error

// Checker reports whether a named world-state predicate holds. kind and arg
// come from an action.Precondition / action.Postcondition; args is the
// action's full argument map for context.
type Checker func(ctx context.Context, kind, arg string, args map[string]string) (bool, error)

// Executor holds the injected capability implementations and world checker.
type Executor struct {
	Impls map[string]Impl
	Check Checker
}

// CheckResult records one precondition/postcondition evaluation.
type CheckResult struct {
	Kind   string `json:"kind"`
	Arg    string `json:"arg,omitempty"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// Result is the full account of an execution attempt.
type Result struct {
	ActionID         string        `json:"action_id"`
	Capability       string        `json:"capability"`
	Risk             string        `json:"risk"`
	Preconditions    []CheckResult `json:"preconditions"`
	Executed         bool          `json:"executed"`
	Postconditions   []CheckResult `json:"postconditions"`
	Verified         bool          `json:"verified"`
	RolledBack       bool          `json:"rolled_back"`
	RollbackVerified bool          `json:"rollback_verified,omitempty"`
	Failure          string        `json:"failure,omitempty"`
}

// Execute is the whole mechanical sequence. It returns a non-nil error only
// when it could not complete the protocol (unknown capability, checker
// blew up); an action that ran but failed verification comes back with
// err == nil and Verified == false (and RolledBack set if a rollback
// existed).
func (executor Executor) Execute(ctx context.Context, act action.ActionIR) (Result, error) {
	result := Result{ActionID: act.ActionID, Capability: act.Capability, Risk: act.RiskName}

	impl, ok := executor.Impls[act.Capability]
	if !ok {
		return result, fmt.Errorf("executor: no implementation registered for capability %q", act.Capability)
	}
	if executor.Check == nil {
		return result, fmt.Errorf("executor: no world checker configured")
	}

	// 1. preconditions — all must hold, or nothing runs.
	for _, pre := range act.Preconditions {
		passed, err := executor.Check(ctx, pre.Kind, pre.Arg, act.Arguments)
		entry := CheckResult{Kind: pre.Kind, Arg: pre.Arg, Passed: passed}
		if err != nil {
			entry.Passed = false
			entry.Error = err.Error()
		}
		result.Preconditions = append(result.Preconditions, entry)
		if !entry.Passed {
			result.Failure = fmt.Sprintf("precondition %q not satisfied", pre.Kind)
			return result, nil
		}
	}

	// 2. execute exactly once.
	if err := impl(ctx, act.Arguments); err != nil {
		result.Failure = "execution failed: " + err.Error()
		return result, nil
	}
	result.Executed = true

	// 3. postconditions — the world-level verification.
	verified := true
	for _, post := range act.ExpectedPostconditions {
		passed, err := executor.Check(ctx, post.Kind, post.Arg, act.Arguments)
		entry := CheckResult{Kind: post.Kind, Arg: post.Arg, Passed: passed}
		if err != nil {
			entry.Passed = false
			entry.Error = err.Error()
		}
		result.Postconditions = append(result.Postconditions, entry)
		if !entry.Passed {
			verified = false
		}
	}
	result.Verified = verified
	if verified {
		return result, nil
	}

	// 4. verification failed — roll back if we can.
	result.Failure = "postcondition verification failed"
	if act.Rollback == nil {
		return result, nil
	}
	rollbackImpl, ok := executor.Impls[act.Rollback.Capability]
	if !ok {
		result.Failure += "; no rollback implementation"
		return result, nil
	}
	if err := rollbackImpl(ctx, act.Rollback.Arguments); err != nil {
		result.Failure += "; rollback also failed: " + err.Error()
		return result, nil
	}
	result.RolledBack = true

	// Verify the rollback restored the precondition state where we can.
	rollbackOK := true
	for _, pre := range act.Preconditions {
		passed, err := executor.Check(ctx, pre.Kind, pre.Arg, act.Arguments)
		if err != nil || !passed {
			rollbackOK = false
		}
	}
	result.RollbackVerified = rollbackOK
	return result, nil
}
