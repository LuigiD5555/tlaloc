package swarmbench

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const RunSchemaR0 = "tlaloc.swarm-bench-run.r0"

// Run pairs one Score with the Topology that produced it. Score and Topology
// are computed independently, but the cost-model fit in Phase 3 needs both
// values from the same execution held together, not reassembled after the
// fact from two separate result files.
type Run struct {
	Schema     string         `json:"schema"`
	RunID      string         `json:"run_id"`
	DatasetID  string         `json:"dataset_id"`
	PlanID     string         `json:"plan_id"`
	Topology   Topology       `json:"topology"`
	Score      Score          `json:"score"`
	SwarmError string         `json:"swarm_error,omitempty"`
	NodeErrors map[string]int `json:"node_errors,omitempty"`
}

// swarmFields decodes the JSON a router Tlaloque returns into the Fields the
// scorer compares against ground truth. A field absent from the swarm's
// output decodes to its zero value, which ScoreItem already treats as
// incorrect rather than as missing.
func swarmFields(output json.RawMessage) Fields {
	var fields Fields
	if len(output) == 0 {
		return fields
	}
	_ = json.Unmarshal(output, &fields)
	return fields
}

// Execute runs one SwarmPlan against every item in dataset and returns the
// paired Score/Topology Run. terminalNodeID names the DAG's sink node — the
// one whose output carries the recovered Fields for scoring.
//
// A node failure on one item does not abort the sweep: that item scores as a
// total miss (ScoreItem against the zero Fields) and the failure is recorded
// in NodeErrors, so one bad document cannot silently vanish from the dataset
// count the way an abandoned node currently vanishes from a SwarmReport (see
// the swarm_runtime.go cancellation gap tracked separately).
func Execute(ctx context.Context, registry *tlaloque.Registry, plan tlaloque.SwarmPlan, dataset Dataset, terminalNodeID string) (Run, error) {
	plan, err := plan.Normalize()
	if err != nil {
		return Run{}, fmt.Errorf("swarm-bench run: %w", err)
	}
	runner := tlaloque.SwarmRunner{Registry: registry}
	nodeErrors := map[string]int{}

	score := ScoreDataset(dataset, func(item Item) Fields {
		input, marshalErr := marshalTaskInput(item)
		if marshalErr != nil {
			return Fields{}
		}
		report, runErr := runner.Run(ctx, plan, item.ID, input)
		if runErr != nil {
			for _, node := range report.Nodes {
				if node.Error != "" {
					nodeErrors[node.NodeID]++
				}
			}
			return Fields{}
		}
		output, ok := report.TerminalOutputs[terminalNodeID]
		if !ok {
			nodeErrors[terminalNodeID+"/missing_terminal"]++
			return Fields{}
		}
		return swarmFields(output)
	})

	return Run{
		Schema:     RunSchemaR0,
		RunID:      plan.ID + "/" + dataset.ID,
		DatasetID:  dataset.ID,
		PlanID:     plan.ID,
		Topology:   AnalyzeTopology(plan),
		Score:      score,
		NodeErrors: nodeErrors,
	}, nil
}
