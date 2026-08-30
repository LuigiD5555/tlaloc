package pdfmemory

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Expand(storeDir string, m Manifest, address, fidelity string, maxTokens int) (Packet, error) {
	if maxTokens <= 0 {
		maxTokens = DefaultBudget
	}
	p := Packet{Schema: ToolProtocol + ".context-packet", Budget: Budget{MaxTokens: maxTokens, RemainingTokens: maxTokens}}
	if fidelity == "" {
		fidelity = "exact"
	}
	if strings.Contains(address, "/regions/") {
		rr, err := ReadRegion(storeDir, m, address)
		if err != nil {
			p.Unknown = true
			p.Reason = err.Error()
			return p, nil
		}
		p.Metrics.ObjectsOpened = 1
		p.Metrics.ExactBytesRead = len(rr.Bytes)
		merkle, err := LoadMerkle(storeDir)
		if err != nil {
			return p, err
		}
		proofAddr := rr.Page.Address + "/layout"
		proof, ok := MerkleProof(merkle, proofAddr)
		if !ok {
			return p, fmt.Errorf("layout missing from Merkle index")
		}
		leafHash := hex.EncodeToString(sha([]byte("layout:" + proofAddr + ":" + rr.Page.LayoutCID)))
		if !VerifyMerkleProof(leafHash, proof, merkle.Root) {
			return p, fmt.Errorf("layout Merkle proof failed")
		}
		cost := estimateTokens(rr.Bytes)
		if rr.Binary {
			cost = 64
		}
		if cost > maxTokens {
			p.Unknown = true
			p.Reason = "BUDGET_EXCEEDED_EXACT"
			return p, nil
		}
		content := string(rr.Bytes)
		if rr.Binary {
			meta, _ := json.Marshal(map[string]any{"kind": rr.Region.Kind, "cid_sha256": rr.Region.CID, "bbox": rr.Region.BBox, "bytes": len(rr.Bytes)})
			content = string(meta)
		}
		p.Evidence = []Evidence{{Address: rr.Region.Address, CID: rr.Region.CID, Fidelity: fidelity, Verified: true, TokenCost: cost, Content: content, Kind: rr.Region.Kind, DocID: rr.Page.DocID, Page: rr.Page.Number, MerkleProof: proof, ProofAddress: proofAddr, Binary: rr.Binary, BBox: rr.Region.BBox}}
		p.Budget.UsedTokens = cost
		p.Budget.RemainingTokens -= cost
		p.Metrics.TokensExposed = cost
		return p, nil
	}
	doc, page, kind, block, err := ParseAddress(m.CarrierID, address)
	if err != nil {
		p.Unknown = true
		p.Reason = err.Error()
		return p, nil
	}
	doc = resolveDocAlias(m, doc)
	merkle, err := LoadMerkle(storeDir)
	if err != nil {
		return p, err
	}
	switch kind {
	case "page":
		ref, b, err := ReadPage(storeDir, m, doc, page)
		if err != nil {
			return p, err
		}
		p.Metrics.ObjectsOpened++
		p.Metrics.ExactBytesRead += len(b)
		cost := estimateTokens(b)
		if fidelity == "exact" {
			if cost > maxTokens {
				p.Unknown = true
				p.Reason = "BUDGET_EXCEEDED_EXACT"
				p.ExpandableRefs = append([]string(nil), ref.Blocks...)
				return p, nil
			}
			ev, err := evidenceForObject(merkle, "page", ref.Address, ref.CID, b, cost, "exact", doc, page, 0)
			if err != nil {
				return p, err
			}
			p.Evidence = []Evidence{ev}
			p.Budget.UsedTokens = cost
			p.Budget.RemainingTokens -= cost
			p.Metrics.TokensExposed = cost
			return p, nil
		}
		// Bounded page unfolding: preserve exact verified block boundaries rather than truncate bytes.
		for _, baddr := range ref.Blocks {
			br, bb, err := ReadBlock(storeDir, m, baddr)
			if err != nil {
				return p, err
			}
			c := estimateTokens(bb)
			if c > p.Budget.RemainingTokens {
				p.ExpandableRefs = append(p.ExpandableRefs, baddr)
				continue
			}
			ev, err := evidenceForObject(merkle, "block", br.Address, br.CID, bb, c, "evidence", br.DocID, br.Page, br.Number)
			if err != nil {
				return p, err
			}
			p.Evidence = append(p.Evidence, ev)
			p.Budget.UsedTokens += c
			p.Budget.RemainingTokens -= c
			p.Metrics.ObjectsOpened++
			p.Metrics.ExactBytesRead += len(bb)
			p.Metrics.TokensExposed += c
		}
		if len(p.Evidence) == 0 {
			p.Unknown = true
			p.Reason = "no page block fits context budget"
		}
		return p, nil
	case "block":
		_ = block
		br, b, err := ReadBlock(storeDir, m, address)
		if err != nil {
			return p, err
		}
		p.Metrics.ObjectsOpened++
		p.Metrics.ExactBytesRead += len(b)
		cost := estimateTokens(b)
		if fidelity == "exact" && cost > maxTokens {
			p.Unknown = true
			p.Reason = "BUDGET_EXCEEDED_EXACT"
			return p, nil
		}
		if cost > maxTokens {
			p.Unknown = true
			p.Reason = "BUDGET_EXCEEDED"
			return p, nil
		}
		ev, err := evidenceForObject(merkle, "block", br.Address, br.CID, b, cost, fidelity, br.DocID, br.Page, br.Number)
		if err != nil {
			return p, err
		}
		p.Evidence = []Evidence{ev}
		p.Budget.UsedTokens = cost
		p.Budget.RemainingTokens -= cost
		p.Metrics.TokensExposed = cost
		return p, nil
	case "source":
		for _, d := range m.Documents {
			if d.ID == doc {
				b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(d.SourcePath)))
				if err != nil {
					return p, err
				}
				cost := estimateTokens(b)
				p.Metrics.ObjectsOpened = 1
				p.Metrics.ExactBytesRead = len(b)
				if cost > maxTokens {
					p.Unknown = true
					p.Reason = "BUDGET_EXCEEDED_EXACT_SOURCE"
					return p, nil
				}
				p.Unknown = true
				p.Reason = "binary PDF source is verified by origami_verify but is not injected into model context"
				return p, nil
			}
		}
		p.Unknown = true
		p.Reason = "source address not found"
		return p, nil
	default:
		p.Unknown = true
		p.Reason = "unsupported address kind"
		return p, nil
	}
}
