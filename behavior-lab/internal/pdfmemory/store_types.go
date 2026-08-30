package pdfmemory

const (
	Schema        = "tlaloc.pdf-memory.r2"
	AddressSchema = "ohf-address.r2"
	ToolProtocol  = "tlaloc.origami-tools.r2"
	DefaultBudget = 4000
)

type SourceSpec struct {
	Path string `json:"path"`
	ID   string `json:"id,omitempty"`
}

type DocumentRef struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SourceSHA256  string `json:"source_sha256"`
	SourcePath    string `json:"source_path"`
	SourceAddress string `json:"source_address"`
	PageCount     int    `json:"page_count"`
	CanonicalPath string `json:"canonical_path"`
	DigitalPages  int    `json:"digital_pages"`
	OCRPages      int    `json:"ocr_pages"`
	RegionCount   int    `json:"region_count"`
	FigureCount   int    `json:"figure_count"`
}

type PageRef struct {
	DocID          string   `json:"doc_id"`
	Number         int      `json:"number"`
	Address        string   `json:"address"`
	CID            string   `json:"cid_sha256"`
	Bytes          int      `json:"bytes"`
	TokenEq        int      `json:"token_eq"`
	Path           string   `json:"path"`
	Blocks         []string `json:"blocks,omitempty"`
	LayoutPath     string   `json:"layout_path,omitempty"`
	LayoutCID      string   `json:"layout_cid_sha256,omitempty"`
	ExtractionMode string   `json:"extraction_mode,omitempty"`
	RegionCount    int      `json:"region_count,omitempty"`
}

type BlockRef struct {
	DocID     string `json:"doc_id"`
	Page      int    `json:"page"`
	Number    int    `json:"number"`
	Address   string `json:"address"`
	CID       string `json:"cid_sha256"`
	Bytes     int    `json:"bytes"`
	TokenEq   int    `json:"token_eq"`
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
}

type Manifest struct {
	Schema               string        `json:"schema"`
	AddressSchema        string        `json:"address_schema"`
	ToolProtocol         string        `json:"tool_protocol"`
	CarrierID            string        `json:"carrier_id"`
	SourceName           string        `json:"source_name,omitempty"` // single-document compatibility
	SourceSHA256         string        `json:"source_sha256"`         // aggregate source hash; equals source hash for one document
	SourcePath           string        `json:"source_path,omitempty"` // single-document compatibility
	DocumentCount        int           `json:"document_count"`
	PageCount            int           `json:"page_count"`
	BlockCount           int           `json:"block_count"`
	RegionCount          int           `json:"region_count"`
	CandidateCount       int           `json:"candidate_count"`
	CanonicalClaimCount  int           `json:"canonical_claim_count"`
	ConflictCount        int           `json:"conflict_count"`
	ObjectCount          int           `json:"object_count"`
	StoreRootSHA256      string        `json:"store_root_sha256"`
	CanonicalStateHash   string        `json:"canonical_state_hash_sha256"`
	CanonicalStatePath   string        `json:"canonical_state_path"`
	CandidatePath        string        `json:"candidate_path"`
	LedgerPath           string        `json:"ledger_path"`
	VerificationPlanPath string        `json:"verification_plan_path"`
	Documents            []DocumentRef `json:"documents"`
	Pages                []PageRef     `json:"pages"`
	Blocks               []BlockRef    `json:"blocks"`
	ClassificationNote   string        `json:"classification_note"`
}

type Posting struct {
	Address string `json:"address"`
	DocID   string `json:"doc_id"`
	Page    int    `json:"page"`
	Block   int    `json:"block,omitempty"`
	Kind    string `json:"kind"`
}

type Index struct {
	Schema   string           `json:"schema"`
	Postings map[string][]int `json:"postings"` // block indexes into Manifest.Blocks
}

type GraphEdge struct {
	Term   string `json:"term"`
	Weight int    `json:"weight"`
}
type GraphNode struct {
	Term              string      `json:"term"`
	DocumentFrequency int         `json:"document_frequency"`
	Neighbors         []GraphEdge `json:"neighbors,omitempty"`
}
type Graph struct {
	Schema string               `json:"schema"`
	Nodes  map[string]GraphNode `json:"nodes"`
}

type MerkleLeaf struct {
	Index   int    `json:"index"`
	Kind    string `json:"kind"`
	Address string `json:"address"`
	CID     string `json:"cid_sha256"`
	Hash    string `json:"leaf_hash_sha256"`
}
type MerkleIndex struct {
	Schema string       `json:"schema"`
	Root   string       `json:"root_sha256"`
	Leaves []MerkleLeaf `json:"leaves"`
}

type FixedCarrierMetadata struct {
	CarrierID      string `json:"carrier_id"`
	StoreRoot      string `json:"store_root_sha256"`
	SourceSHA256   string `json:"source_sha256"`
	PageCount      uint32 `json:"page_count"`
	BlockCount     uint32 `json:"block_count"`
	DocumentCount  uint32 `json:"document_count"`
	ObjectCount    uint32 `json:"object_count"`
	GraphSignature []byte `json:"graph_signature"`
	Flags          uint16 `json:"flags"`
}

type BuildResult struct {
	Manifest             Manifest             `json:"manifest"`
	FixedCarrierMetadata FixedCarrierMetadata `json:"fixed_carrier_metadata"`
	StoreDir             string               `json:"store_dir"`
}
