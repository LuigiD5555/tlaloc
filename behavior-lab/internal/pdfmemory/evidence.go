package pdfmemory

import (
	"encoding/hex"
)

type StoreEvidenceVerifier struct {
	StoreDir string
	Manifest Manifest
	Merkle   MerkleIndex
}

func NewStoreEvidenceVerifier(storeDir string, manifest Manifest) (*StoreEvidenceVerifier, error) {
	mi, err := LoadMerkle(storeDir)
	if err != nil {
		return nil, err
	}
	return &StoreEvidenceVerifier{StoreDir: storeDir, Manifest: manifest, Merkle: mi}, nil
}

func (v *StoreEvidenceVerifier) Verify(address, cid string) (bool, error) {
	doc, n, kind, _, err := ParseAddress(v.Manifest.CarrierID, address)
	if err != nil {
		return false, nil
	}
	doc = resolveDocAlias(v.Manifest, doc)
	mi := v.Merkle
	switch kind {
	case "page":
		ref, b, err := ReadPage(v.StoreDir, v.Manifest, doc, n)
		if err != nil {
			return false, nil
		}
		if cid != "" && cid != ref.CID {
			return false, nil
		}
		return verifyLeaf(mi, "page", ref.Address, ref.CID, b), nil
	case "block":
		ref, b, err := ReadBlock(v.StoreDir, v.Manifest, address)
		if err != nil {
			return false, nil
		}
		if cid != "" && cid != ref.CID {
			return false, nil
		}
		return verifyLeaf(mi, "block", ref.Address, ref.CID, b), nil
	default:
		return false, nil
	}
}
func verifyLeaf(mi MerkleIndex, kind, address, cid string, b []byte) bool {
	if hash(b) != cid {
		return false
	}
	proof, ok := MerkleProof(mi, address)
	if !ok {
		return false
	}
	leaf := hex.EncodeToString(sha([]byte(kind + ":" + address + ":" + cid)))
	return VerifyMerkleProof(leaf, proof, mi.Root)
}
