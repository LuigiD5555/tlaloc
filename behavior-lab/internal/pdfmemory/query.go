package pdfmemory

import (
	"fmt"
	"sort"
)

func Query(storeDir string, m Manifest, idx Index, q string, maxTokens, limit int) (Packet, error) {
	if maxTokens <= 0 {
		maxTokens = DefaultBudget
	}
	if limit <= 0 {
		limit = 8
	}
	p := Packet{Schema: ToolProtocol + ".context-packet", Query: q, Budget: Budget{MaxTokens: maxTokens, RemainingTokens: maxTokens}}
	terms := tokenize(q)
	p.Metrics.IndexTerms = len(terms)
	p.Attention = append(p.Attention, AttentionStep{Level: "H0", Action: "intent_terms", InputCount: 1, OutputCount: len(terms)})
	if len(terms) == 0 {
		p.Unknown = true
		p.Reason = "UNKNOWN"
		p.Uncertainty = 1
		p.StopReason = "no_query_terms"
		return p, nil
	}
	if st, err := LoadCanonicalState(storeDir, m); err == nil {
		p.CanonicalClaims = matchCanonicalClaims(st, terms, 8)
	}

	// H1 routes into document scopes before deeper graph work. This stays metadata-only.
	docSet := map[string]struct{}{}
	matchedTerms := 0
	for _, t := range terms {
		rows := idx.Postings[t]
		if len(rows) > 0 {
			matchedTerms++
		}
		for _, bi := range rows {
			if bi >= 0 && bi < len(m.Blocks) {
				docSet[m.Blocks[bi].DocID] = struct{}{}
			}
		}
	}
	docs := make([]string, 0, len(docSet))
	for d := range docSet {
		docs = append(docs, d)
	}
	sort.Strings(docs)
	coverage := float64(matchedTerms) / float64(len(terms))
	unc := 1 - coverage
	p.Attention = append(p.Attention, AttentionStep{Level: "H1", Action: "document_route", InputCount: len(m.Documents), OutputCount: len(docs), Uncertainty: unc, Addresses: docs})

	g, err := LoadGraph(storeDir)
	if err != nil {
		return p, err
	}
	depth := 1
	if unc > .25 {
		depth = 2
	}
	if unc > .60 {
		depth = 3
	}
	expanded := graphExpandTerms(g, terms, depth)
	p.Metrics.GraphTermsExpanded = len(expanded) - len(terms)
	p.Attention = append(p.Attention, AttentionStep{Level: "H2", Action: fmt.Sprintf("graph_expand_depth_%d", depth), InputCount: len(terms), OutputCount: len(expanded), Uncertainty: unc})

	hits := rankHits(m, idx, terms, expanded, limit*4)
	p.Metrics.CandidateObjects = len(hits)
	addrs := make([]string, 0, minInt(len(hits), 8))
	for i, h := range hits {
		if i >= 8 {
			break
		}
		addrs = append(addrs, h.Address)
	}
	if len(hits) == 0 {
		p.Unknown = true
		p.Reason = "UNKNOWN"
		p.Uncertainty = 1
		p.StopReason = "no_candidate_objects"
		p.Attention = append(p.Attention, AttentionStep{Level: "H3", Action: "rank_blocks", OutputCount: 0, Uncertainty: 1})
		return p, nil
	}
	// Candidate dispersion contributes to uncertainty: many similarly plausible objects require more evidence.
	candidateUnc := unc
	if len(hits) > limit*2 {
		candidateUnc += .15
	}
	if candidateUnc > 1 {
		candidateUnc = 1
	}
	p.Attention = append(p.Attention, AttentionStep{Level: "H3", Action: "rank_blocks", InputCount: len(m.Blocks), OutputCount: len(hits), Uncertainty: candidateUnc, Addresses: addrs})

	merkle, err := LoadMerkle(storeDir)
	if err != nil {
		return p, err
	}
	seenCID := map[string]struct{}{}
	seenAddr := map[string]struct{}{}
	for _, h := range hits {
		if len(p.Evidence) >= limit {
			break
		}
		br, b, err := ReadBlock(storeDir, m, h.Address)
		if err != nil {
			return p, err
		}
		p.Metrics.ObjectsOpened++
		p.Metrics.ExactBytesRead += len(b)
		if _, ok := seenCID[br.CID]; ok {
			continue
		}
		if _, ok := seenAddr[br.Address]; ok {
			continue
		}
		c := estimateTokens(b)
		if c > p.Budget.RemainingTokens {
			p.ExpandableRefs = append(p.ExpandableRefs, br.Address)
			continue
		}
		ev, err := evidenceForObject(merkle, "block", br.Address, br.CID, b, c, "evidence", br.DocID, br.Page, br.Number)
		if err != nil {
			return p, err
		}
		p.Evidence = append(p.Evidence, ev)
		p.Budget.UsedTokens += c
		p.Budget.RemainingTokens -= c
		p.Metrics.TokensExposed += c
		seenCID[br.CID] = struct{}{}
		seenAddr[br.Address] = struct{}{}
	}
	p.Attention = append(p.Attention, AttentionStep{Level: "H4", Action: "selective_unfold", InputCount: len(hits), OutputCount: len(p.Evidence), Uncertainty: candidateUnc})
	verified := 0
	for _, e := range p.Evidence {
		if e.Verified {
			verified++
		}
	}
	finalUnc := candidateUnc
	if len(p.Evidence) > 0 && verified == len(p.Evidence) {
		finalUnc *= .35
	}
	if finalUnc < 0 {
		finalUnc = 0
	}
	p.Uncertainty = finalUnc
	p.Attention = append(p.Attention, AttentionStep{Level: "H5", Action: "exact_evidence_verify", InputCount: len(p.Evidence), OutputCount: verified, Uncertainty: finalUnc})
	for _, t := range terms {
		if node, ok := g.Nodes[t]; ok {
			for i, e := range node.Neighbors {
				if i >= 3 {
					break
				}
				p.Relations = append(p.Relations, Relation{From: "concept:" + t, Relation: "co_occurs", To: "concept:" + e.Term, Weight: e.Weight})
			}
		}
	}
	if len(p.Evidence) == 0 {
		p.Unknown = true
		if len(p.ExpandableRefs) > 0 {
			p.Reason = "relevant objects found but none fit context budget"
			p.StopReason = "budget"
		} else {
			p.Reason = "UNKNOWN"
			p.StopReason = "no_evidence"
		}
	} else {
		p.StopReason = "sufficient_verified_evidence"
	}
	sort.Strings(p.ExpandableRefs)
	return p, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func rankHits(m Manifest, idx Index, original, expanded []string, limit int) []BlockRef {
	if limit <= 0 {
		limit = 32
	}
	weight := map[string]int{}
	for _, t := range expanded {
		weight[t] = 1
	}
	for _, t := range original {
		weight[t] = 3
	}
	scores := map[int]int{}
	for term, w := range weight {
		for _, bi := range idx.Postings[term] {
			if bi >= 0 && bi < len(m.Blocks) {
				scores[bi] += w
			}
		}
	}
	type row struct{ i, s int }
	rows := make([]row, 0, len(scores))
	for i, s := range scores {
		rows = append(rows, row{i, s})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].s == rows[j].s {
			return m.Blocks[rows[i].i].Address < m.Blocks[rows[j].i].Address
		}
		return rows[i].s > rows[j].s
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]BlockRef, len(rows))
	for i, r := range rows {
		out[i] = m.Blocks[r.i]
	}
	return out
}
