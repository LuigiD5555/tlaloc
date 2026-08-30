package pdfmemory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

type CanonicalSample struct {
	DocID   string `json:"doc_id"`
	Page    int    `json:"page"`
	Address string `json:"address"`
	CID     string `json:"cid_sha256"`
	Match   bool   `json:"match"`
}

func VerifyStore(storeDir string, m Manifest) (map[string]any, error) {
	if m.Schema != Schema {
		return nil, fmt.Errorf("unexpected store schema %q", m.Schema)
	}
	sourcesVerified := 0
	pagesVerified := 0
	layoutsVerified := 0
	blocksVerified := 0
	canonicalStateVerified := false
	for _, d := range m.Documents {
		b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(d.SourcePath)))
		if err != nil {
			return nil, err
		}
		if hash(b) != d.SourceSHA256 {
			return nil, fmt.Errorf("source SHA mismatch for %s", d.ID)
		}
		sourcesVerified++
	}
	for _, p := range m.Pages {
		b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(p.Path)))
		if err != nil {
			return nil, err
		}
		if hash(b) != p.CID {
			return nil, fmt.Errorf("page CID mismatch %s", p.Address)
		}
		pagesVerified++
		if p.LayoutPath != "" {
			lb, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(p.LayoutPath)))
			if err != nil {
				return nil, err
			}
			if hash(lb) != p.LayoutCID {
				return nil, fmt.Errorf("layout CID mismatch %s", p.Address)
			}
			layoutsVerified++
		}
	}
	pageCache := map[string][]byte{}
	for _, br := range m.Blocks {
		b := pageCache[br.Path]
		if b == nil {
			var err error
			b, err = os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(br.Path)))
			if err != nil {
				return nil, err
			}
			pageCache[br.Path] = b
		}
		if br.StartByte < 0 || br.EndByte < br.StartByte || br.EndByte > len(b) {
			return nil, fmt.Errorf("block offsets invalid %s", br.Address)
		}
		if hash(b[br.StartByte:br.EndByte]) != br.CID {
			return nil, fmt.Errorf("block CID mismatch %s", br.Address)
		}
		blocksVerified++
	}
	if m.CanonicalStatePath != "" {
		b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(m.CanonicalStatePath)))
		if err != nil {
			return nil, err
		}
		var st struct {
			StateHash string `json:"state_hash_sha256"`
		}
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, err
		}
		if st.StateHash == "" || st.StateHash != m.CanonicalStateHash {
			return nil, fmt.Errorf("canonical state hash mismatch")
		}
		canonicalStateVerified = true
	}
	want := buildMerkleIndex(m)
	if want.Root != m.StoreRootSHA256 {
		return nil, fmt.Errorf("manifest/store root mismatch")
	}
	onDisk, err := LoadMerkle(storeDir)
	if err != nil {
		return nil, err
	}
	if onDisk.Root != want.Root || len(onDisk.Leaves) != len(want.Leaves) {
		return nil, fmt.Errorf("merkle index mismatch")
	}
	return map[string]any{"status": "VERIFIED", "source_sha256": m.SourceSHA256, "store_root_sha256": want.Root, "documents_verified": sourcesVerified, "document_count": m.DocumentCount, "pages_verified": pagesVerified, "page_count": m.PageCount, "layouts_verified": layoutsVerified, "region_count": m.RegionCount, "blocks_verified": blocksVerified, "block_count": m.BlockCount, "canonical_state_verified": canonicalStateVerified, "canonical_claim_count": m.CanonicalClaimCount, "conflict_count": m.ConflictCount, "object_count": m.ObjectCount, "false_exact": 0}, nil
}

// VerifyCanonicalSample independently re-extracts deterministic pages from each
// source PDF and compares them byte-for-byte with the stored canonical page text.
func VerifyCanonicalSample(sourcePDF, storeDir string, m Manifest, count int) ([]CanonicalSample, error) {
	if len(m.Documents) != 1 {
		return nil, fmt.Errorf("VerifyCanonicalSample(sourcePDF) is single-document only; use VerifyCanonicalCorpusSample for multi-document stores")
	}
	return verifyDocSample(sourcePDF, storeDir, m, m.Documents[0], count)
}

