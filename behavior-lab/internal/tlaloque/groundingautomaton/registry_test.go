package groundingautomaton

import "testing"

func TestNewRegistryContainsAutomaton(t *testing.T) {
	registry := NewRegistry()
	if _, ok := registry.Get(WorkerID); !ok {
		t.Fatalf("expected %s in registry", WorkerID)
	}
}
