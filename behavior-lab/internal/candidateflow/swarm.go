package candidateflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/canonicalstate"
)

const Schema = "tlaloc.compilation-swarm.r0"

var Roles = []string{"perception", "structure", "semantic", "relation", "retrieval", "evidence", "conflict", "verifier"}

type RoleContract struct {
	Role          string   `json:"role"`
	Input         []string `json:"input"`
	AllowedOutput string   `json:"allowed_output"`
	MaxContext    int      `json:"max_context_token_eq"`
	Authority     string   `json:"authority"`
}

type Trace struct {
	Schema     string                     `json:"schema"`
	Contracts  []RoleContract             `json:"contracts"`
	Candidates []canonicalstate.Candidate `json:"candidates"`
	Notes      []string                   `json:"notes,omitempty"`
}

func Contracts() []RoleContract {
	out := make([]RoleContract, 0, len(Roles))
	for _, r := range Roles {
		out = append(out, RoleContract{Role: r, Input: []string{"bounded task", "reopenable evidence addresses"}, AllowedOutput: canonicalstate.CandidateSchema, MaxContext: 4000, Authority: "PROPOSAL_ONLY"})
	}
	return out
}

// ProposeFromCanonical is the deterministic reference compiler. Production LLM
// Tlaloques may propose richer candidates through the same schema, but neither
// path is allowed to mutate CanonicalState directly.
func ProposeFromCanonical(doc canonicaldoc.Document, loadPage func(int) (canonicaldoc.Page, error)) (Trace, error) {
	tr := Trace{Schema: Schema, Contracts: Contracts(), Notes: []string{"reference proposals are deterministic; model Tlaloques plug into the same candidate schema"}}
	var headings []canonicaldoc.Region
	for n := 1; n <= doc.PageCount; n++ {
		p, err := loadPage(n)
		if err != nil {
			return tr, err
		}
		for _, r := range p.Regions {
			if r.Kind != "heading" && r.Kind != "subheading" {
				continue
			}
			text := strings.TrimSpace(r.Text)
			if text == "" {
				continue
			}
			c := NewCandidate("structure", doc.DocumentID, "contains_heading", text, true, []canonicalstate.EvidenceRef{{Address: p.Address, Kind: "page"}}, .99, "structure-reference")
			tr.Candidates = append(tr.Candidates, c)
			headings = append(headings, r)
		}
	}
	for i := 0; i+1 < len(headings); i++ {
		a, b := headings[i], headings[i+1]
		c := NewCandidate("relation", a.Text, "precedes_heading", b.Text, true, []canonicalstate.EvidenceRef{{Address: pageFromRegion(a.Address), Kind: "page"}, {Address: pageFromRegion(b.Address), Kind: "page"}}, .95, "relation-reference")
		tr.Candidates = append(tr.Candidates, c)
	}
	sort.Slice(tr.Candidates, func(i, j int) bool { return tr.Candidates[i].ID < tr.Candidates[j].ID })
	return tr, nil
}

func NewCandidate(role, subject, predicate, object string, polarity bool, evidence []canonicalstate.EvidenceRef, confidence float64, producer string) canonicalstate.Candidate {
	claim := canonicalstate.Claim{Subject: subject, Predicate: predicate, Object: object, Polarity: polarity}
	refs := append([]canonicalstate.EvidenceRef(nil), evidence...)
	sort.Slice(refs, func(i, j int) bool { return refs[i].Address+refs[i].CID < refs[j].Address+refs[j].CID })
	var eb strings.Builder
	for _, e := range refs {
		eb.WriteString(e.Address)
		eb.WriteByte(0)
		eb.WriteString(e.CID)
		eb.WriteByte(0)
	}
	seed := role + "\x00" + subject + "\x00" + predicate + "\x00" + object + fmt.Sprint(polarity) + "\x00" + producer + "\x00" + eb.String()
	h := sha256.Sum256([]byte(seed))
	return canonicalstate.Candidate{Schema: canonicalstate.CandidateSchema, ID: "cand:" + hex.EncodeToString(h[:8]), Role: role, Claim: claim, Evidence: evidence, Confidence: confidence, Producer: producer}
}
func pageFromRegion(addr string) string {
	if i := strings.Index(addr, "/regions/"); i >= 0 {
		return addr[:i]
	}
	return addr
}

func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return osWrite(path, append(b, '\n'))
}
