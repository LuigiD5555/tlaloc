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

type Reducer struct{ Verifier EvidenceVerifier }

type verifiedCandidate struct {
	C        Candidate
	Evidence []EvidenceRef
}

func (r Reducer) Reduce(candidates []Candidate) (State, error) {
	ordered := append([]Candidate(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	inputHash := hashJSON(ordered)
	state := State{Schema: StateSchema, ReducerVersion: ReducerVersion, InputHash: inputHash}
	groups := map[string][]verifiedCandidate{}
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
		key := claimBaseKey(c.Claim)
		groups[key] = append(groups[key], verifiedCandidate{C: c, Evidence: c.Evidence})
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rows := groups[key]
		var pos, neg []verifiedCandidate
		for _, v := range rows {
			if v.C.Claim.Polarity {
				pos = append(pos, v)
			} else {
				neg = append(neg, v)
			}
		}
		if len(pos) > 0 && len(neg) > 0 {
			conf := Conflict{ID: "conflict:" + shortHash(key), ClaimKey: key, Status: "UNRESOLVED"}
			for _, v := range pos {
				conf.PositiveIDs = append(conf.PositiveIDs, v.C.ID)
			}
			for _, v := range neg {
				conf.NegativeIDs = append(conf.NegativeIDs, v.C.ID)
			}
			sort.Strings(conf.PositiveIDs)
			sort.Strings(conf.NegativeIDs)
			state.Conflicts = append(state.Conflicts, conf)
			state.Metrics.Conflicts++
			claim := canonicalFromConflict(conf, pos, neg)
			state.Claims = append(state.Claims, claim)
			state.Metrics.Unresolved++
			for _, v := range rows {
				state.Decisions = append(state.Decisions, CandidateDecision{CandidateID: v.C.ID, Status: "CONFLICT", Reasons: []string{conf.ID}})
			}
			continue
		}
		active := pos
		if len(active) == 0 {
			active = neg
		}
		claim := canonicalFromGroup(active)
		state.Claims = append(state.Claims, claim)
		state.Metrics.Accepted += len(active)
		for i, v := range active {
			status := "MERGED"
			if i == 0 {
				status = "ACCEPTED"
			}
			state.Decisions = append(state.Decisions, CandidateDecision{CandidateID: v.C.ID, Status: status})
		}
	}
	state.Metrics.CandidateCount = len(candidates)
	if len(state.Claims) > 0 {
		verified := 0
		for _, c := range state.Claims {
			if c.Status == "VERIFIED" && len(c.Evidence) > 0 {
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
	c := CanonicalClaim{ID: "claim:" + shortHash(claimBaseKey(first.Claim)+fmt.Sprint(first.Claim.Polarity)), Claim: first.Claim, Status: "VERIFIED", Evidence: ev, CandidateIDs: ids}
	c.Hash = hashJSON(c)
	return c
}
func canonicalFromConflict(conf Conflict, pos, neg []verifiedCandidate) CanonicalClaim {
	rows := append(append([]verifiedCandidate{}, pos...), neg...)
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
	c := CanonicalClaim{ID: "claim:" + shortHash(conf.ClaimKey), Claim: base, Status: "UNRESOLVED", Evidence: ev, CandidateIDs: ids, ConflictIDs: []string{conf.ID}}
	c.Hash = hashJSON(c)
	return c
}
func dedupEvidence(in []EvidenceRef) []EvidenceRef {
	m := map[string]EvidenceRef{}
	for _, e := range in {
		k := e.Address + "\x00" + e.CID
		m[k] = e
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]EvidenceRef, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}
func uncertainty(m Metrics) float64 {
	if m.CandidateCount == 0 {
		return 1
	}
	v := (float64(m.Conflicts)*2 + float64(m.Unresolved) + float64(m.Rejected)*0.5) / float64(m.CandidateCount)
	if v > 1 {
		return 1
	}
	return v
}
func stateHash(s State) string { copy := s; copy.StateHash = ""; return hashJSON(copy) }
func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func shortHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:8]) }
