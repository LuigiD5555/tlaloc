package perceptenvelope

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// ExperimentID is the frozen experiment identifier.
const ExperimentID = "parrot-perceptual-envelope-r1"

// MaxCandidatesPerPage is the page-diversity guard fixed before allocation
// (protocol section 6.3).
const MaxCandidatesPerPage = 2

// R1ASize / R1BSize are the frozen base counts.
const (
	R1ASize = 30
	R1BSize = 30
)

// rankKey is sha256(seed || candidate_id); lexicographic ascending order is
// the deterministic partition rule (protocol section 6).
func rankKey(candidateID string) string {
	sum := sha256.Sum256(append([]byte(Seed), []byte(candidateID)...))
	return hex.EncodeToString(sum[:])
}

// Allocation is the frozen R1-A / R1-B split (R1A_BASES.json / R1B_BASES.json
// share this type; the file records which slice it holds).
type Allocation struct {
	Schema           string   `json:"schema"`
	ExperimentID     string   `json:"experiment_id"`
	Stage            string   `json:"stage"` // R1-A | R1-B
	Seed             string   `json:"seed"`
	RankRule         string   `json:"rank_rule"`
	AlgorithmVersion string   `json:"selection_algorithm_version"`
	SourcePoolSHA256 string   `json:"source_pool_sha256"`
	PageDiversity    string   `json:"page_diversity_guard"`
	BaseCount        int      `json:"base_count"`
	BaseIDs          []string `json:"base_ids"`
	Bases            []Base   `json:"bases"`
}

// Base is one allocated stimulus: a candidate plus its stage-local base id.
type Base struct {
	BaseID    string    `json:"base_id"`
	Stage     string    `json:"stage"`
	RankKey   string    `json:"rank_key"`
	Candidate Candidate `json:"candidate"`
}

const allocSchema = "tlaloc.parrot-perceptual-envelope-r1.allocation.r1"

// Allocate performs the frozen deterministic partition: rank every primary
// candidate by rankKey, then sweep in rank order applying the
// <=MaxCandidatesPerPage guard, assigning the first R1ASize eligible to
// R1-A and the next disjoint R1BSize to R1-B.
func Allocate(pool SourcePool, poolSHA256 string) (r1a, r1b Allocation) {
	type ranked struct {
		cand Candidate
		key  string
	}
	items := make([]ranked, 0, len(pool.Candidates))
	for _, c := range pool.Candidates {
		items = append(items, ranked{cand: c, key: rankKey(c.CandidateID)})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].key != items[j].key {
			return items[i].key < items[j].key
		}
		return items[i].cand.CandidateID < items[j].cand.CandidateID
	})

	perPage := map[int]int{}
	var aBases, bBases []Base
	for _, it := range items {
		if perPage[it.cand.Page] >= MaxCandidatesPerPage {
			continue
		}
		if len(aBases) < R1ASize {
			perPage[it.cand.Page]++
			aBases = append(aBases, Base{Stage: "R1-A", RankKey: it.key, Candidate: it.cand})
			continue
		}
		if len(bBases) < R1BSize {
			perPage[it.cand.Page]++
			bBases = append(bBases, Base{Stage: "R1-B", RankKey: it.key, Candidate: it.cand})
			continue
		}
		break
	}
	for i := range aBases {
		aBases[i].BaseID = baseID("r1a", i, aBases[i].Candidate.CandidateID)
	}
	for i := range bBases {
		bBases[i].BaseID = baseID("r1b", i, bBases[i].Candidate.CandidateID)
	}

	mk := func(stage string, bs []Base) Allocation {
		ids := make([]string, len(bs))
		for i, b := range bs {
			ids[i] = b.BaseID
		}
		return Allocation{
			Schema: allocSchema, ExperimentID: ExperimentID, Stage: stage,
			Seed: Seed, RankRule: "sha256(seed || candidate_id) ascending",
			AlgorithmVersion: SelectionAlgorithmVersion, SourcePoolSHA256: poolSHA256,
			PageDiversity: "at most 2 primary candidates per source page, enforced during the rank-order sweep",
			BaseCount:     len(bs), BaseIDs: ids, Bases: bs,
		}
	}
	return mk("R1-A", aBases), mk("R1-B", bBases)
}

// R1A1Size is the fresh canonical R1-A1 base count.
const R1A1Size = 30

// AllocateR1A1 selects R1-A1's fresh bases: the same frozen rank
// (sha256(seed||candidate_id)) and <=2/page rule, over the candidates
// remaining after excluding every R1-A0 and R1-B candidate_id.
func AllocateR1A1(pool SourcePool, exclude map[string]struct{}, poolSHA256 string) Allocation {
	type ranked struct {
		cand Candidate
		key  string
	}
	items := make([]ranked, 0, len(pool.Candidates))
	for _, c := range pool.Candidates {
		if _, skip := exclude[c.CandidateID]; skip {
			continue
		}
		items = append(items, ranked{cand: c, key: rankKey(c.CandidateID)})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].key != items[j].key {
			return items[i].key < items[j].key
		}
		return items[i].cand.CandidateID < items[j].cand.CandidateID
	})
	perPage := map[int]int{}
	var bs []Base
	for _, it := range items {
		if len(bs) >= R1A1Size {
			break
		}
		if perPage[it.cand.Page] >= MaxCandidatesPerPage {
			continue
		}
		perPage[it.cand.Page]++
		bs = append(bs, Base{Stage: "R1-A1", RankKey: it.key, Candidate: it.cand})
	}
	for i := range bs {
		bs[i].BaseID = baseID("r1a1", i, bs[i].Candidate.CandidateID)
	}
	ids := make([]string, len(bs))
	for i, b := range bs {
		ids[i] = b.BaseID
	}
	return Allocation{
		Schema: allocSchema, ExperimentID: ExperimentID, Stage: "R1-A1",
		Seed: Seed, RankRule: "sha256(seed || candidate_id) ascending, R1-A0 + R1-B candidate_ids excluded",
		AlgorithmVersion: SelectionAlgorithmVersion, SourcePoolSHA256: poolSHA256,
		PageDiversity: "at most 2 candidates per source page within R1-A1",
		BaseCount:     len(bs), BaseIDs: ids, Bases: bs,
	}
}

func baseID(prefix string, index int, candidateID string) string {
	return prefix + "-" + pad2(index+1) + "-" + candidateID[:8]
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
