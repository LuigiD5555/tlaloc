package canonicalstate

import "time"

const (
	CandidateSchema = "tlaloc.candidate.r0"
	StateSchema     = "tlaloc.canonical-state.r0"
	LedgerSchema    = "tlaloc.state-ledger.r0"
)

type Claim struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Polarity  bool   `json:"polarity"`
}

type EvidenceRef struct {
	Address string `json:"address"`
	CID     string `json:"cid_sha256,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

type Candidate struct {
	Schema     string        `json:"schema"`
	ID         string        `json:"id"`
	Role       string        `json:"role"`
	Claim      Claim         `json:"claim"`
	Evidence   []EvidenceRef `json:"evidence"`
	Confidence float64       `json:"confidence"`
	Producer   string        `json:"producer,omitempty"`
	RunID      string        `json:"run_id,omitempty"`
}

type CandidateDecision struct {
	CandidateID string   `json:"candidate_id"`
	Status      string   `json:"status"` // ACCEPTED|MERGED|CONFLICT|REJECTED|UNRESOLVED
	Reasons     []string `json:"reasons,omitempty"`
}

type CanonicalClaim struct {
	ID           string        `json:"id"`
	Claim        Claim         `json:"claim"`
	Status       string        `json:"status"` // VERIFIED|UNRESOLVED
	Evidence     []EvidenceRef `json:"evidence"`
	CandidateIDs []string      `json:"candidate_ids"`
	ConflictIDs  []string      `json:"conflict_ids,omitempty"`
	Hash         string        `json:"hash_sha256"`
}

type Conflict struct {
	ID          string   `json:"id"`
	ClaimKey    string   `json:"claim_key"`
	PositiveIDs []string `json:"positive_candidate_ids,omitempty"`
	NegativeIDs []string `json:"negative_candidate_ids,omitempty"`
	Status      string   `json:"status"` // RESOLVED|UNRESOLVED
}

type Metrics struct {
	CandidateCount  int     `json:"candidate_count"`
	Accepted        int     `json:"accepted"`
	Rejected        int     `json:"rejected"`
	Conflicts       int     `json:"conflicts"`
	Unresolved      int     `json:"unresolved"`
	EvidenceClosure float64 `json:"evidence_closure"`
	Uncertainty     float64 `json:"uncertainty"`
}

type State struct {
	Schema         string              `json:"schema"`
	ReducerVersion string              `json:"reducer_version"`
	InputHash      string              `json:"input_hash_sha256"`
	StateHash      string              `json:"state_hash_sha256"`
	Claims         []CanonicalClaim    `json:"claims"`
	Conflicts      []Conflict          `json:"conflicts,omitempty"`
	Decisions      []CandidateDecision `json:"decisions"`
	Metrics        Metrics             `json:"metrics"`
}

type Transition struct {
	Index        int       `json:"index"`
	Operation    string    `json:"operation"`
	Actor        string    `json:"actor"`
	InputHash    string    `json:"input_hash_sha256"`
	OutputHash   string    `json:"output_hash_sha256"`
	Evidence     []string  `json:"evidence,omitempty"`
	TimestampUTC time.Time `json:"timestamp_utc"`
}

type Ledger struct {
	Schema      string       `json:"schema"`
	Transitions []Transition `json:"transitions"`
	HeadHash    string       `json:"head_hash_sha256"`
}

type EvidenceVerifier interface {
	Verify(address, cid string) (bool, error)
}
