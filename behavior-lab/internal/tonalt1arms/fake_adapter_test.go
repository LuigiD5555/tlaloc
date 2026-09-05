package tonalt1arms

import (
	"context"
	"fmt"
	"sync"
)

// fakeParrotAdapter is the offline-only, zero-network ParrotAdapter used by
// every executor test in this package. It counts total calls and calls per
// capability, and can be configured to return a specific/deterministic
// value per (capability, operand image) or fail a specific capability
// entirely -- enough to exercise both the all-success path and the
// fail-closed/BLOCKED_BY_DEPENDENCY propagation path without ever touching
// a real network.
type fakeParrotAdapter struct {
	mu                sync.Mutex
	calls             int
	callsByCapability map[string]int

	// answerForImage, keyed by capability+"|"+string(image bytes), lets a
	// test control exactly what EXTRACT_NUMBER returns per operand image.
	answerForImage map[string]float64
	// defaultAnswer is returned when answerForImage has no entry and the
	// capability is not forced to fail.
	defaultAnswer float64
	// failCapability, if set, makes every call for that capability return a
	// transport error -- used to exercise BLOCKED_BY_DEPENDENCY.
	failCapability string
}

func newFakeParrotAdapter() *fakeParrotAdapter {
	return &fakeParrotAdapter{
		callsByCapability: make(map[string]int),
		answerForImage:    make(map[string]float64),
	}
}

func (f *fakeParrotAdapter) Call(ctx context.Context, req ParrotRequest) (ParrotResponse, error) {
	f.mu.Lock()
	f.calls++
	f.callsByCapability[req.Capability]++
	f.mu.Unlock()

	if f.failCapability != "" && req.Capability == f.failCapability {
		return ParrotResponse{}, fmt.Errorf("fake transport failure for %s", req.Capability)
	}

	value := f.defaultAnswer
	if len(req.Image) > 0 {
		key := req.Capability + "|" + string(req.Image)
		if v, ok := f.answerForImage[key]; ok {
			value = v
		}
	}
	return ParrotResponse{
		RawOutput:   fmt.Sprintf("%v", value),
		ParsedValue: value,
		ParsedOK:    true,
		TransportOK: true,
		SchemaOK:    true,
		ContractOK:  true,
	}, nil
}

func (f *fakeParrotAdapter) totalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeParrotAdapter) callsFor(capability string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callsByCapability[capability]
}
