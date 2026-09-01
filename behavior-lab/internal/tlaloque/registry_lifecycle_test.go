package tlaloque

import "testing"

func TestRegistryActivationAndRollback(t *testing.T) {
	registry := NewRegistry()
	versionOne := testWorker{desc: generalDescriptor("intent-v1", "INTENT")}
	versionTwo := testWorker{desc: generalDescriptor("intent-v2", "INTENT")}
	mustRegister(t, registry, versionOne, versionTwo)

	if err := registry.Activate("intent-v1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Activate("intent-v2"); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select(SelectionRequest{Capability: "INTENT"})
	if err != nil {
		t.Fatal(err)
	}
	if selected.Descriptor().ID != "intent-v2" {
		t.Fatalf("selected %q, want intent-v2", selected.Descriptor().ID)
	}
	if err := registry.Unregister("intent-v2"); err == nil {
		t.Fatal("active worker was unregistered")
	}
	if err := registry.Unregister("intent-v1"); err == nil {
		t.Fatal("rollback worker was unregistered")
	}
	rolledBackID, err := registry.Rollback("intent")
	if err != nil {
		t.Fatal(err)
	}
	if rolledBackID != "intent-v1" {
		t.Fatalf("rolled back to %q, want intent-v1", rolledBackID)
	}
	if err := registry.Unregister("intent-v2"); err != nil {
		t.Fatal(err)
	}
}

func TestPinnedInactiveRegistryVersionRemainsAddressable(t *testing.T) {
	registry := NewRegistry()
	mustRegister(t, registry,
		testWorker{desc: generalDescriptor("intent-v1", "INTENT")},
		testWorker{desc: generalDescriptor("intent-v2", "INTENT")},
	)
	if err := registry.Activate("intent-v2"); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select(SelectionRequest{Capability: "INTENT", WorkerID: "intent-v1"})
	if err != nil {
		t.Fatal(err)
	}
	if selected.Descriptor().ID != "intent-v1" {
		t.Fatalf("selected %q, want pinned intent-v1", selected.Descriptor().ID)
	}
}
