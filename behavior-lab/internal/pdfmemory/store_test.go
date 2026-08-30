package pdfmemory

import (
	"bytes"
	"testing"
)

func TestR2AddressParsingSingleAndMultiDocument(t *testing.T) {
	doc, n, kind, block, err := ParseAddress("book", "ohf://book/pages/000637")
	if err != nil || doc != singleDocAlias || n != 637 || kind != "page" || block != 0 {
		t.Fatalf("single page parse: %q %d %q %d %v", doc, n, kind, block, err)
	}
	doc, n, kind, block, err = ParseAddress("book", "ohf://book/pages/000637/blocks/0004")
	if err != nil || n != 637 || kind != "block" || block != 4 {
		t.Fatalf("single block parse: %q %d %q %d %v", doc, n, kind, block, err)
	}
	doc, n, kind, block, err = ParseAddress("book", "ohf://book/docs/algorithms/pages/000235/blocks/0002")
	if err != nil || doc != "algorithms" || n != 235 || kind != "block" || block != 2 {
		t.Fatalf("multi block parse: %q %d %q %d %v", doc, n, kind, block, err)
	}
	if _, _, _, _, err := ParseAddress("book", "ohf://other/pages/000637"); err == nil {
		t.Fatal("expected carrier mismatch")
	}
}

func TestMasterPromptDelegatesBootstrapToCarrierT0(t *testing.T) {
	p := MasterPrompt()
	for _, want := range []string{"T0 PLAINTEXT BOOT", "origami_boot", "ORIGAMI_TOOL_REQUIRED", "FALSE_EXACT", "Merkle", "OCR"} {
		if !contains(p, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
	if len(p) > 5000 {
		t.Fatalf("R2 prompt should remain bootstrap-sized, got %d bytes", len(p))
	}
}

func TestBlockSegmentationIsExactAndClassified(t *testing.T) {
	page := []byte("CHAPTER 7 GRAPH TRAVERSAL\n\n    for i := 0; i < n; i++ {\n        visit(i);\n    }\n\nA normal paragraph follows here.\n")
	segs := segmentBlocks(page)
	var joined []byte
	for _, s := range segs {
		joined = append(joined, s.Data...)
	}
	if !bytes.Equal(joined, page) {
		t.Fatalf("segments do not reconstruct page exactly\nwant=%q\ngot=%q", page, joined)
	}
	if len(segs) < 2 {
		t.Fatalf("expected multiple blocks, got %d", len(segs))
	}
	if got := classifyBlock(segs[0].Data); got != "heading" {
		t.Fatalf("first block kind=%s", got)
	}
}

func TestMerkleProofRoundTrip(t *testing.T) {
	m := Manifest{CarrierID: "c", Documents: []DocumentRef{{ID: "d", SourceAddress: "ohf://c/source/d", SourceSHA256: hash([]byte("pdf"))}}, Pages: []PageRef{{DocID: "d", Number: 1, Address: "ohf://c/pages/000001", CID: hash([]byte("page"))}}, Blocks: []BlockRef{{DocID: "d", Page: 1, Number: 1, Address: "ohf://c/pages/000001/blocks/0001", CID: hash([]byte("block"))}}}
	mi := buildMerkleIndex(m)
	proof, ok := MerkleProof(mi, m.Blocks[0].Address)
	if !ok {
		t.Fatal("proof missing")
	}
	leaf := hash([]byte("block:" + m.Blocks[0].Address + ":" + m.Blocks[0].CID))
	if !VerifyMerkleProof(leaf, proof, mi.Root) {
		t.Fatal("proof should verify")
	}
	if VerifyMerkleProof(hash([]byte("wrong")), proof, mi.Root) {
		t.Fatal("wrong leaf verified")
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && index(s, sub) >= 0 }
func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
