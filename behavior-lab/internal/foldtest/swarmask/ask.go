package swarmask

import (
	"context"
	"encoding/json"
	"fmt"

	"tlaloc.local/behaviorlab/internal/blackboard"
	"tlaloc.local/behaviorlab/internal/tlaloque"
)

// Ask runs the full swarm plan (classifier/scout/entities -> answer ->
// consolidate) for a single question, sharing state through store under
// runID. It returns the consolidated answer (the loro's answer text plus a
// grounding verdict checked against the real page content) and the full
// SwarmReport (including which nodes ran and what they observed), so a
// caller can inspect exactly what was shared on the blackboard.
func Ask(ctx context.Context, store blackboard.Store, runID string, in AskInput) (ConsolidationOutput, tlaloque.SwarmReport, error) {
	inputRaw, err := json.Marshal(in)
	if err != nil {
		return ConsolidationOutput{}, tlaloque.SwarmReport{}, err
	}

	runner := tlaloque.SwarmRunner{
		Registry:   NewRegistry(in.ClassifierEndpoint),
		Blackboard: &tlaloque.BlackboardRuntime{Store: store, RunID: runID},
	}

	report, err := runner.Run(ctx, NewPlan(), runID, inputRaw)
	if err != nil {
		return ConsolidationOutput{}, report, err
	}

	terminalRaw, ok := report.TerminalOutputs["consolidate"]
	if !ok {
		return ConsolidationOutput{}, report, fmt.Errorf("swarmask: no terminal output for node %q", "consolidate")
	}
	var out ConsolidationOutput
	if err := json.Unmarshal(terminalRaw, &out); err != nil {
		return ConsolidationOutput{}, report, fmt.Errorf("swarmask: decoding consolidation output: %w", err)
	}
	return out, report, nil
}
