package tlaloque

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const TransportHTTPJSON = "HTTP_JSON"

// HTTPWorker is intended for resident micro-model services. A Python service
// can load BERT/BART once, keep it in RAM/VRAM, and accept many bounded JSON
// requests without model reload on every Tlaloque invocation.
type HTTPWorker struct {
	Desc     CapabilityDescriptor
	Endpoint string
	Timeout  time.Duration
	Client   *http.Client
}

func (w HTTPWorker) Descriptor() CapabilityDescriptor { return w.Desc }

func (w HTTPWorker) Execute(ctx context.Context, req CapabilityRequest) (CapabilityResponse, error) {
	endpoint := strings.TrimSpace(w.Endpoint)
	if endpoint == "" { return CapabilityResponse{}, fmt.Errorf("http worker %q has no endpoint", w.Desc.ID) }
	body, err := json.Marshal(req)
	if err != nil { return CapabilityResponse{}, err }
	if w.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.Timeout)
		defer cancel()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil { return CapabilityResponse{}, err }
	httpReq.Header.Set("Content-Type", "application/json")
	client := w.Client
	if client == nil { client = http.DefaultClient }
	resp, err := client.Do(httpReq)
	if err != nil { return CapabilityResponse{}, fmt.Errorf("worker %s: %w", w.Desc.ID, err) }
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil { return CapabilityResponse{}, err }
	if resp.StatusCode/100 != 2 {
		return CapabilityResponse{}, fmt.Errorf("worker %s status %s: %s", w.Desc.ID, resp.Status, strings.TrimSpace(string(raw)))
	}
	var out CapabilityResponse
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil { return CapabilityResponse{}, fmt.Errorf("worker %s invalid response: %w", w.Desc.ID, err) }
	if out.WorkerID == "" { out.WorkerID = w.Desc.ID }
	if out.WorkerID != w.Desc.ID { return CapabilityResponse{}, fmt.Errorf("worker identity mismatch: got %q want %q", out.WorkerID, w.Desc.ID) }
	if len(out.Output) == 0 || !json.Valid(out.Output) { return CapabilityResponse{}, fmt.Errorf("worker %s returned invalid JSON output", w.Desc.ID) }
	return out, nil
}
