package canonicalstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ReducerVersion = "go-reducer.r0"

type Reducer struct {
	Verifier EvidenceVerifier
	Policies *ClaimPolicyRegistry
	Fusion   map[FusionMode]FusionStrategy
}

type verifiedCandidate struct {
	C        Candidate
	Evidence []EvidenceRef
}

func (r Reducer) Reduce(candidates []Candidate) (State, error) {
	ordered := append([]Candidate(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	inputHash := hashJSON(ordered)
	state := State{Schema: StateSchema, ReducerVersion: ReducerVersion, InputHash: inputHash}
	if r.Policies != nil {
		state.PolicyVersion = r.Policies.ID
	}
	groups := map[string][]verifiedCandidate{}
	groupPolicies := map[string]ClaimPolicy{}
	// Evidence verification is deterministic for one reducer invocation. Cache by
	// address+CID so thousands of Tlaloque candidates sharing the same source do
	// not repeatedly reopen and hash identical exact objects.
	verifiedCache := map[string]bool{}
	for _, c := range ordered {
		d := CandidateDecision{CandidateID: c.ID}
		if err := validateCandidate(c); err != nil {
			d.Status = "REJECTED"
			d.Reasons = []string{err.Error()}
			state.Decisions = append(state.Decisions, d)
			state.Metrics.Rejected++
			continue
		}
		var good []EvidenceRef
		for _, e := range c.Evidence {
			cacheKey := e.Address + "\x00" + e.CID
			ok, cached := verifiedCache[cacheKey]
			if !cached {
				ok = true
				var err error
				if r.Verifier != nil {
					ok, err = r.Verifier.Verify(e.Address, e.CID)
				}
				if err != nil {
					return State{}, err
				}
				verifiedCache[cacheKey] = ok
			}
			if ok {
				good = append(good, e)
			}
		}
		if len(good) == 0 {
			d.Status = "REJECTED"
			d.Reasons = []string{"NO_VERIFIED_EVIDENCE"}
			state.Decisions = append(state.Decisions, d)
			state.Metrics.Rejected++
			continue
		}
		c.Evidence = dedupEvidence(good)
		policy := r.Policies.PolicyFor(c.Claim.Predicate)
		key := claimGroupKey(c.Claim, policy)
		groups[key] = append(groups[key], verifiedCandidate{C: c, Evidence: c.Evidence})
		groupPolicies[key] = policy
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		strategy, err := r.fusionStrategy(groupPolicies[key].Fusion)
		if err != nil {
			return State{}, err
		}
		outcome, err := strategy.Fuse(key, groups[key])
		if err != nil {
			return State{}, err
		}
		state.Claims = append(state.Claims, outcome.Claim)
		state.Decisions = append(state.Decisions, outcome.Decisions...)
		state.Metrics.Accepted += outcome.Accepted
		state.Metrics.Unresolved += outcome.Unresolved
		if outcome.Conflict != nil {
			state.Conflicts = append(state.Conflicts, *outcome.Conflict)
			state.Metrics.Conflicts++
		}
	}

	state.Metrics.CandidateCount = len(candidates)
	if len(state.Claims) > 0 {
		verified := 0
		for _, claim := range state.Claims {
			if claim.Status == "VERIFIED" && len(claim.Evidence) > 0 {
				verified++
			}
		}
		state.Metrics.EvidenceClosure = float64(verified) / float64(len(state.Claims))
	}
	state.Metrics.Uncertainty = uncertainty(state.Metrics)
	sort.Slice(state.Claims, func(i, j int) bool { return state.Claims[i].ID < state.Claims[j].ID })
	sort.Slice(state.Conflicts, func(i, j int) bool { return state.Conflicts[i].ID < state.Conflicts[j].ID })
	sort.Slice(state.Decisions, func(i, j int) bool { return state.Decisions[i].CandidateID < state.Decisions[j].CandidateID })
	state.StateHash = stateHash(state)
	return state, nil
}

func (r Reducer) fusionStrategy(mode FusionMode) (FusionStrategy, error) {
	if r.Fusion != nil {
		if strategy, ok := r.Fusion[mode]; ok && strategy != nil {
			return strategy, nil
		}
	}
	strategy, ok := defaultFusionStrategies[mode]
	if !ok || strategy == nil {
		return nil, fmt.Errorf("unsupported fusion mode %q", mode)
	}
	return strategy, nil
}

func validateCandidate(c Candidate) error {
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("MISSING_ID")
	}
	if strings.TrimSpace(c.Claim.Subject) == "" || strings.TrimSpace(c.Claim.Predicate) == "" || strings.TrimSpace(c.Claim.Object) == "" {
		return fmt.Errorf("MALFORMED_CLAIM")
	}
	if len(c.Evidence) == 0 {
		return fmt.Errorf("MISSING_EVIDENCE")
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("INVALID_CONFIDENCE")
	}
	return nil
}

func claimBaseKey(c Claim) string {
	return norm(c.Subject) + "\x00" + norm(c.Predicate) + "\x00" + norm(c.Object)
}

func norm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func canonicalFromGroup(rows []verifiedCandidate) CanonicalClaim {
	first := rows[0].C
	ev := []EvidenceRef{}
	ids := []string{}
	for _, v := range rows {
		ev = append(ev, v.Evidence...)
		ids = append(ids, v.C.ID)
	}
	ev = dedupEvidence(ev)
	sort.Strings(ids)
	claim := CanonicalClaim{ID: "claim:" + shortHash(claimBaseKey(first.Claim)+fmt.Sprint(first.Claim.Polarity)), Claim: first.Claim, Status: "VERIFIED", Evidence: ev, CandidateIDs: ids}
	claim.Hash = hashJSON(claim)
	return claim
}

func canonicalFromConflict(conf Conflict, pos, neg []verifiedCandidate) CanonicalClaim {
	rows := append(append([]verifiedCandidate{}, pos...), neg...)
	return canonicalFromConflictRows(conf, rows)
}

func canonicalFromConflictRows(conf Conflict, rows []verifiedCandidate) CanonicalClaim {
	ev := []EvidenceRef{}
	ids := []string{}
	for _, v := range rows {
		ev = append(ev, v.Evidence...)
		ids = append(ids, v.C.ID)
	}
	ev = dedupEvidence(ev)
	sort.Strings(ids)
	base := rows[0].C.Claim
	base.Polarity = true
	if len(conf.Values) > 0 {
		base.Object = ""
	}
	claim := CanonicalClaim{ID: "claim:" + shortHash(conf.ClaimKey), Claim: base, Status: "UNRESOLVED", Evidence: ev, CandidateIDs: ids, ConflictIDs: []string{conf.ID}}
	claim.Hash = hashJSON(claim)
	return claim
}

func dedupEvidence(in []EvidenceRef) []EvidenceRef {
	m := map[string]EvidenceRef{}
	for _, e := range in {
		key := e.Address + "\x00" + e.CID
		m[key] = e
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]EvidenceRef, 0, len(keys))
	for _, key := range keys {
		out = append(out, m[key])
	}
	return out
}

func uncertainty(m Metrics) float64 {
	if m.CandidateCount == 0 {
		return 1
	}
	value := (float64(m.Conflicts)*2 + float64(m.Unresolved) + float64(m.Rejected)*0.5) / float64(m.CandidateCount)
	if value > 1 {
		return 1
	}
	return value
}

func stateHash(s State) string {
	copy := s
	copy.StateHash = ""
	return hashJSON(copy)
}

func hashJSON(v any) string {
	body, _ := json.Marshal(v)
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func shortHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:8])
}
