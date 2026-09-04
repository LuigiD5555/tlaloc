package tonalt1

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

// deriveCandidateID computes a stable id from the candidate's physical
// source identity ALONE — never from process order. Re-running the scanner
// over the unchanged store yields identical ids.
//
// Inputs: source pdf hash, page, containing-line region id, normalized
// containing-line text, token rune span, normalized numeric token.
func deriveCandidateID(cand Candidate) string {
	payload := strings.Join([]string{
		SelectorVersion,
		cand.Corpus.SourcePDFSHA256,
		"p" + itoa(cand.Corpus.Page),
		cand.Identity.RegionID,
		normalizeLineText(cand.Source.ContainingLineText),
		fmt.Sprintf("off%d-%d", cand.Identity.CharStart, cand.Identity.CharEnd),
		cand.Source.NumericNormalized,
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])[:32]
}

// bboxKey renders a bbox at 1-unit rounding for equality comparison
// between candidates and heterogeneous historical artifacts.
func bboxKey(box canonicaldoc.BBox) string {
	return fmt.Sprintf("%d,%d,%d,%d",
		int(math.Round(box.X1)), int(math.Round(box.Y1)),
		int(math.Round(box.X2)), int(math.Round(box.Y2)))
}

// bboxAlmostEqual reports whether two line boxes are the same physical
// line: every corner within tol store units.
func bboxAlmostEqual(a, b canonicaldoc.BBox, tol float64) bool {
	return math.Abs(a.X1-b.X1) <= tol && math.Abs(a.Y1-b.Y1) <= tol &&
		math.Abs(a.X2-b.X2) <= tol && math.Abs(a.Y2-b.Y2) <= tol
}

// spansOverlap reports whether rune spans [aStart,aEnd) and [bStart,bEnd)
// intersect.
func spansOverlap(aStart, aEnd, bStart, bEnd int) bool {
	return aStart < bEnd && bStart < aEnd
}
