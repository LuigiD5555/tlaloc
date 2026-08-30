package canonicalstate

import "sort"

const ControllerVersion = "uncertainty-controller.r0"

type VerificationTask struct {
	ID            string   `json:"id"`
	ConflictID    string   `json:"conflict_id,omitempty"`
	Action        string   `json:"action"`
	Evidence      []string `json:"evidence"`
	Priority      float64  `json:"priority"`
	ContextBudget int      `json:"context_budget"`
}
type VerificationPlan struct {
	Schema            string             `json:"schema"`
	ControllerVersion string             `json:"controller_version"`
	StateHash         string             `json:"state_hash_sha256"`
	Uncertainty       float64            `json:"uncertainty"`
	Action            string             `json:"action"`
	Tasks             []VerificationTask `json:"tasks,omitempty"`
}

func BuildVerificationPlan(s State, maxBudget int) VerificationPlan {
	if maxBudget <= 0 {
		maxBudget = 4000
	}
	p := VerificationPlan{Schema: "tlaloc.verification-plan.r0", ControllerVersion: ControllerVersion, StateHash: s.StateHash, Uncertainty: s.Metrics.Uncertainty, Action: "REDUCE_COMPLETE"}
	evidenceByCandidate := map[string][]string{}
	for _, c := range s.Claims {
		for _, id := range c.CandidateIDs {
			for _, e := range c.Evidence {
				evidenceByCandidate[id] = append(evidenceByCandidate[id], e.Address)
			}
		}
	}
	for _, conf := range s.Conflicts {
		seen := map[string]struct{}{}
		var ev []string
		ids := append(append([]string{}, conf.PositiveIDs...), conf.NegativeIDs...)
		for _, id := range ids {
			for _, a := range evidenceByCandidate[id] {
				if _, ok := seen[a]; !ok {
					seen[a] = struct{}{}
					ev = append(ev, a)
				}
			}
		}
		sort.Strings(ev)
		p.Tasks = append(p.Tasks, VerificationTask{ID: "verify:" + conf.ID, ConflictID: conf.ID, Action: "REOPEN_EXACT_EVIDENCE_AND_PROPOSE_VERDICT", Evidence: ev, Priority: 1, ContextBudget: maxBudget})
	}
	if len(p.Tasks) > 0 {
		p.Action = "VERIFY_CONFLICTS"
	} else if s.Metrics.Uncertainty > .35 {
		p.Action = "EXPAND_EVIDENCE"
	}
	sort.Slice(p.Tasks, func(i, j int) bool { return p.Tasks[i].ID < p.Tasks[j].ID })
	return p
}
