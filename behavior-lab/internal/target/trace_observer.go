package target

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	TraceRequestStart = "request.start"
	TraceFirstDelta   = "response.first_delta"
	TraceDelta        = "response.delta"
	TraceRequestDone  = "request.done"
	TraceRequestError = "request.error"
)

type ModelTraceEvent struct {
	Type       string
	Model      string
	Question   string
	Delta      string
	Elapsed    time.Duration
	Characters int
	Err        error
}

type ModelTraceObserver interface {
	Observe(ModelTraceEvent)
}

// WriterTraceObserver renders live model output without contaminating the
// machine-readable stdout used by Tlaloc commands. Callers normally pass
// os.Stderr as the writer.
type WriterTraceObserver struct {
	w  io.Writer
	mu sync.Mutex
}

func NewWriterTraceObserver(w io.Writer) *WriterTraceObserver {
	return &WriterTraceObserver{w: w}
}

func (o *WriterTraceObserver) Observe(event ModelTraceEvent) {
	if o == nil || o.w == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	switch event.Type {
	case TraceRequestStart:
		q := strings.ReplaceAll(strings.TrimSpace(event.Question), "\n", " ")
		if len(q) > 160 {
			q = q[:157] + "..."
		}
		fmt.Fprintf(o.w, "\n[trace] request.start model=%s question=%q\n[trace] response.stream\n", event.Model, q)
	case TraceFirstDelta:
		fmt.Fprintf(o.w, "[trace] first-delta after=%s\n", event.Elapsed.Round(time.Millisecond))
	case TraceDelta:
		fmt.Fprint(o.w, event.Delta)
	case TraceRequestDone:
		fmt.Fprintf(o.w, "\n[trace] request.done model=%s elapsed=%s chars=%d\n", event.Model, event.Elapsed.Round(time.Millisecond), event.Characters)
	case TraceRequestError:
		fmt.Fprintf(o.w, "\n[trace] request.error model=%s elapsed=%s error=%v\n", event.Model, event.Elapsed.Round(time.Millisecond), event.Err)
	}
}
