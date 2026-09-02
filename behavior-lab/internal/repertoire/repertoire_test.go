package repertoire

import (
	"testing"
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

func at(minute int) time.Time {
	return time.Date(2026, 9, 2, 12, minute, 0, 0, time.UTC)
}

func admissibleProfile(workerID string) calibration.CalibrationProfile {
	return calibration.CalibrationProfile{
		Schema:            calibration.Schema,
		WorkerID:          workerID,
		ConfidenceFloor:   0.7,
		InDistribution:    calibration.EvalSlice{N: 500, Accuracy: 0.99, ECE: 0.03},
		OutOfDistribution: calibration.EvalSlice{N: 200, Accuracy: 0.86, ECE: 0.08},
		AbstentionCurve: []calibration.AbstentionPoint{
			{Threshold: 0.5, Coverage: 1, CoveredAccuracy: 0.8},
			{Threshold: 0.7, Coverage: 0.8, CoveredAccuracy: 0.9},
			{Threshold: 0.9, Coverage: 0.5, CoveredAccuracy: 0.97},
		},
	}
}

// Positive case: a specialist walks the full pipeline and its history is
// preserved intact at every step.
func TestLifecycle_FullPathPreservesHistory(t *testing.T) {
	entry := New("scout-r1", "r1", "first build", at(0))

	steps := []struct {
		to     Phase
		reason string
	}{
		{Candidate, "passed smoke tests"},
		{Shadow, "shadowing the rule-based scout"},
	}
	var err error
	for index, step := range steps {
		entry, err = entry.Transition(step.to, step.reason, "", at(index+1))
		if err != nil {
			t.Fatalf("transition to %s: %v", step.to, err)
		}
	}

	entry, err = PromoteToActive(entry, admissibleProfile("scout-r1"),
		Assessment{WorkerID: "scout-r1", ProposedPhase: Active, Reasons: []string{"beats baseline on 3 weeks of shadow traffic"}}, at(5))
	if err != nil {
		t.Fatalf("PromoteToActive: %v", err)
	}

	entry, err = entry.Transition(Degraded, "regression after registry change", "run-991", at(9))
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	entry, err = entry.Transition(Retired, "replaced by scout-r2", "", at(10))
	if err != nil {
		t.Fatalf("retire: %v", err)
	}
	entry, err = entry.Transition(Archived, "cold storage", "", at(11))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}

	if entry.Phase != Archived {
		t.Fatalf("final phase %s, want ARCHIVED", entry.Phase)
	}
	// genesis + 2 + promote + degrade + retire + archive = 7 events, none lost.
	if len(entry.History) != 7 {
		t.Fatalf("history length %d, want 7: %+v", len(entry.History), entry.History)
	}
	if entry.History[0].To != Experimental || entry.History[3].To != Active {
		t.Errorf("history order corrupted: %+v", entry.History)
	}
}

// Transitions not in the state machine are rejected, and a rejected
// transition does not mutate the entry.
func TestTransition_RejectsIllegalMovesWithoutMutating(t *testing.T) {
	entry := New("w", "r1", "build", at(0))

	if _, err := entry.Transition(Active, "skip the line", "", at(1)); err == nil {
		t.Error("EXPERIMENTAL -> ACTIVE must be rejected")
	}
	if _, err := entry.Transition(Candidate, "", "", at(1)); err == nil {
		t.Error("a transition with no reason must be rejected")
	}
	if entry.Phase != Experimental || len(entry.History) != 1 {
		t.Errorf("rejected transitions mutated the entry: %+v", entry)
	}

	// History aliasing cannot corrupt an earlier snapshot.
	promoted, err := entry.Transition(Candidate, "ok", "", at(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.History) != 1 {
		t.Error("Transition mutated the source entry's history")
	}
	_ = promoted
}

// Interaction with the promotion gate: SHADOW + admissible profile +
// assessment proposing ACTIVE is required. Missing any one blocks it.
func TestPromoteToActive_GateInteraction(t *testing.T) {
	shadow, _ := New("m", "r1", "b", at(0)).Transition(Candidate, "c", "", at(1))
	shadow, _ = shadow.Transition(Shadow, "s", "", at(2))

	good := Assessment{WorkerID: "m", ProposedPhase: Active, Reasons: []string{"cleared"}}

	// Not in SHADOW.
	if _, err := PromoteToActive(New("m", "r1", "b", at(0)), admissibleProfile("m"), good, at(3)); err == nil {
		t.Error("must refuse a non-SHADOW entry")
	}
	// Profile not admissible (no OOD measurement — the questionclass shape).
	unmeasured := admissibleProfile("m")
	unmeasured.OutOfDistribution = calibration.EvalSlice{}
	if _, err := PromoteToActive(shadow, unmeasured, good, at(3)); err == nil {
		t.Error("must refuse when the calibration gate refuses")
	}
	// Assessment does not propose ACTIVE.
	if _, err := PromoteToActive(shadow, admissibleProfile("m"), Assessment{WorkerID: "m", ProposedPhase: Candidate}, at(3)); err == nil {
		t.Error("must refuse when the assessment does not propose ACTIVE")
	}
	// All three satisfied.
	promoted, err := PromoteToActive(shadow, admissibleProfile("m"), good, at(3))
	if err != nil {
		t.Fatalf("valid promotion refused: %v", err)
	}
	if promoted.Phase != Active {
		t.Errorf("phase %s, want ACTIVE", promoted.Phase)
	}
}

// Non-applicable case: Monitor does nothing for a specialist that is not
// ACTIVE/DEGRADED, and false-positive boundary: a single failure with a
// matching recovery is not enough to degrade.
func TestMonitor_NonApplicableAndBoundary(t *testing.T) {
	candidate := New("m", "r1", "b", at(0))
	candidate, _ = candidate.Transition(Candidate, "c", "", at(1))
	if _, _, act := Monitor(candidate, []Experience{{Failures: 10}}, at(30), time.Hour); act {
		t.Error("Monitor must not act on a CANDIDATE")
	}

	active, _ := New("m", "r1", "b", at(0)).Transition(Candidate, "c", "", at(1))
	active, _ = active.Transition(Shadow, "s", "", at(2))
	active, _ = PromoteToActive(active, admissibleProfile("m"), Assessment{WorkerID: "m", ProposedPhase: Active, Reasons: []string{"ok"}}, at(3))

	balanced := make([]Experience, 6)
	for index := range balanced {
		balanced[index] = Experience{Failures: 1, Recoveries: 1}
	}
	if _, _, act := Monitor(active, balanced, at(40), time.Hour); act {
		t.Error("failures matched by recoveries must not trigger DEGRADED")
	}

	// One confirmed false action retires immediately.
	phase, _, act := Monitor(active, []Experience{{FalseActions: 1}}, at(40), time.Hour)
	if !act || phase != Retired {
		t.Errorf("a confirmed false action must retire: act=%v phase=%s", act, phase)
	}
}
