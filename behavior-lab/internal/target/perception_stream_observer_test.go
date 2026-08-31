package target

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type recordingTraceObserver struct {
	mu     sync.Mutex
	events []ModelTraceEvent
}

func (o *recordingTraceObserver) Observe(event ModelTraceEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

func (o *recordingTraceObserver) deltas() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var b strings.Builder
	for _, event := range o.events {
		if event.Type == TraceDelta {
			b.WriteString(event.Delta)
		}
	}
	return b.String()
}

func (o *recordingTraceObserver) has(kind string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, event := range o.events {
		if event.Type == kind {
			return true
		}
	}
	return false
}

func TestCompletePerceptionStreamsThroughContextObserver(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		stream, _ := body["stream"].(bool)
		if !stream {
			t.Fatalf("stream=true required when observer is attached: %#v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"SELECT 1\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" > SELECT 2\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	observer := &recordingTraceObserver{}
	ctx := WithModelTraceObserver(context.Background(), observer)
	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "fixture-vlm"}
	result, err := client.CompletePerception(ctx, PerceptionInput{
		Question:  "Execute the image.",
		Image:     []byte("png-fixture"),
		MediaType: "image/png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "SELECT 1 > SELECT 2" {
		t.Fatalf("unexpected reconstructed response %q", result.Content)
	}
	if got := observer.deltas(); got != "SELECT 1 > SELECT 2" {
		t.Fatalf("unexpected observed deltas %q", got)
	}
	for _, kind := range []string{TraceRequestStart, TraceFirstDelta, TraceRequestDone} {
		if !observer.has(kind) {
			t.Fatalf("missing trace event %s", kind)
		}
	}
}
