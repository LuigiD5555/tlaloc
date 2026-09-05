package tonalt1arms

import "testing"

func TestBlackboard_RecordAndRequire(t *testing.T) {
	bb := NewBlackboard("wf-1")
	err := bb.Record(NodeRecord{
		NodeID: "locate_A", Capability: "LOCATE_REGION", Status: NodeStatusDone,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := bb.Require("locate_A")
	if err != nil {
		t.Fatal(err)
	}
	if rec.NodeID != "locate_A" {
		t.Fatalf("got %q", rec.NodeID)
	}
}

func TestBlackboard_RequireMissingFailsClosed(t *testing.T) {
	bb := NewBlackboard("wf-1")
	if _, err := bb.Require("nonexistent"); err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestBlackboard_RequireNotDoneFailsClosed(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "n1", Status: NodeStatusRunning})
	if _, err := bb.Require("n1"); err == nil {
		t.Fatal("expected error for non-DONE node")
	}
}

func TestBlackboard_RecordBlocksOnMissingDependency(t *testing.T) {
	bb := NewBlackboard("wf-1")
	err := bb.Record(NodeRecord{
		NodeID: "norm_A", Status: NodeStatusDone, DependsOn: []string{"read_A"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := bb.Nodes["norm_A"]
	if rec.Status != NodeStatusBlockedByDependency {
		t.Fatalf("status = %s, want BLOCKED_BY_DEPENDENCY", rec.Status)
	}
	if rec.ModelCallDelta != 0 {
		t.Fatalf("ModelCallDelta = %d, want 0 for a blocked node", rec.ModelCallDelta)
	}
}

func TestBlackboard_RecordBlocksOnFailedDependency(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "read_A", Status: NodeStatusFailedTransport})
	_ = bb.Record(NodeRecord{NodeID: "norm_A", Status: NodeStatusDone, DependsOn: []string{"read_A"}})
	if bb.Nodes["norm_A"].Status != NodeStatusBlockedByDependency {
		t.Fatalf("status = %s, want BLOCKED_BY_DEPENDENCY", bb.Nodes["norm_A"].Status)
	}
}

func TestBlackboard_RecordSucceedsWhenDependenciesDone(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "read_A", Status: NodeStatusDone, Outputs: map[string]float64{"A": 95}})
	_ = bb.Record(NodeRecord{NodeID: "norm_A", Status: NodeStatusDone, DependsOn: []string{"read_A"}, Outputs: map[string]float64{"norm_A": 95}})
	if bb.Nodes["norm_A"].Status != NodeStatusDone {
		t.Fatalf("status = %s, want DONE", bb.Nodes["norm_A"].Status)
	}
}

func TestBlackboard_PromoteFinal_OnlyVerify(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "diff", Operation: OpSubtract, Status: NodeStatusDone, Outputs: map[string]float64{"final": 939}})
	if err := bb.PromoteFinal("diff"); err == nil {
		t.Fatal("expected error promoting a non-VERIFY node")
	}

	_ = bb.Record(NodeRecord{NodeID: "verify", Operation: OpVerify, Status: NodeStatusDone, Outputs: map[string]float64{"final": 939}})
	if err := bb.PromoteFinal("verify"); err != nil {
		t.Fatal(err)
	}
	val, nodeID, ok := bb.Promoted()
	if !ok || nodeID != "verify" || val != 939 {
		t.Fatalf("Promoted() = (%v,%v,%v), want (939,verify,true)", val, nodeID, ok)
	}
}

func TestBlackboard_PromoteFinal_RequiresDone(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "verify", Operation: OpVerify, Status: NodeStatusBlockedByDependency})
	if err := bb.PromoteFinal("verify"); err == nil {
		t.Fatal("expected error promoting a non-DONE VERIFY node")
	}
}

func TestBlackboard_CloneIsolatesMutation(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "read_A", Status: NodeStatusDone, Outputs: map[string]float64{"A": 95}})

	clone := bb.Clone()
	clone.Nodes["read_A"].Outputs["A"] = 999
	clone.Nodes["read_A"].Status = NodeStatusFailedContract

	if bb.Nodes["read_A"].Outputs["A"] != 95 {
		t.Fatalf("original mutated: Outputs[A] = %v, want 95", bb.Nodes["read_A"].Outputs["A"])
	}
	if bb.Nodes["read_A"].Status != NodeStatusDone {
		t.Fatalf("original mutated: Status = %s, want DONE", bb.Nodes["read_A"].Status)
	}
}

func TestBlackboard_CloneDeepCopiesDependsOnSlice(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "norm_A", DependsOn: []string{"read_A"}, Status: NodeStatusDone})
	clone := bb.Clone()
	clone.Nodes["norm_A"].DependsOn[0] = "mutated"
	if bb.Nodes["norm_A"].DependsOn[0] != "read_A" {
		t.Fatalf("original DependsOn mutated via clone: %v", bb.Nodes["norm_A"].DependsOn)
	}
}

func TestBlackboard_OrderedNodeIDsIsInsertionOrder(t *testing.T) {
	bb := NewBlackboard("wf-1")
	_ = bb.Record(NodeRecord{NodeID: "c", Status: NodeStatusDone})
	_ = bb.Record(NodeRecord{NodeID: "a", Status: NodeStatusDone})
	_ = bb.Record(NodeRecord{NodeID: "b", Status: NodeStatusDone})
	got := bb.OrderedNodeIDs()
	want := []string{"c", "a", "b"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
