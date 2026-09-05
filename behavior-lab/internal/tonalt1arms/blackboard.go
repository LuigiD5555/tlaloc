package tonalt1arms

import "fmt"

// NodeStatus is the terminal (or in-flight) state of one Blackboard node
// execution. The five terminal values are exactly the trace states the
// task's call accounting requires so every planned slot in a run is always
// accountable, whether or not it actually reached the network:
// COMPLETED (here: Done), FAILED_TRANSPORT, FAILED_SCHEMA, FAILED_CONTRACT,
// BLOCKED_BY_DEPENDENCY.
type NodeStatus string

const (
	NodeStatusPending             NodeStatus = "PENDING"
	NodeStatusRunning             NodeStatus = "RUNNING"
	NodeStatusDone                NodeStatus = "DONE"
	NodeStatusFailedTransport     NodeStatus = "FAILED_TRANSPORT"
	NodeStatusFailedSchema        NodeStatus = "FAILED_SCHEMA"
	NodeStatusFailedContract      NodeStatus = "FAILED_CONTRACT"
	NodeStatusBlockedByDependency NodeStatus = "BLOCKED_BY_DEPENDENCY"
)

// NodeRecord is the Blackboard's in-memory record of one DAG node's
// execution for one workflow run: capability/operation/dependencies (from
// the shared ShapeDAG), the executor that ran it, its observed input/output
// values, its terminal status, and accounting (model-call delta, latency).
type NodeRecord struct {
	NodeID         string
	Capability     string
	Operation      string
	ExecutorID     string
	DependsOn      []string
	Inputs         map[string]float64
	Outputs        map[string]float64
	OutputVerdict  string // set instead of/alongside Outputs for verdict-producing nodes (COMPARE_NUMBERS-family)
	Status         NodeStatus
	ModelCallDelta int // 1 if this node made a real/fake Parrot adapter call, else 0
	LatencyMS      int64
	Error          string
}

// Blackboard is a small, purpose-built, in-memory working-state store scoped
// to exactly one workflow's DAG run. It is deliberately NOT
// internal/blackboard's content-addressed, cross-run durable store -- that
// package answers a different question (a persistent, append-only,
// Fact-promotion log shared across runs); this one only needs to track "did
// this DAG node finish, and what did it produce" for the duration of a
// single Arm B/C execution and its counterfactual replays.
type Blackboard struct {
	WorkflowID string
	Nodes      map[string]*NodeRecord
	order      []string // insertion order, for deterministic iteration/serialization
	promoted   bool
	promotedBy string
	finalValue float64
}

// NewBlackboard creates an empty Blackboard for one workflow run.
func NewBlackboard(workflowID string) *Blackboard {
	return &Blackboard{
		WorkflowID: workflowID,
		Nodes:      make(map[string]*NodeRecord),
	}
}

// Require returns the named node's record, failing closed (an error, not a
// zero value) if the node is missing or not DONE. Used by any consumer that
// needs an upstream node's output before it can proceed (executors,
// counterfactual replay).
func (b *Blackboard) Require(nodeID string) (*NodeRecord, error) {
	rec, ok := b.Nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("tonalt1arms: Blackboard.Require: node %q has no record", nodeID)
	}
	if rec.Status != NodeStatusDone {
		return nil, fmt.Errorf("tonalt1arms: Blackboard.Require: node %q status is %s, want DONE", nodeID, rec.Status)
	}
	return rec, nil
}

// Record stores one node's execution result. If the node declares
// dependencies, every one of them must already be recorded and DONE --
// otherwise Record itself marks this node BLOCKED_BY_DEPENDENCY (overriding
// whatever Status the caller passed) rather than accepting a result that
// was computed on top of a missing/failed upstream. This is the mechanical
// enforcement behind "dependencies must be enforced; missing required
// upstream state must fail closed."
func (b *Blackboard) Record(rec NodeRecord) error {
	if rec.NodeID == "" {
		return fmt.Errorf("tonalt1arms: Blackboard.Record: NodeID is empty")
	}
	if _, exists := b.Nodes[rec.NodeID]; !exists {
		b.order = append(b.order, rec.NodeID)
	}

	for _, dep := range rec.DependsOn {
		depRec, ok := b.Nodes[dep]
		if !ok || depRec.Status != NodeStatusDone {
			blocked := rec
			blocked.Status = NodeStatusBlockedByDependency
			blocked.ModelCallDelta = 0
			blocked.Error = fmt.Sprintf("upstream dependency %q is not DONE", dep)
			stored := blocked
			b.Nodes[rec.NodeID] = &stored
			return nil
		}
	}

	stored := rec
	b.Nodes[rec.NodeID] = &stored
	return nil
}

// PromoteFinal marks the workflow's terminal value as promoted from the
// named node. Only a node whose Operation is VERIFY may promote -- mirroring
// the frozen "only VERIFY may promote FACT" semantics (T1_D7_PROTOCOL.json).
// For shapes with no VERIFY node (has_verify=false; see ShapeDAG), the
// terminal value is read directly from the terminal node's own Outputs by
// the caller instead -- PromoteFinal is never called for those shapes, and
// that is a correct, documented difference in scoring path, not an omission.
func (b *Blackboard) PromoteFinal(nodeID string) error {
	rec, ok := b.Nodes[nodeID]
	if !ok {
		return fmt.Errorf("tonalt1arms: Blackboard.PromoteFinal: node %q has no record", nodeID)
	}
	if rec.Operation != OpVerify {
		return fmt.Errorf("tonalt1arms: Blackboard.PromoteFinal: node %q has Operation %q, only VERIFY may promote", nodeID, rec.Operation)
	}
	if rec.Status != NodeStatusDone {
		return fmt.Errorf("tonalt1arms: Blackboard.PromoteFinal: node %q status is %s, want DONE", nodeID, rec.Status)
	}
	val, ok := rec.Outputs["final"]
	if !ok {
		return fmt.Errorf("tonalt1arms: Blackboard.PromoteFinal: node %q has no \"final\" output to promote", nodeID)
	}
	b.promoted = true
	b.promotedBy = nodeID
	b.finalValue = val
	return nil
}

// Promoted reports whether PromoteFinal has succeeded, and the promoted
// value/promoting node id.
func (b *Blackboard) Promoted() (value float64, nodeID string, ok bool) {
	return b.finalValue, b.promotedBy, b.promoted
}

// Clone deep-copies the Blackboard so a counterfactual mutation never
// touches the original run's recorded state.
func (b *Blackboard) Clone() *Blackboard {
	clone := &Blackboard{
		WorkflowID: b.WorkflowID,
		Nodes:      make(map[string]*NodeRecord, len(b.Nodes)),
		order:      append([]string(nil), b.order...),
		promoted:   b.promoted,
		promotedBy: b.promotedBy,
		finalValue: b.finalValue,
	}
	for id, rec := range b.Nodes {
		copied := *rec
		copied.DependsOn = append([]string(nil), rec.DependsOn...)
		if rec.Inputs != nil {
			copied.Inputs = make(map[string]float64, len(rec.Inputs))
			for k, v := range rec.Inputs {
				copied.Inputs[k] = v
			}
		}
		if rec.Outputs != nil {
			copied.Outputs = make(map[string]float64, len(rec.Outputs))
			for k, v := range rec.Outputs {
				copied.Outputs[k] = v
			}
		}
		clone.Nodes[id] = &copied
	}
	return clone
}

// OrderedNodeIDs returns node IDs in the order they were first recorded --
// used for deterministic trace serialization.
func (b *Blackboard) OrderedNodeIDs() []string {
	return append([]string(nil), b.order...)
}
