package pdfmemory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"tlaloc.local/behaviorlab/internal/canonicalstate"
)

func writeObject(outDir, cid, ext string, data []byte) (string, error) {
	dir := filepath.Join(outDir, "objects", cid[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	rel := filepath.ToSlash(filepath.Join("objects", cid[:2], cid+"."+ext))
	path := filepath.Join(outDir, filepath.FromSlash(rel))
	if old, err := os.ReadFile(path); err == nil {
		if !bytes.Equal(old, data) {
			return "", fmt.Errorf("CID collision for %s", cid)
		}
		return rel, nil
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return rel, nil
}

type blockSegment struct {
	Start, End int
	Data       []byte
}

func segmentBlocks(p []byte) []blockSegment {
	if len(p) == 0 {
		return []blockSegment{{0, 0, []byte{}}}
	}
	lines := bytes.SplitAfter(p, []byte("\n"))
	var out []blockSegment
	start := 0
	offset := 0
	lineCount := 0
	for i, line := range lines {
		offset += len(line)
		lineCount++
		blank := len(bytes.TrimSpace(line)) == 0
		last := i == len(lines)-1
		if (blank && lineCount >= 2) || lineCount >= 12 || last {
			if offset > start {
				data := append([]byte(nil), p[start:offset]...)
				out = append(out, blockSegment{start, offset, data})
			}
			start = offset
			lineCount = 0
		}
	}
	if start < len(p) {
		out = append(out, blockSegment{start, len(p), append([]byte(nil), p[start:]...)})
	}
	return out
}
func classifyBlock(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "whitespace"
	}
	lines := strings.Split(s, "\n")
	first := strings.TrimSpace(lines[0])
	if len(first) <= 90 && (headingLike(first) || strings.HasPrefix(first, "Chapter ") || strings.HasPrefix(first, "CHAPTER ")) {
		return "heading"
	}
	math := 0
	code := 0
	table := 0
	for _, r := range s {
		if strings.ContainsRune("=+*/^<>≤≥∑∫√", r) {
			math++
		}
		if strings.ContainsRune("{}[]();_", r) {
			code++
		}
	}
	for _, l := range lines {
		if strings.Contains(l, "    ") {
			table++
		}
		if len(l)-len(strings.TrimLeft(l, " \t")) >= 4 {
			code++
		}
	}
	if math > 12 && math*20 > len([]rune(s)) {
		return "equation"
	}
	if code > len(lines)*2 {
		return "code"
	}
	if table >= 2 {
		return "table"
	}
	return "text"
}
func headingLike(s string) bool {
	r := []rune(s)
	if len(r) < 2 {
		return false
	}
	letters, upper := 0, 0
	for _, c := range r {
		if unicode.IsLetter(c) {
			letters++
			if unicode.IsUpper(c) {
				upper++
			}
		}
	}
	return letters > 0 && upper*100/letters >= 70
}

func buildMerkleIndex(m Manifest) MerkleIndex {
	var leaves []MerkleLeaf
	add := func(kind, address, cid string) {
		h := sha([]byte(kind + ":" + address + ":" + cid))
		leaves = append(leaves, MerkleLeaf{Index: len(leaves), Kind: kind, Address: address, CID: cid, Hash: hex.EncodeToString(h)})
	}
	for _, d := range m.Documents {
		add("source", d.SourceAddress, d.SourceSHA256)
	}
	for _, p := range m.Pages {
		add("page", p.Address, p.CID)
		if p.LayoutCID != "" {
			add("layout", p.Address+"/layout", p.LayoutCID)
		}
	}
	for _, b := range m.Blocks {
		add("block", b.Address, b.CID)
	}
	if m.CanonicalStateHash != "" {
		add("canonical_state", "ohf://"+m.CarrierID+"/canonical-state", m.CanonicalStateHash)
	}
	hashes := make([][]byte, len(leaves))
	for i, l := range leaves {
		h, _ := hex.DecodeString(l.Hash)
		hashes[i] = h
	}
	return MerkleIndex{Schema: Schema + ".merkle-index", Root: merkleHex(hashes), Leaves: leaves}
}

type preMerkleEvidenceVerifier struct {
	StoreDir string
	Manifest *Manifest
}

func (v preMerkleEvidenceVerifier) Verify(address, cid string) (bool, error) {
	doc, n, kind, _, err := ParseAddress(v.Manifest.CarrierID, address)
	if err != nil {
		return false, nil
	}
	doc = resolveDocAlias(*v.Manifest, doc)
	switch kind {
	case "page":
		ref, b, err := ReadPage(v.StoreDir, *v.Manifest, doc, n)
		if err != nil {
			return false, nil
		}
		if cid != "" && cid != ref.CID {
			return false, nil
		}
		return hash(b) == ref.CID, nil
	case "block":
		ref, b, err := ReadBlock(v.StoreDir, *v.Manifest, address)
		if err != nil {
			return false, nil
		}
		if cid != "" && cid != ref.CID {
			return false, nil
		}
		return hash(b) == ref.CID, nil
	default:
		return false, nil
	}
}
func hashJSONValue(v any) string { b, _ := json.Marshal(v); return hash(b) }
func canonicalEvidenceAddresses(s canonicalstate.State) []string {
	seen := map[string]struct{}{}
	for _, c := range s.Claims {
		for _, e := range c.Evidence {
			seen[e.Address] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

func pdfPageCount(path string) (int, error) {
	out, err := exec.Command("pdfinfo", path).Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if err == nil {
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("pdfinfo Pages field missing")
}
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func sha(b []byte) []byte  { s := sha256.Sum256(b); return append([]byte(nil), s[:]...) }
func estimateTokens(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return (len(b) + 3) / 4
}
func merkleHex(leaves [][]byte) string {
	if len(leaves) == 0 {
		return hash(nil)
	}
	nodes := make([][]byte, len(leaves))
	for i, l := range leaves {
		nodes[i] = append([]byte(nil), l...)
	}
	for len(nodes) > 1 {
		var next [][]byte
		for i := 0; i < len(nodes); i += 2 {
			right := nodes[i]
			if i+1 < len(nodes) {
				right = nodes[i+1]
			}
			joined := append(append([]byte{}, nodes[i]...), right...)
			next = append(next, sha(joined))
		}
		nodes = next
	}
	return hex.EncodeToString(nodes[0])
}
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !(unicode.IsLetter(r) || unicode.IsDigit(r)) })
	seen := map[string]struct{}{}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len([]rune(f)) < 3 || stop(f) {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}
func stop(s string) bool {
	switch s {
	case "the", "and", "for", "that", "with", "from", "this", "are", "was", "were", "you", "your", "into", "what", "where", "which", "how", "why", "never", "exists", "exist", "que", "para", "por", "con", "del", "las", "los", "una", "uno", "como", "qué", "cómo":
		return true
	}
	return false
}
func slugID(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	dash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func debugf(format string, args ...any) {
	if os.Getenv("TLALOC_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[pdfmemory %s] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
	}
}

type minHashSketch struct{ mins [64]uint32 }

func newMinHashSketch() *minHashSketch {
	s := &minHashSketch{}
	for i := range s.mins {
		s.mins[i] = ^uint32(0)
	}
	return s
}
func (s *minHashSketch) Add(term string) {
	base := sha256.Sum256([]byte(term))
	for i := 0; i < 64; i++ {
		a := uint32(base[(i%8)*4])<<24 | uint32(base[(i%8)*4+1])<<16 | uint32(base[(i%8)*4+2])<<8 | uint32(base[(i%8)*4+3])
		v := a ^ (uint32(i+1) * 0x9e3779b9)
		v ^= v >> 16
		v *= 0x85ebca6b
		v ^= v >> 13
		v *= 0xc2b2ae35
		v ^= v >> 16
		if v < s.mins[i] {
			s.mins[i] = v
		}
	}
}
func (s *minHashSketch) Bytes() []byte {
	out := make([]byte, 256)
	for i, v := range s.mins {
		out[i*4] = byte(v >> 24)
		out[i*4+1] = byte(v >> 16)
		out[i*4+2] = byte(v >> 8)
		out[i*4+3] = byte(v)
	}
	return out
}
