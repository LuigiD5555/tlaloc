package executor

import (
	"context"
	"fmt"
	"testing"

	"tlaloc.local/behaviorlab/internal/action"
)

// fakeFS is an in-memory stand-in for the filesystem: the set of paths
// that "exist". No real OS is touched.
type fakeFS struct{ exists map[string]bool }

func newFakeFS(paths ...string) *fakeFS {
	fs := &fakeFS{exists: map[string]bool{}}
	for _, path := range paths {
		fs.exists[path] = true
	}
	return fs
}

func (fs *fakeFS) checker() Checker {
	return func(_ context.Context, kind, arg string, args map[string]string) (bool, error) {
		path := args[arg]
		switch kind {
		case "path_exists":
			return fs.exists[path], nil
		case "path_absent":
			return !fs.exists[path], nil
		default:
			return false, fmt.Errorf("unknown check %q", kind)
		}
	}
}

func (fs *fakeFS) moveImpl() Impl {
	return func(_ context.Context, args map[string]string) error {
		src, dst := args["source"], args["destination"]
		if !fs.exists[src] {
			return fmt.Errorf("source missing")
		}
		delete(fs.exists, src)
		fs.exists[dst] = true
		return nil
	}
}

// brokenMoveImpl reports success but does nothing — the case the
// postcondition check exists to catch.
func brokenMoveImpl() Impl {
	return func(_ context.Context, _ map[string]string) error { return nil }
}

func moveAction(t *testing.T) action.ActionIR {
	t.Helper()
	act, err := action.Compile(
		action.ActionCandidate{Capability: "FILE.MOVE", Arguments: map[string]string{
			"source": "/a/x.pdf", "destination": "/b/x.pdf",
		}},
		action.DefaultCatalog(),
		action.Policy{MaxRisk: action.R4Privileged},
	)
	if err != nil {
		t.Fatalf("compile move: %v", err)
	}
	return act
}

func TestExecute_PositivePathVerifies(t *testing.T) {
	fs := newFakeFS("/a/x.pdf")
	ex := Executor{Impls: map[string]Impl{"FILE.MOVE": fs.moveImpl()}, Check: fs.checker()}

	result, err := ex.Execute(context.Background(), moveAction(t))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Executed || !result.Verified || result.RolledBack {
		t.Fatalf("expected executed+verified, no rollback: %+v", result)
	}
	if fs.exists["/a/x.pdf"] || !fs.exists["/b/x.pdf"] {
		t.Errorf("world not in the expected state: %+v", fs.exists)
	}
}

func TestExecute_PreconditionFailsNothingRuns(t *testing.T) {
	fs := newFakeFS("/a/x.pdf", "/b/x.pdf") // destination already exists
	ex := Executor{Impls: map[string]Impl{"FILE.MOVE": fs.moveImpl()}, Check: fs.checker()}

	result, _ := ex.Execute(context.Background(), moveAction(t))
	if result.Executed {
		t.Fatal("the move must not run when destination_absent fails")
	}
	if result.Failure == "" {
		t.Error("a blocked action should record why")
	}
}

func TestExecute_PostconditionFailTriggersRollback(t *testing.T) {
	fs := newFakeFS("/a/x.pdf")
	ex := Executor{Impls: map[string]Impl{"FILE.MOVE": brokenMoveImpl()}, Check: fs.checker()}

	result, err := ex.Execute(context.Background(), moveAction(t))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Executed || result.Verified {
		t.Fatalf("a no-op impl must execute but fail verification: %+v", result)
	}
	if !result.RolledBack {
		t.Fatal("a failed verification with a rollback present must roll back")
	}
	// The rollback (destination -> source, both no-ops here) restores the
	// original precondition state: source present, destination absent.
	if !result.RollbackVerified {
		t.Errorf("rollback should have restored precondition state: %+v", result)
	}
}

func TestExecute_UnknownCapabilityErrors(t *testing.T) {
	ex := Executor{Impls: map[string]Impl{}, Check: newFakeFS().checker()}
	if _, err := ex.Execute(context.Background(), action.ActionIR{Capability: "FILE.MOVE"}); err == nil {
		t.Error("an unregistered capability implementation must be an error")
	}
}
