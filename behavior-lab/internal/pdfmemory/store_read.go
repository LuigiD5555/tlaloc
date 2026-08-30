package pdfmemory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func Load(storeDir string) (Manifest, Index, error) {
	var m Manifest
	var idx Index
	b, err := os.ReadFile(filepath.Join(storeDir, "manifest.json"))
	if err != nil {
		return m, idx, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, idx, err
	}
	b, err = os.ReadFile(filepath.Join(storeDir, "index.json"))
	if err != nil {
		return m, idx, err
	}
	if err := json.Unmarshal(b, &idx); err != nil {
		return m, idx, err
	}
	if m.Schema != Schema {
		return m, idx, fmt.Errorf("unexpected store schema %q", m.Schema)
	}
	return m, idx, nil
}
func LoadGraph(storeDir string) (Graph, error) {
	var g Graph
	b, err := os.ReadFile(filepath.Join(storeDir, "graph.json"))
	if err != nil {
		return g, err
	}
	err = json.Unmarshal(b, &g)
	return g, err
}
func LoadMerkle(storeDir string) (MerkleIndex, error) {
	var m MerkleIndex
	b, err := os.ReadFile(filepath.Join(storeDir, "merkle.json"))
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	return m, err
}

func ReadPage(storeDir string, m Manifest, docID string, n int) (PageRef, []byte, error) {
	for _, ref := range m.Pages {
		if ref.DocID == docID && ref.Number == n {
			b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(ref.Path)))
			if err != nil {
				return PageRef{}, nil, err
			}
			if hash(b) != ref.CID {
				return PageRef{}, nil, fmt.Errorf("page %s/%d CID mismatch", docID, n)
			}
			return ref, b, nil
		}
	}
	return PageRef{}, nil, fmt.Errorf("page out of range: %s/%d", docID, n)
}
func ReadBlock(storeDir string, m Manifest, address string) (BlockRef, []byte, error) {
	for _, ref := range m.Blocks {
		if ref.Address != address {
			continue
		}
		b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(ref.Path)))
		if err != nil {
			return BlockRef{}, nil, err
		}
		if ref.StartByte < 0 || ref.EndByte < ref.StartByte || ref.EndByte > len(b) {
			return BlockRef{}, nil, fmt.Errorf("block offsets invalid")
		}
		slice := append([]byte(nil), b[ref.StartByte:ref.EndByte]...)
		if hash(slice) != ref.CID {
			return BlockRef{}, nil, fmt.Errorf("block CID mismatch")
		}
		return ref, slice, nil
	}
	return BlockRef{}, nil, fmt.Errorf("block address not found")
}

func PageNumberFromAddress(carrierID, address string) (int, error) {
	doc, n, kind, _, err := ParseAddress(carrierID, address)
	if err != nil {
		return 0, err
	}
	_ = doc
	if kind != "page" {
		return 0, fmt.Errorf("address is not a page")
	}
	return n, nil
}
func ParseAddress(carrierID, address string) (docID string, page int, kind string, block int, err error) {
	prefix := "ohf://" + carrierID + "/"
	if !strings.HasPrefix(address, prefix) {
		return "", 0, "", 0, fmt.Errorf("address not in carrier %q", carrierID)
	}
	rest := strings.TrimPrefix(address, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) >= 2 && parts[0] == "pages" {
		n, e := strconv.Atoi(parts[1])
		if e != nil {
			return "", 0, "", 0, fmt.Errorf("invalid page address")
		}
		docID = singleDocAlias
		if len(parts) == 2 {
			return docID, n, "page", 0, nil
		}
		if len(parts) == 4 && parts[2] == "blocks" {
			b, e := strconv.Atoi(parts[3])
			if e != nil {
				return "", 0, "", 0, fmt.Errorf("invalid block address")
			}
			return docID, n, "block", b, nil
		}
	}
	if len(parts) >= 4 && parts[0] == "docs" && parts[2] == "pages" {
		docID = parts[1]
		n, e := strconv.Atoi(parts[3])
		if e != nil {
			return "", 0, "", 0, fmt.Errorf("invalid page address")
		}
		if len(parts) == 4 {
			return docID, n, "page", 0, nil
		}
		if len(parts) == 6 && parts[4] == "blocks" {
			b, e := strconv.Atoi(parts[5])
			if e != nil {
				return "", 0, "", 0, fmt.Errorf("invalid block address")
			}
			return docID, n, "block", b, nil
		}
	}
	if len(parts) == 2 && parts[0] == "source" {
		return parts[1], 0, "source", 0, nil
	}
	return "", 0, "", 0, fmt.Errorf("unsupported address")
}

