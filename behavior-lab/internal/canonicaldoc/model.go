package canonicaldoc

import (
	"strings"
	"time"
)

const Schema = "tlaloc.canonical-document.r0"

type BBox struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

type Word struct {
	Text string `json:"text"`
	BBox BBox   `json:"bbox"`
}

type Region struct {
	ID           string `json:"id"`
	Address      string `json:"address"`
	CID          string `json:"cid_sha256"`
	Kind         string `json:"kind"`
	BBox         BBox   `json:"bbox"`
	ReadingOrder int    `json:"reading_order"`
	Text         string `json:"text,omitempty"`
	FontSize     int    `json:"font_size,omitempty"`
	Words        []Word `json:"words,omitempty"`
	SourceObject string `json:"source_object,omitempty"`
}

type Page struct {
	Number         int      `json:"number"`
	Address        string   `json:"address"`
	Width          float64  `json:"width"`
	Height         float64  `json:"height"`
	ExtractionMode string   `json:"extraction_mode"`
	Regions        []Region `json:"regions"`
	TextChars      int      `json:"text_chars"`
	ImageCount     int      `json:"image_count"`
}

type Document struct {
	Schema           string    `json:"schema"`
	DocumentID       string    `json:"document_id"`
	CarrierID        string    `json:"carrier_id"`
	SourceSHA256     string    `json:"source_sha256"`
	PageCount        int       `json:"page_count"`
	DigitalPages     int       `json:"digital_pages"`
	OCRPages         int       `json:"ocr_pages"`
	RegionCount      int       `json:"region_count"`
	FigureCount      int       `json:"figure_count"`
	CanonicalVersion string    `json:"canonical_version"`
	BuiltAt          time.Time `json:"built_at"`
	Pages            []string  `json:"pages"` // relative page IR paths
}

// CanonicalText is the deterministic text view of a canonical page. Geometry,
// figures and semantic annotations remain in Page; this byte stream exists for
// exact textual addressing and never replaces the layout IR.
func CanonicalText(p Page) []byte {
	var b strings.Builder
	for _, r := range p.Regions {
		if r.Text == "" || r.Kind == "figure" {
			continue
		}
		b.WriteString(r.Text)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
