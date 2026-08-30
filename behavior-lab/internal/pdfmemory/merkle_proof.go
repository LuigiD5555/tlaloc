package pdfmemory

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func evidenceForObject(mi MerkleIndex, kind, address, cid string, b []byte, cost int, fidelity, doc string, page, block int) (Evidence, error) {
	if hash(b) != cid {
		return Evidence{}, fmt.Errorf("object CID mismatch")
	}
	proof, ok := MerkleProof(mi, address)
	if !ok {
		return Evidence{}, fmt.Errorf("object missing from Merkle index: %s", address)
	}
	leafHash := hex.EncodeToString(sha([]byte(kind + ":" + address + ":" + cid)))
	if !VerifyMerkleProof(leafHash, proof, mi.Root) {
		return Evidence{}, fmt.Errorf("Merkle proof failed for %s", address)
	}
	return Evidence{Address: address, CID: cid, Fidelity: fidelity, Verified: true, TokenCost: cost, Content: string(b), Kind: kind, DocID: doc, Page: page, Block: block, MerkleProof: proof}, nil
}

func MerkleProof(mi MerkleIndex, address string) ([]ProofStep, bool) {
	idx := -1
	for _, l := range mi.Leaves {
		if l.Address == address {
			idx = l.Index
			break
		}
	}
	if idx < 0 {
		return nil, false
	}
	nodes := make([][]byte, len(mi.Leaves))
	for i, l := range mi.Leaves {
		b, err := hex.DecodeString(l.Hash)
		if err != nil {
			return nil, false
		}
		nodes[i] = b
	}
	pos := idx
	var proof []ProofStep
	for len(nodes) > 1 {
		sibling := pos ^ 1
		if sibling >= len(nodes) {
			sibling = pos
		}
		side := "right"
		if sibling < pos {
			side = "left"
		}
		proof = append(proof, ProofStep{Side: side, SHA256: hex.EncodeToString(nodes[sibling])})
		var next [][]byte
		for i := 0; i < len(nodes); i += 2 {
			right := nodes[i]
			if i+1 < len(nodes) {
				right = nodes[i+1]
			}
			next = append(next, sha(append(append([]byte{}, nodes[i]...), right...)))
		}
		pos /= 2
		nodes = next
	}
	return proof, true
}
func VerifyMerkleProof(leafHash string, proof []ProofStep, root string) bool {
	cur, err := hex.DecodeString(leafHash)
	if err != nil {
		return false
	}
	for _, p := range proof {
		s, err := hex.DecodeString(p.SHA256)
		if err != nil {
			return false
		}
		if p.Side == "left" {
			cur = sha(append(append([]byte{}, s...), cur...))
		} else {
			cur = sha(append(append([]byte{}, cur...), s...))
		}
	}
	return hex.EncodeToString(cur) == root
}

func PageAddresses(m Manifest, nums []int) []string {
	var out []string
	if len(m.Documents) != 1 {
		return out
	}
	doc := m.Documents[0].ID
	for _, n := range nums {
		for _, p := range m.Pages {
			if p.DocID == doc && p.Number == n {
				out = append(out, p.Address)
			}
		}
	}
	sort.Strings(out)
	return out
}
func ValidateBudget(n int) error {
	if n < 1 || n > 32768 {
		return fmt.Errorf("budget out of bounds: %d", n)
	}
	return nil
}

func containsAny(s string, terms []string) bool {
	for _, t := range terms {
		if strings.Contains(strings.ToLower(s), strings.ToLower(t)) {
			return true
		}
	}
	return false
}