const singleDocAlias = "__single__"

func resolveDocAlias(m Manifest, docID string) string {
	if docID == singleDocAlias && len(m.Documents) == 1 {
		return m.Documents[0].ID
	}
	return docID
}
func pageAddress(carrierID, docID string, page int, single bool) string {
	if single {
		return fmt.Sprintf("ohf://%s/pages/%06d", carrierID, page)
	}
	return fmt.Sprintf("ohf://%s/docs/%s/pages/%06d", carrierID, docID, page)
}
func blockAddress(pageAddress string, block int) string {
	return fmt.Sprintf("%s/blocks/%04d", pageAddress, block)
}

func Search(m Manifest, idx Index, q string, limit int) []BlockRef {
	if limit <= 0 {
		limit = 8
	}
	scores := map[int]int{}
	for _, term := range tokenize(q) {
		for _, bi := range idx.Postings[term] {
			if bi >= 0 && bi < len(m.Blocks) {
				scores[bi]++
			}
		}
	}
	type row struct{ i, s int }
	rows := make([]row, 0, len(scores))
	for i, s := range scores {
		rows = append(rows, row{i, s})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].s == rows[j].s {
			return m.Blocks[rows[i].i].Address < m.Blocks[rows[j].i].Address
		}
		return rows[i].s > rows[j].s
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]BlockRef, len(rows))
	for i, r := range rows {
		out[i] = m.Blocks[r.i]
	}
	return out
}

func buildGraph(idx Index, blocks []BlockRef, pageTerms map[string]map[string]struct{}, maxNodes, maxNeighbors int) Graph {
	df := map[string]int{}
	// pageTerms is already deduplicated per page; counting it directly avoids
	// allocating one seen-page map for every term in the corpus.
	for _, set := range pageTerms {
		for term := range set {
			df[term]++
		}
	}
	for term := range idx.Postings {
		if _, ok := df[term]; !ok {
			df[term] = 0
		}
	}
	type tr struct {
		t  string
		df int
	}
	rank := make([]tr, 0, len(df))
	for t, n := range df {
		rank = append(rank, tr{t, n})
	}
	sort.Slice(rank, func(i, j int) bool {
		if rank[i].df == rank[j].df {
			return rank[i].t < rank[j].t
		}
		return rank[i].df > rank[j].df
	})
	if len(rank) > maxNodes {
		rank = rank[:maxNodes]
	}
	allowed := map[string]struct{}{}
	for _, r := range rank {
		allowed[r.t] = struct{}{}
	}
	co := map[string]map[string]int{}
	for _, set := range pageTerms {
		var terms []string
		for t := range set {
			if _, ok := allowed[t]; ok {
				terms = append(terms, t)
			}
		}
		sort.Strings(terms)
		if len(terms) > 64 {
			terms = terms[:64]
		}
		for i := 0; i < len(terms); i++ {
			for j := i + 1; j < len(terms); j++ {
				if co[terms[i]] == nil {
					co[terms[i]] = map[string]int{}
				}
				if co[terms[j]] == nil {
					co[terms[j]] = map[string]int{}
				}
				co[terms[i]][terms[j]]++
				co[terms[j]][terms[i]]++
			}
		}
	}
	g := Graph{Schema: Schema + ".term-graph", Nodes: map[string]GraphNode{}}
	for _, r := range rank {
		type ew struct {
			t string
			w int
		}
		var es []ew
		for t, w := range co[r.t] {
			es = append(es, ew{t, w})
		}
		sort.Slice(es, func(i, j int) bool {
			if es[i].w == es[j].w {
				return es[i].t < es[j].t
			}
			return es[i].w > es[j].w
		})
		if len(es) > maxNeighbors {
			es = es[:maxNeighbors]
		}
		n := GraphNode{Term: r.t, DocumentFrequency: r.df}
		for _, e := range es {
			n.Neighbors = append(n.Neighbors, GraphEdge{Term: e.t, Weight: e.w})
		}
		g.Nodes[r.t] = n
	}
	return g
}

func graphExpandTerms(g Graph, terms []string, perTerm int) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, t := range terms {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
		if n, ok := g.Nodes[t]; ok {
			for i, e := range n.Neighbors {
				if i >= perTerm {
					break
				}
				if _, ok := seen[e.Term]; !ok {
					seen[e.Term] = struct{}{}
					out = append(out, e.Term)
				}
			}
		}
	}
	return out
}
