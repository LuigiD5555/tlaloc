package pdfmemory

import "tlaloc.local/behaviorlab/internal/canonicalstate"

type Budget struct {
	MaxTokens       int `json:"max_tokens"`
	UsedTokens      int `json:"used_tokens"`
	RemainingTokens int `json:"remaining_tokens"`
}
type ProofStep struct {
	Side   string `json:"side"`
	SHA256 string `json:"sha256"`
}
type Evidence struct {
	Address      string      `json:"address"`
	CID          string      `json:"cid_sha256"`
	Fidelity     string      `json:"fidelity"`
	Verified     bool        `json:"verified"`
	TokenCost    int         `json:"token_cost"`
	Content      string      `json:"content"`
	Kind         string      `json:"kind,omitempty"`
	DocID        string      `json:"doc_id,omitempty"`
	Page         int         `json:"page,omitempty"`
	Block        int         `json:"block,omitempty"`
	MerkleProof  []ProofStep `json:"merkle_proof,omitempty"`
	ProofAddress string      `json:"proof_address,omitempty"`
	Binary       bool        `json:"binary,omitempty"`
	BBox         any         `json:"bbox,omitempty"`
}
type Relation struct {
	From     string `json:"from"`
	Relation string `json:"relation"`
	To       string `json:"to"`
	Weight   int    `json:"weight,omitempty"`
}
type AccessMetrics struct {
	IndexTerms         int `json:"index_terms"`
	CandidateObjects   int `json:"candidate_objects"`
	ObjectsOpened      int `json:"objects_opened"`
	ExactBytesRead     int `json:"exact_bytes_read"`
	TokensExposed      int `json:"tokens_exposed"`
	GraphTermsExpanded int `json:"graph_terms_expanded"`
}
type AttentionStep struct {
	Level       string   `json:"level"`
	Action      string   `json:"action"`
	InputCount  int      `json:"input_count,omitempty"`
	OutputCount int      `json:"output_count,omitempty"`
	Uncertainty float64  `json:"uncertainty,omitempty"`
	Addresses   []string `json:"addresses,omitempty"`
}

type Packet struct {
	Schema          string                          `json:"schema"`
	Query           string                          `json:"query,omitempty"`
	Evidence        []Evidence                      `json:"evidence,omitempty"`
	Relations       []Relation                      `json:"relations,omitempty"`
	CanonicalClaims []canonicalstate.CanonicalClaim `json:"canonical_claims,omitempty"`
	ExpandableRefs  []string                        `json:"expandable_refs,omitempty"`
	Budget          Budget                          `json:"budget"`
	Metrics         AccessMetrics                   `json:"metrics"`
	Attention       []AttentionStep                 `json:"external_recursive_attention,omitempty"`
	Uncertainty     float64                         `json:"uncertainty"`
	StopReason      string                          `json:"stop_reason,omitempty"`
	Unknown         bool                            `json:"unknown"`
	Reason          string                          `json:"reason,omitempty"`
	FalseExact      int                             `json:"false_exact"`
}
