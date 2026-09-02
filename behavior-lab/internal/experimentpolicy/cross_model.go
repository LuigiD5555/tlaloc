package experimentpolicy

import (
	"sort"
	"strings"
)

const CrossModelCompatibilitySchemaR1 = "tlaloc.cross-model-compatibility-report.r1"

const (
	ModelCompatibilityPreservePass  = "PRESERVE_PASS"
	ModelCompatibilityImproveToPass = "IMPROVE_TO_PASS"
)

type ModelCompatibilityRequirement struct {
	ModelID               string   `json:"model_id"`
	Mode                  string   `json:"mode"`
	BaselinePass          bool     `json:"baseline_pass"`
	RequiredCandidatePass bool     `json:"required_candidate_pass"`
	EvidenceIDs           []string `json:"evidence_ids,omitempty"`
}

type ModelTrialOutcome struct {
	ModelID string `json:"model_id"`
	Pass    bool   `json:"pass"`
}

type ModelCompatibilityCheck struct {
	ModelID      string `json:"model_id"`
	Mode         string `json:"mode"`
	BaselinePass bool   `json:"baseline_pass"`
	Trials       int    `json:"trials"`
	PassedTrials int    `json:"passed_trials"`
	Pass         bool   `json:"pass"`
	Reason       string `json:"reason,omitempty"`
}

type CrossModelCompatibilityReport struct {
	Schema      string                    `json:"schema"`
	CandidateID string                    `json:"candidate_id"`
	Pass        bool                      `json:"pass"`
	Checks      []ModelCompatibilityCheck `json:"checks"`
	FailureCode string                    `json:"failure_code,omitempty"`
}

// CheckCrossModelCompatibility is the promotion-time gate for candidates that
// carry a model panel. A passing baseline model must stay passing, while a model
// marked IMPROVE_TO_PASS must pass the candidate. Every recorded trial for a
// required-pass model must pass; a single regression rejects promotion.
func CheckCrossModelCompatibility(candidate CandidateManifest, outcomes []ModelTrialOutcome) CrossModelCompatibilityReport {
	r := CrossModelCompatibilityReport{Schema: CrossModelCompatibilitySchemaR1, CandidateID: candidate.ID, Pass: true}
	if len(candidate.CompatibilityPanel) == 0 {
		r.Pass = false
		r.FailureCode = "MODEL_PANEL_REQUIRED"
		return r
	}

	byModel := map[string][]bool{}
	for _, o := range outcomes {
		id := strings.TrimSpace(o.ModelID)
		if id == "" {
			continue
		}
		byModel[id] = append(byModel[id], o.Pass)
	}

	panel := append([]ModelCompatibilityRequirement(nil), candidate.CompatibilityPanel...)
	sort.Slice(panel, func(i, j int) bool { return panel[i].ModelID < panel[j].ModelID })
	seen := map[string]bool{}
	for _, req := range panel {
		id := strings.TrimSpace(req.ModelID)
		check := ModelCompatibilityCheck{ModelID: id, Mode: req.Mode, BaselinePass: req.BaselinePass}
		if id == "" || seen[id] {
			check.Pass = false
			check.Reason = "panel model ids must be non-empty and unique"
			r.Pass = false
			r.Checks = append(r.Checks, check)
			continue
		}
		seen[id] = true
		mode := strings.ToUpper(strings.TrimSpace(req.Mode))
		if mode == ModelCompatibilityPreservePass && !req.BaselinePass {
			check.Pass = false
			check.Reason = "PRESERVE_PASS requires a passing baseline"
			r.Pass = false
			r.Checks = append(r.Checks, check)
			continue
		}
		if mode == ModelCompatibilityImproveToPass && req.BaselinePass {
			check.Pass = false
			check.Reason = "IMPROVE_TO_PASS requires a failing baseline"
			r.Pass = false
			r.Checks = append(r.Checks, check)
			continue
		}
		if mode != ModelCompatibilityPreservePass && mode != ModelCompatibilityImproveToPass {
			check.Pass = false
			check.Reason = "unknown compatibility mode"
			r.Pass = false
			r.Checks = append(r.Checks, check)
			continue
		}

		trials := byModel[id]
		check.Trials = len(trials)
		for _, pass := range trials {
			if pass {
				check.PassedTrials++
			}
		}
		if len(trials) == 0 {
			check.Pass = false
			check.Reason = "candidate has no real-model outcome for required panel member"
			r.Pass = false
		} else if req.RequiredCandidatePass {
			check.Pass = check.PassedTrials == check.Trials
			if !check.Pass {
				check.Reason = "all candidate trials must pass for this model"
				r.Pass = false
			}
		} else {
			check.Pass = check.PassedTrials == 0
			if !check.Pass {
				check.Reason = "candidate outcome differs from required panel state"
				r.Pass = false
			}
		}
		r.Checks = append(r.Checks, check)
	}
	if !r.Pass && r.FailureCode == "" {
		r.FailureCode = "CROSS_MODEL_COMPATIBILITY_FAILED"
	}
	return r
}
