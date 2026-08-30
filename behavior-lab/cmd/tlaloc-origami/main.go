package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/target"
)

type VerificationReport struct {
	Schema                string                      `json:"schema"`
	CarrierBytes          int                         `json:"carrier_bytes"`
	CarrierMaxBytes       int                         `json:"carrier_max_bytes"`
	CarrierSHA256         string                      `json:"carrier_sha256"`
	PromptSHA256          string                      `json:"prompt_sha256"`
	SourceSHA256          string                      `json:"source_sha256"`
	StoreRootSHA256       string                      `json:"store_root_sha256"`
	DocumentCount         int                         `json:"document_count"`
	PageCount             int                         `json:"page_count"`
	BlockCount            int                         `json:"block_count"`
	RegionCount           int                         `json:"region_count"`
	CandidateCount        int                         `json:"candidate_count"`
	CanonicalClaimCount   int                         `json:"canonical_claim_count"`
	ConflictCount         int                         `json:"conflict_count"`
	CanonicalStateHash    string                      `json:"canonical_state_hash_sha256"`
	ObjectCount           int                         `json:"object_count"`
	PagesVerified         int                         `json:"pages_verified"`
	BlocksVerified        int                         `json:"blocks_verified"`
	CanonicalSample       []pdfmemory.CanonicalSample `json:"canonical_sample"`
	CanonicalSamplePassed bool                        `json:"canonical_sample_passed"`
	OCRRequired           bool                        `json:"ocr_required"`
	T0PlaintextBoot       bool                        `json:"t0_plaintext_boot"`
	FalseExact            int                         `json:"false_exact"`
}
type BundleManifest struct {
	Schema              string `json:"schema"`
	Carrier             string `json:"carrier"`
	Prompt              string `json:"prompt"`
	Store               string `json:"store"`
	CarrierSHA256       string `json:"carrier_sha256"`
	PromptSHA256        string `json:"prompt_sha256"`
	StoreRootSHA256     string `json:"store_root_sha256"`
	SourceSHA256        string `json:"source_sha256"`
	DocumentCount       int    `json:"document_count"`
	PageCount           int    `json:"page_count"`
	BlockCount          int    `json:"block_count"`
	RegionCount         int    `json:"region_count"`
	CanonicalClaimCount int    `json:"canonical_claim_count"`
	CanonicalStateHash  string `json:"canonical_state_hash_sha256"`
	ObjectCount         int    `json:"object_count"`
	FixedCarrierBytes   int    `json:"fixed_carrier_bytes"`
	MaxCarrierBytes     int    `json:"max_carrier_bytes"`
}
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "compile":
		compileCmd(os.Args[2:])
	case "boot":
		toolCmd("origami_boot", os.Args[2:])
	case "query":
		toolCmd("origami_query", os.Args[2:])
	case "expand":
		toolCmd("origami_expand", os.Args[2:])
	case "verify":
		toolCmd("origami_verify", os.Args[2:])
	case "canonical-verify":
		canonicalVerifyCmd(os.Args[2:])
	case "replay-state":
		replayStateCmd(os.Args[2:])
	case "chat":
		chatCmd(os.Args[2:])
	case "tools":
		b, _ := json.MarshalIndent(target.OrigamiFixedTools(), "", "  ")
		fmt.Println(string(b))
	default:
		usage()
		os.Exit(2)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "tlaloc-origami <compile|boot|query|expand|verify|canonical-verify|replay-state|chat|tools> [flags]")
}
