// Package curriculum is the training/evaluation ladder for Tlaloque
// specialists. A tiny model that scores 100% on a clean synthetic set has
// proven almost nothing (see questionclass-charcnn-r0: 100% in-distribution,
// 51% out-of-distribution). This package makes the harder levels explicit
// and ordered, so "competence" is measured as how far up the ladder a
// specialist still holds — training for a system that must not misfire,
// not for an NLP benchmark.
//
// The generator of concrete cases per stage is domain-specific (text,
// Origami glyphs, file intents, ...); this package owns the ladder, the
// sequencer, and the competence-frontier assessment.
package curriculum

// Stage is one rung. Higher index = harder / more hostile.
type Stage string

const (
	C0Clean               Stage = "C0_CLEAN"
	C1Variation           Stage = "C1_VARIATION"
	C2Ambiguous           Stage = "C2_AMBIGUOUS"
	C3OOD                 Stage = "C3_OOD"
	C4ConflictingEvidence Stage = "C4_CONFLICTING_EVIDENCE"
	C5WorkerFailure       Stage = "C5_WORKER_FAILURE"
	C6CommunicationDelay  Stage = "C6_COMMUNICATION_DELAY"
	C7ResourcePressure    Stage = "C7_RESOURCE_PRESSURE"
	C8CorruptedInput      Stage = "C8_CORRUPTED_INPUT"
	C9Adversarial         Stage = "C9_ADVERSARIAL"
)

// Ladder is the canonical order.
var Ladder = []Stage{
	C0Clean, C1Variation, C2Ambiguous, C3OOD, C4ConflictingEvidence,
	C5WorkerFailure, C6CommunicationDelay, C7ResourcePressure,
	C8CorruptedInput, C9Adversarial,
}

var stageIndex = func() map[Stage]int {
	index := map[Stage]int{}
	for position, stage := range Ladder {
		index[stage] = position
	}
	return index
}()

// Index is the stage's position in the ladder, or -1 if unknown.
func (stage Stage) Index() int {
	if position, ok := stageIndex[stage]; ok {
		return position
	}
	return -1
}

// Harder reports whether stage is strictly harder than other.
func (stage Stage) Harder(other Stage) bool {
	return stage.Index() > other.Index() && stage.Index() >= 0 && other.Index() >= 0
}

// Describe is a one-line explanation of what the stage tests.
func (stage Stage) Describe() string {
	switch stage {
	case C0Clean:
		return "canonical, well-formed inputs from the training distribution"
	case C1Variation:
		return "same task, natural rephrasing / formatting / casing variation"
	case C2Ambiguous:
		return "under-specified inputs a reasonable person could read two ways"
	case C3OOD:
		return "inputs outside the trained distribution — the specialist should abstain, not guess"
	case C4ConflictingEvidence:
		return "signals that disagree (path says PDF, magic bytes say ZIP)"
	case C5WorkerFailure:
		return "a dependency worker is down or errors"
	case C6CommunicationDelay:
		return "a dependency responds after the timeout"
	case C7ResourcePressure:
		return "insufficient RAM/VRAM to run the preferred path"
	case C8CorruptedInput:
		return "malformed / truncated / invalid inputs"
	case C9Adversarial:
		return "inputs crafted to induce a wrong or unsafe action (prompt injection, poisoned content)"
	default:
		return "unknown stage"
	}
}

// IsRobustnessStage reports whether the stage tests system-level robustness
// (worker failure, delay, resource pressure) rather than input difficulty.
// A pure classifier is only responsible up to C4; C5-C7 are the swarm's
// job, C8-C9 are shared.
func (stage Stage) IsRobustnessStage() bool {
	switch stage {
	case C5WorkerFailure, C6CommunicationDelay, C7ResourcePressure:
		return true
	default:
		return false
	}
}
