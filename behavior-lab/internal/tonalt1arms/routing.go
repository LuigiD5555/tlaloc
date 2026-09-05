package tonalt1arms

import "fmt"

// Binding is one capability's routing decision for one arm: which executor
// identity handles it, and whether that executor is the Parrot adapter.
type Binding struct {
	Capability string
	ExecutorID string
	UsesParrot bool
}

// armBExecutorIDs/armCExecutorIDs give a stable, human-readable executor
// identity per capability, distinct between the two arms even where the
// underlying implementation might look similar, so a trace can always show
// which arm's binding table produced a given node's ExecutorID.
const (
	armBParrotExecutorID      = "arm-b-parrot-adapter"
	armBDeterministicExecutor = "arm-b-deterministic"
	armCParrotExecutorID      = "arm-c-parrot-adapter"
	armCDeterministicExecutor = "arm-c-deterministic"
)

// BuildArmBBindings builds Arm B's immutable capability->executor binding
// table straight from the frozen T1_D5_ARM_B_POLICY.json: every capability
// in policy.ParrotAdapters routes to the Parrot adapter; every capability in
// policy.DeterministicNodes routes to Arm B's own deterministic executor
// (LOCATE_REGION/CROP_REGION/VERIFY -- geometry and a final finite/DONE
// check, never arithmetic). The returned map is a fresh map literal owned
// solely by the caller -- there is no shared mutable state between this and
// any other call, so one arm's executor structurally cannot see or mutate
// another arm's bindings (task requirement: NOT just relying on the type
// system, but no aliasing at all).
func BuildArmBBindings(policy *ArmBPolicy) (map[string]Binding, error) {
	if policy == nil {
		return nil, fmt.Errorf("tonalt1arms: BuildArmBBindings: nil policy")
	}
	bindings := make(map[string]Binding, len(policy.ParrotAdapters)+len(policy.DeterministicNodes))
	for capability := range policy.ParrotAdapters {
		bindings[capability] = Binding{Capability: capability, ExecutorID: armBParrotExecutorID, UsesParrot: true}
	}
	for _, capability := range policy.DeterministicNodes {
		bindings[capability] = Binding{Capability: capability, ExecutorID: armBDeterministicExecutor, UsesParrot: false}
	}
	return bindings, nil
}

// BuildArmCBindings builds Arm C's immutable capability->executor binding
// table straight from the frozen T1_D5_ARM_C_POLICY.json's executor_routing
// map: "parrot_required" routes to the Parrot adapter; every other value
// ("deterministic", "deterministic_preferred", "deterministic_required")
// routes to Arm C's own deterministic executor. Per the frozen policy, only
// EXTRACT_NUMBER is parrot_required -- everything else, including
// NORMALIZE/COMPARE_NUMBERS/ARITHMETIC (which Arm B routes to Parrot), is
// deterministic here. Independent map literal, same isolation guarantee as
// BuildArmBBindings.
func BuildArmCBindings(policy *ArmCPolicy) (map[string]Binding, error) {
	if policy == nil {
		return nil, fmt.Errorf("tonalt1arms: BuildArmCBindings: nil policy")
	}
	bindings := make(map[string]Binding, len(policy.ExecutorRouting))
	for capability, mode := range policy.ExecutorRouting {
		usesParrot := mode == "parrot_required"
		executorID := armCDeterministicExecutor
		if usesParrot {
			executorID = armCParrotExecutorID
		}
		bindings[capability] = Binding{Capability: capability, ExecutorID: executorID, UsesParrot: usesParrot}
	}
	return bindings, nil
}
