package tlaloque

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"tlaloc.local/behaviorlab/internal/blackboard"
)

// ProcessWorker turns any local executable into a Tlaloque. The child receives
// exactly one CapabilityRequest JSON document on stdin and must return exactly
// one CapabilityResponse JSON document on stdout. This keeps Python/BERT/BART,
// Rust, Go and shell-backed specialists behind the same contract.
type ProcessWorker struct {
	Desc    CapabilityDescriptor
	Command []string
	Timeout time.Duration
}

func (w ProcessWorker) Descriptor() CapabilityDescriptor { return w.Desc }

func (w ProcessWorker) Execute(ctx context.Context, req CapabilityRequest) (CapabilityResponse, error) {
	if len(w.Command) == 0 {
		return CapabilityResponse{}, fmt.Errorf("process worker %q has no command", w.Desc.ID)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return CapabilityResponse{}, err
	}
	if w.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, w.Command[0], w.Command[1:]...)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return CapabilityResponse{}, fmt.Errorf("worker %s: %w", w.Desc.ID, ctx.Err())
		}
		return CapabilityResponse{}, fmt.Errorf("worker %s: %w: %s", w.Desc.ID, err, stderr.String())
	}
	var out CapabilityResponse
	dec := json.NewDecoder(&stdout)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return CapabilityResponse{}, fmt.Errorf("worker %s invalid response: %w", w.Desc.ID, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return CapabilityResponse{}, fmt.Errorf("worker %s returned more than one JSON document", w.Desc.ID)
		}
		return CapabilityResponse{}, fmt.Errorf("worker %s invalid trailing output: %w", w.Desc.ID, err)
	}
	if out.WorkerID == "" {
		out.WorkerID = w.Desc.ID
	}
	if out.WorkerID != w.Desc.ID {
		return CapabilityResponse{}, fmt.Errorf("worker identity mismatch: got %q want %q", out.WorkerID, w.Desc.ID)
	}
	if len(out.Output) == 0 || !json.Valid(out.Output) {
		return CapabilityResponse{}, fmt.Errorf("worker %s returned invalid JSON output", w.Desc.ID)
	}
	for i := range out.Observations {
		o, err := blackboard.NormalizeObservation(out.Observations[i])
		if err != nil {
			return CapabilityResponse{}, fmt.Errorf("worker %s invalid observation: %w", w.Desc.ID, err)
		}
		out.Observations[i] = o
	}
	return out, nil
}
