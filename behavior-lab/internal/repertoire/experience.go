package repertoire

import (
	"fmt"
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque/calibration"
)

// Experience is the canonical assessment input every Tlaloc learning
// subsystem should converge on (Observation -> Experience -> Assessment).
// It records what ran, in what world, at what cost, and how it turned out
// — enough to learn "for INTENT=X in ENV=Y with constraints=Z, coalition
// A+D+F works". Fields are optional; a subsystem fills what it measured.
type Experience struct {
	Schema string `json:"schema"`

	IntentClass  string `json:"intent_class,omitempty"`
	InputProfile string `json:"input_profile,omitempty"`
	Environment  string `json:"environment,omitempty"`

	PlanHash     string   `json:"plan_hash,omitempty"`
	TopologyHash string   `json:"topology_hash,omitempty"`
	Workers      []string `json:"workers,omitempty"`

	Result         string `json:"result,omitempty"`
	VerifiedResult string `json:"verified_result,omitempty"`

	LatencyMS    int64 `json:"latency_ms,omitempty"`
	TokensTotal  int   `json:"tokens_total,omitempty"`
	Failures     int   `json:"failures,omitempty"`
	Recoveries   int   `json:"recoveries,omitempty"`
	Abstained    bool  `json:"abstained,omitempty"`
	FalseActions int   `json:"false_actions,omitempty"`
	SideEffects  int   `json:"side_effects,omitempty"`

	FinalVerdict string `json:"final_verdict,omitempty"`
}

const experienceSchema = "tlaloc.repertoire-experience.r0"

// NewExperience stamps the schema.
func NewExperience() Experience { return Experience{Schema: experienceSchema} }

// Assessment is the decision an assessor reaches about a specialist from a
// body of experience — never made by the specialist itself.
type Assessment struct {
	WorkerID      string   `json:"worker_id"`
	ProposedPhase Phase    `json:"proposed_phase"`
	Reasons       []string `json:"reasons"`
	EvidenceRef   string   `json:"evidence_ref,omitempty"`
}

// PromoteToActive is the SHADOW -> ACTIVE gate. It refuses unless (1) the
// entry is actually in SHADOW, (2) an external CalibrationProfile clears
// calibration.AdmitAsActive, and (3) the assessment proposes ACTIVE. The
// specialist has no say — the profile and the assessment come from Tlaloc,
// not from the worker. This keeps promotion authority centralized.
func PromoteToActive(entry Entry, profile calibration.CalibrationProfile, assessment Assessment, at time.Time) (Entry, error) {
	if entry.Phase != Shadow {
		return Entry{}, fmt.Errorf("repertoire: only a SHADOW specialist can be promoted to ACTIVE, %s is %s", entry.WorkerID, entry.Phase)
	}
	if assessment.WorkerID != entry.WorkerID {
		return Entry{}, fmt.Errorf("repertoire: assessment is for %q, entry is %q", assessment.WorkerID, entry.WorkerID)
	}
	if assessment.ProposedPhase != Active {
		return Entry{}, fmt.Errorf("repertoire: assessment does not propose ACTIVE for %s", entry.WorkerID)
	}
	if admitted, reasons := profile.AdmitAsActive(); !admitted {
		return Entry{}, fmt.Errorf("repertoire: calibration gate refused %s: %v", entry.WorkerID, reasons)
	}
	reason := "calibration gate + assessment cleared"
	if len(assessment.Reasons) > 0 {
		reason = assessment.Reasons[0]
	}
	return entry.Transition(Active, reason, assessment.EvidenceRef, at)
}

// Monitor decides whether an ACTIVE specialist should be flagged DEGRADED
// or pulled to RETIRED, from ongoing experience. Deterministic thresholds,
// documented: any confirmed false action retires immediately; a run of
// failures with no recoveries degrades; long disuse degrades.
func Monitor(entry Entry, recent []Experience, now time.Time, maxIdle time.Duration) (proposed Phase, reason string, act bool) {
	if entry.Phase != Active && entry.Phase != Degraded {
		return "", "", false
	}

	var failures, recoveries, falseActions, runs int
	for _, experience := range recent {
		runs++
		failures += experience.Failures
		recoveries += experience.Recoveries
		falseActions += experience.FalseActions
	}

	if falseActions > 0 {
		return Retired, fmt.Sprintf("%d confirmed false action(s) in recent experience", falseActions), true
	}
	if runs >= 5 && failures > recoveries && entry.Phase == Active {
		return Degraded, fmt.Sprintf("%d failures vs %d recoveries over %d recent runs", failures, recoveries, runs), true
	}
	if runs == 0 && !entry.EnteredActiveAt().IsZero() && now.Sub(entry.EnteredActiveAt()) > maxIdle && entry.Phase == Active {
		return Degraded, fmt.Sprintf("no runs in %s since becoming ACTIVE", now.Sub(entry.EnteredActiveAt()).Round(time.Hour)), true
	}
	return "", "", false
}