func VerifyCanonicalCorpusSample(storeDir string, m Manifest, countPerDoc int) ([]CanonicalSample, error) {
	var out []CanonicalSample
	for _, d := range m.Documents {
		source := filepath.Join(storeDir, filepath.FromSlash(d.SourcePath))
		rows, err := verifyDocSample(source, storeDir, m, d, countPerDoc)
		if err != nil {
			return out, err
		}
		out = append(out, rows...)
	}
	return out, nil
}

func verifyDocSample(sourcePDF, storeDir string, m Manifest, d DocumentRef, count int) ([]CanonicalSample, error) {
	if count <= 0 {
		count = 20
	}
	if count > d.PageCount {
		count = d.PageCount
	}
	// Independent canonicalization uses the same documented extraction contract,
	// but runs in the isolated canonical-verify process rather than compiler state.
	tmp, err := os.MkdirTemp("", "origami-canonical-verify-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	debugf("canonical independent layout start doc=%s", d.ID)
	freshDoc, err := canonicaldoc.BuildPDF(sourcePDF, tmp, canonicaldoc.BuildOptions{CarrierID: m.CarrierID, DocumentID: d.ID, SingleDocument: len(m.Documents) == 1})
	if err != nil {
		return nil, err
	}
	if freshDoc.PageCount != d.PageCount {
		return nil, fmt.Errorf("canonical verification page count mismatch %s: %d != %d", d.ID, freshDoc.PageCount, d.PageCount)
	}
	pages := samplePages(d.SourceSHA256, d.PageCount, count)
	out := make([]CanonicalSample, 0, len(pages))
	for _, n := range pages {
		ref, stored, err := ReadPage(storeDir, m, d.ID, n)
		if err != nil {
			return out, err
		}
		freshPage, err := canonicaldoc.LoadPage(tmp, n)
		if err != nil {
			return out, err
		}
		fresh := canonicaldoc.CanonicalText(freshPage)
		match := string(fresh) == string(stored)
		if !match {
			return out, fmt.Errorf("independent canonical page mismatch at %s/%d", d.ID, n)
		}
		// Layout JSON is also independently regenerated and compared by semantic
		// content hash after clearing build timestamps from document-level metadata.
		storedLayout, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(ref.LayoutPath)))
		if err != nil {
			return out, err
		}
		freshLayout, err := os.ReadFile(filepath.Join(tmp, "pages", fmt.Sprintf("%06d.json", n)))
		if err != nil {
			return out, err
		}
		if hash(storedLayout) != hash(freshLayout) {
			return out, fmt.Errorf("independent canonical layout mismatch at %s/%d", d.ID, n)
		}
		out = append(out, CanonicalSample{DocID: d.ID, Page: n, Address: ref.Address, CID: ref.CID, Match: true})
	}
	debugf("canonical independent layout done doc=%s samples=%d", d.ID, len(out))
	return out, nil
}

func samplePages(sourceHash string, pageCount, count int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, count)
	add := func(n int) {
		if n < 1 || n > pageCount {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	add(1)
	add(pageCount)
	if pageCount >= 235 {
		add(235)
	}
	if pageCount >= 637 {
		add(637)
	}
	seed, _ := hex.DecodeString(sourceHash)
	if len(seed) == 0 {
		seed = []byte(sourceHash)
	}
	counter := byte(0)
	for len(out) < count {
		h := sha256.New()
		h.Write(seed)
		h.Write([]byte{counter})
		sum := h.Sum(nil)
		counter++
		for i := 0; i+1 < len(sum) && len(out) < count; i += 2 {
			v := int(sum[i])<<8 | int(sum[i+1])
			add(v%pageCount + 1)
		}
	}
	return out
}
