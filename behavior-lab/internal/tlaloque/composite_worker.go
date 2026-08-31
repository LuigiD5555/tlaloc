package tlaloque

import (
	"context"
	"encoding/json"
	"fmt"
)

// CompositeWorker exposes a complete sub-swarm as one capability. This is the
// mechanism that lets Tlaloc build hierarchical skills: many dumb atomic
// workers can be reused as one larger but still bounded Tlaloque.
type CompositeWorker struct {
	Desc     CapabilityDescriptor
	Plan     SwarmPlan
	Registry *Registry
}

func (w CompositeWorker) Descriptor() CapabilityDescriptor { return w.Desc }

func (w CompositeWorker) Execute(ctx context.Context, req CapabilityRequest) (CapabilityResponse, error) {
	if w.Registry == nil {
		return CapabilityResponse{}, fmt.Errorf("composite worker %q has no registry", w.Desc.ID)
	}
	input := req.Input
	if len(req.Context) > 0 {
		wrapped, err := json.Marshal(map[string]any{
			"input":   json.RawMessage(req.Input),
			"context": req.Context,
		})
		if err != nil { return CapabilityResponse{}, err }
		input = wrapped
	}
	report, err := (SwarmRunner{Registry: w.Registry}).Run(ctx, w.Plan, req.TaskID+"/"+req.NodeID, input)
	if err != nil { return CapabilityResponse{}, err }
	body, err := json.Marshal(report.TerminalOutputs)
	if err != nil { return CapabilityResponse{}, err }
	return CapabilityResponse{WorkerID: w.Desc.ID, Output: body, Confidence: aggregateConfidence(report.Nodes)}, nil
}

func aggregateConfidence(nodes []NodeExecution) float64 {
	var sum float64
	var count int
	for _, n := range nodes {
		if n.Error == "" && n.Confidence > 0 {
			sum += n.Confidence
			count++
		}
	}
	if count == 0 { return 0 }
	return sum / float64(count)
}
