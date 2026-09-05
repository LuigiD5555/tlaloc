package episode

import (
	"fmt"
	"sort"

	"tlaloc.local/behaviorlab/internal/tonalt1arms"
)

// SourceT1 marks an Episode as imported from the TONAL T1
// heterogeneous-composition experiment, per the corpus-status decision that
// T1 data must never be confused with native prototype-loop data.
const SourceT1 = "TONAL_T1"

// FromT1Workflow builds one Episode from a single T1 WorkflowRecord and the
// NodeCallRecords belonging to the same (WorkflowID, Arm) pair. TaskID is
// "<workflow_id>/<arm>" -- T1 has no separate goal/controller concept, so
// Goal and PrototypeVersion are left empty rather than invented.
//
// nodeRecords may contain records for other workflows/arms; only the ones
// matching wf.WorkflowID and wf.Arm are used, ordered by RequestIndex for a
// deterministic step sequence.
func FromT1Workflow(wf tonalt1arms.WorkflowRecord, nodeRecords []tonalt1arms.NodeCallRecord) Episode {
	var steps []Step
	var matched []tonalt1arms.NodeCallRecord
	for _, rec := range nodeRecords {
		if rec.WorkflowID == wf.WorkflowID && rec.Arm == wf.Arm {
			matched = append(matched, rec)
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].RequestIndex < matched[j].RequestIndex
	})

	totalModelCalls := 0
	var totalLatency int64
	for _, rec := range matched {
		modelCalls := 0
		if rec.TransportStatus == "OK" {
			modelCalls = 1
		}
		totalModelCalls += modelCalls
		totalLatency += rec.LatencyMS

		status := rec.ContractStatus
		if status == "" {
			status = rec.TransportStatus
		}

		steps = append(steps, Step{
			NodeID:             rec.NodeID,
			SelectedCapability: rec.Capability,
			SelectedOperation:  rec.Operation,
			ExecutorID:         rec.ExecutorID,
			Status:             status,
			ModelCalls:         modelCalls,
			LatencyMS:          rec.LatencyMS,
		})
	}

	success := wf.SemanticCorrect

	return Episode{
		Schema:           Schema,
		EpisodeID:        fmt.Sprintf("t1-%s-%s-%s", wf.RunID, wf.WorkflowID, wf.Arm),
		SourceExperiment: SourceT1,
		TaskID:           wf.WorkflowID + "/" + wf.Arm,
		Steps:            steps,
		Success:          success,
		Cost: Cost{
			ModelCalls: totalModelCalls,
			LatencyMS:  totalLatency,
		},
	}
}

// FromT1RunResult builds one Episode per WorkflowRecord in a complete T1
// CrossArmRunner result (180 for the frozen 60-workflow x 3-arm primary
// campaign), in the same order the WorkflowRecords were recorded.
func FromT1RunResult(result tonalt1arms.RunResult) []Episode {
	episodes := make([]Episode, 0, len(result.WorkflowRecords))
	for _, wf := range result.WorkflowRecords {
		episodes = append(episodes, FromT1Workflow(wf, result.NodeRecords))
	}
	return episodes
}
