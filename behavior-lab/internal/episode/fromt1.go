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
// T1's raw records remain authoritative. This adapter is intentionally
// loss-minimising: it preserves raw/parsed outputs, model and input identity,
// and mirrors T1 RunAccounting semantics so later prototype iterations can
// compare failures without reinterpreting what an HTTP/model call meant.
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

	var cost Cost
	failureRootCause := ""
	blockedSeen := false

	for _, rec := range matched {
		modelCalls := 0
		if rec.TransportStatus == "OK" {
			// Backwards-compatible Episode counter: completed transports.
			modelCalls = 1
		}
		cost.ModelCalls += modelCalls
		cost.LatencyMS += rec.LatencyMS

		// Mirror tonalt1arms.deriveAccounting exactly. In particular, a
		// dependency-blocked node is not an HTTP request attempt even if a
		// legacy/raw record also carries a FAILED transport status.
		switch {
		case rec.ContractStatus == "BLOCKED_BY_DEPENDENCY":
			cost.BlockedByDependency++
			blockedSeen = true
		case rec.TransportStatus == "NOT_ATTEMPTED":
			// Pre-call refusal: no HTTP attempt.
		case rec.TransportStatus == "FAILED":
			cost.HTTPRequestAttempts++
			cost.TransportFailures++
			if failureRootCause == "" {
				failureRootCause = "TRANSPORT_FAILURE"
			}
		case rec.TransportStatus == "OK" && rec.SchemaStatus == "FAILED":
			cost.HTTPRequestAttempts++
			cost.SchemaFailures++
			if failureRootCause == "" {
				failureRootCause = "SCHEMA_FAILURE"
			}
		case rec.TransportStatus == "OK" && rec.ContractStatus != "OK":
			cost.HTTPRequestAttempts++
			cost.ModelContractFailures++
			if failureRootCause == "" {
				failureRootCause = "MODEL_CONTRACT_FAILURE"
			}
		case rec.TransportStatus == "OK" && rec.ContractStatus == "OK":
			cost.HTTPRequestAttempts++
			cost.ValidCompletions++
		}

		status := rec.ContractStatus
		if status == "" {
			status = rec.SchemaStatus
		}
		if status == "" {
			status = rec.TransportStatus
		}

		steps = append(steps, Step{
			RequestIndex:       rec.RequestIndex,
			NodeID:             rec.NodeID,
			SelectedCapability: rec.Capability,
			SelectedOperation:  rec.Operation,
			ExecutorID:         rec.ExecutorID,
			Model:              rec.Model,
			InputArtifact:      rec.InputArtifact,
			InputHash:          rec.InputHash,
			RawOutput:          rec.RawOutput,
			ParsedOutput:       rec.ParsedOutput,
			TransportStatus:    rec.TransportStatus,
			SchemaStatus:       rec.SchemaStatus,
			ContractStatus:     rec.ContractStatus,
			Status:             status,
			ModelCalls:         modelCalls,
			LatencyMS:          rec.LatencyMS,
		})
	}

	// Prefer an observable execution-layer root cause over a terminal score.
	// A blocked dependency is usually a consequence, so it is used only when
	// no direct transport/schema/contract failure explains the episode.
	if failureRootCause == "" && !wf.SemanticCorrect {
		failureRootCause = "SEMANTIC_INCORRECT"
	}
	if failureRootCause == "" && !wf.ExactCorrect {
		failureRootCause = "EXACT_INCORRECT"
	}
	if failureRootCause == "" && blockedSeen {
		failureRootCause = "DEPENDENCY_BLOCKED"
	}

	return Episode{
		Schema:            Schema,
		EpisodeID:         fmt.Sprintf("t1-%s-%s-%s", wf.RunID, wf.WorkflowID, wf.Arm),
		SourceExperiment:  SourceT1,
		RunID:             wf.RunID,
		TaskID:            wf.WorkflowID + "/" + wf.Arm,
		Arm:               wf.Arm,
		Family:            wf.Family,
		Steps:             steps,
		Success:           wf.SemanticCorrect,
		SemanticCorrect:   wf.SemanticCorrect,
		ExactCorrect:      wf.ExactCorrect,
		TerminalStatus:    wf.TerminalStatus,
		ContractStatus:    wf.ContractStatus,
		FailureRootCause:  failureRootCause,
		Cost:              cost,
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
