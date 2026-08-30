package canonicaldoc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type xmlDoc struct {
	Pages []xmlPage `xml:"page"`
}
type xmlPage struct {
	Number int        `xml:"number,attr"`
	Width  float64    `xml:"width,attr"`
	Height float64    `xml:"height,attr"`
	Fonts  []xmlFont  `xml:"fontspec"`
	Texts  []xmlText  `xml:"text"`
	Images []xmlImage `xml:"image"`
}
type xmlFont struct {
	ID     int    `xml:"id,attr"`
	Size   int    `xml:"size,attr"`
	Family string `xml:"family,attr"`
}
type xmlText struct {
	Top    int    `xml:"top,attr"`
	Left   int    `xml:"left,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Font   int    `xml:"font,attr"`
	Inner  string `xml:",innerxml"`
}
type xmlImage struct {
	Top    int    `xml:"top,attr"`
	Left   int    `xml:"left,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Src    string `xml:"src,attr"`
}

type BuildOptions struct {
	CarrierID      string
	DocumentID     string
	OCRMinChars    int
	OCRDPI         int
	SingleDocument bool
}

func BuildPDF(pdfPath, outDir string, opts BuildOptions) (Document, error) {
	if opts.CarrierID == "" {
		opts.CarrierID = "document"
	}
	if opts.DocumentID == "" {
		opts.DocumentID = "doc"
	}
	if opts.OCRMinChars <= 0 {
		opts.OCRMinChars = 24
	}
	if opts.OCRDPI <= 0 {
		opts.OCRDPI = 180
	}
	b, err := os.ReadFile(pdfPath)
	if err != nil {
		return Document{}, err
	}
	sourceSHA := hash(b)
	if err := os.MkdirAll(filepath.Join(outDir, "pages"), 0755); err != nil {
		return Document{}, err
	}
	tmp, err := os.MkdirTemp("", "tlaloc-canonical-*")
	if err != nil {
		return Document{}, err
	}
	defer os.RemoveAll(tmp)
	xmlPath := filepath.Join(tmp, "layout.xml")
	// pdftohtml gives us text geometry and image rectangles in one deterministic pass.
	cmd := exec.Command("pdftohtml", "-xml", "-hidden", "-nodrm", pdfPath, xmlPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return Document{}, fmt.Errorf("pdftohtml: %w: %s", err, strings.TrimSpace(string(out)))
	}
	xb, err := os.ReadFile(xmlPath)
	if err != nil {
		return Document{}, err
	}
	var xd xmlDoc
	if err := xml.Unmarshal(xb, &xd); err != nil {
		return Document{}, fmt.Errorf("parse pdftohtml xml: %w", err)
	}
	doc := Document{Schema: Schema, DocumentID: opts.DocumentID, CarrierID: opts.CarrierID, SourceSHA256: sourceSHA, PageCount: len(xd.Pages), CanonicalVersion: "r0", BuiltAt: time.Now().UTC()}
	for i := range xd.Pages {
		xp := xd.Pages[i]
		pageAddr := fmt.Sprintf("ohf://%s/docs/%s/pages/%06d", opts.CarrierID, opts.DocumentID, xp.Number)
		if opts.SingleDocument {
			pageAddr = fmt.Sprintf("ohf://%s/pages/%06d", opts.CarrierID, xp.Number)
		}
		p := Page{Number: xp.Number, Address: pageAddr, Width: xp.Width, Height: xp.Height, ExtractionMode: "digital"}
		order := 0
		fontMap := map[int]xmlFont{}
		for _, f := range xp.Fonts {
			fontMap[f.ID] = f
		}
		topCounts := map[int]int{}
		for _, xt := range xp.Texts {
			topCounts[xt.Top]++
		}
		for _, xt := range xp.Texts {
			txt := normalizeXMLText(xt.Inner)
			if strings.TrimSpace(txt) == "" {
				continue
			}
			order++
			f := fontMap[xt.Font]
			kind := classify(txt, xt.Height, xp.Height)
			family := strings.ToLower(f.Family)
			if strings.Contains(family, "cmtt") || strings.Contains(family, "mono") {
				kind = "code"
			} else if strings.Contains(family, "cmmi") || strings.Contains(family, "math") {
				kind = "equation"
			} else if topCounts[xt.Top] >= 3 && xt.Width < int(xp.Width*.55) {
				kind = "table_cell"
			}
			addr := fmt.Sprintf("%s/regions/text-%04d", p.Address, order)
			p.Regions = append(p.Regions, Region{ID: fmt.Sprintf("text-%04d", order), Address: addr, CID: hash([]byte(txt)), Kind: kind, BBox: BBox{X1: float64(xt.Left), Y1: float64(xt.Top), X2: float64(xt.Left + xt.Width), Y2: float64(xt.Top + xt.Height)}, ReadingOrder: order, Text: txt, FontSize: xt.Height})
			p.TextChars += utf8.RuneCountInString(txt)
		}
		for j, xi := range xp.Images {
			order++
			src := filepath.Join(tmp, xi.Src)
			cid := ""
			stableObject := ""
			if ib, e := os.ReadFile(src); e == nil {
				cid = hash(ib)
				ext := filepath.Ext(xi.Src)
				if ext == "" {
					ext = ".bin"
				}
				stableObject = filepath.ToSlash(filepath.Join("figures", cid+ext))
				_ = os.MkdirAll(filepath.Join(outDir, "figures"), 0755)
				if e := os.WriteFile(filepath.Join(outDir, filepath.FromSlash(stableObject)), ib, 0644); e != nil {
					return Document{}, e
				}
			}
			if cid == "" {
				cid = hash([]byte(fmt.Sprintf("%d:%d:%d:%d:%d", xp.Number, xi.Left, xi.Top, xi.Width, xi.Height)))
			}
			addr := fmt.Sprintf("%s/regions/figure-%04d", p.Address, j+1)
			p.Regions = append(p.Regions, Region{ID: fmt.Sprintf("figure-%04d", j+1), Address: addr, CID: cid, Kind: "figure", BBox: BBox{X1: float64(xi.Left), Y1: float64(xi.Top), X2: float64(xi.Left + xi.Width), Y2: float64(xi.Top + xi.Height)}, ReadingOrder: order, SourceObject: stableObject})
			p.ImageCount++
			doc.FigureCount++
		}
		if p.TextChars < opts.OCRMinChars {
			ocr, e := extractOCRFromPageImages(xp, opts, tmp, p.Address)
			if e == nil && ocr.TextChars > p.TextChars {
				p = ocr
				p.Number = xp.Number
				p.Width = xp.Width
				p.Height = xp.Height
				doc.OCRPages++
			} else {
				doc.DigitalPages++
			}
		} else {
			doc.DigitalPages++
		}
		sort.SliceStable(p.Regions, func(a, b int) bool { return p.Regions[a].ReadingOrder < p.Regions[b].ReadingOrder })
		doc.RegionCount += len(p.Regions)
		rel := filepath.ToSlash(filepath.Join("pages", fmt.Sprintf("%06d.json", p.Number)))
		if err := writeJSON(filepath.Join(outDir, filepath.FromSlash(rel)), p); err != nil {
			return Document{}, err
		}
		doc.Pages = append(doc.Pages, rel)
	}
	if err := writeJSON(filepath.Join(outDir, "document.json"), doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func LoadPage(canonicalDir string, n int) (Page, error) {
	var p Page
	b, err := os.ReadFile(filepath.Join(canonicalDir, "pages", fmt.Sprintf("%06d.json", n)))
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(b, &p)
	return p, err
}

func extractOCRFromPageImages(xp xmlPage, opts BuildOptions, tmp, pageAddr string) (Page, error) {
	if len(xp.Images) == 0 {
		return Page{}, fmt.Errorf("no raster image available for OCR")
	}
	best := xp.Images[0]
	bestArea := best.Width * best.Height
	for _, im := range xp.Images[1:] {
		if a := im.Width * im.Height; a > bestArea {
			best = im
			bestArea = a
		}
	}
	img := filepath.Join(tmp, best.Src)
	if _, err := os.Stat(img); err != nil {
		return Page{}, err
	}
	prefix := filepath.Join(tmp, fmt.Sprintf("ocr-%06d", xp.Number))
	out, err := exec.Command("tesseract", img, prefix, "hocr").CombinedOutput()
	if err != nil {
		return Page{}, fmt.Errorf("tesseract page %d: %w: %s", xp.Number, err, string(out))
	}
	hb, err := os.ReadFile(prefix + ".hocr")
	if err != nil {
		return Page{}, err
	}
	regions := parseHOCR(string(hb), pageAddr)
	p := Page{Number: xp.Number, Address: pageAddr, ExtractionMode: "ocr", Regions: regions}
	for _, r := range regions {
		p.TextChars += utf8.RuneCountInString(r.Text)
	}
	return p, nil
}

func parseHOCR(s, pageAddr string) []Region {
	// Deliberately simple deterministic parser: one region per OCR line.
	var out []Region
	pos := 0
	order := 0
	for {
		i := strings.Index(s[pos:], "ocr_line")
		if i < 0 {
			break
		}
		i += pos
		start := strings.LastIndex(s[:i], "<span")
		if start < 0 {
			pos = i + 8
			continue
		}
		end := strings.Index(s[i:], "</span>")
		if end < 0 {
			break
		}
		end += i + 7
		chunk := s[start:end]
		bbox := BBox{}
		if bi := strings.Index(chunk, "bbox "); bi >= 0 {
			fields := strings.Fields(chunk[bi+5:])
			if len(fields) >= 4 {
				x1, _ := strconv.ParseFloat(strings.Trim(fields[0], ";\"'"), 64)
				y1, _ := strconv.ParseFloat(strings.Trim(fields[1], ";\"'"), 64)
				x2, _ := strconv.ParseFloat(strings.Trim(fields[2], ";\"'"), 64)
				y2, _ := strconv.ParseFloat(strings.Trim(fields[3], ";\"'"), 64)
				bbox = BBox{x1, y1, x2, y2}
			}
		}
		txt := stripTags(chunk)
		if strings.TrimSpace(txt) != "" {
			order++
			out = append(out, Region{ID: fmt.Sprintf("ocr-%04d", order), Address: fmt.Sprintf("%s/regions/ocr-%04d", pageAddr, order), CID: hash([]byte(txt)), Kind: "ocr_text", BBox: bbox, ReadingOrder: order, Text: txt})
		}
		pos = end
	}
	return out
}

func normalizeXMLText(s string) string { return strings.TrimSpace(html.UnescapeString(stripTags(s))) }
func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			continue
		}
		if !in {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
func classify(text string, height int, pageH float64) string {
	t := strings.TrimSpace(text)
	upper := strings.ToUpper(t) == t && len(t) > 3
	if height >= 20 || upper {
		return "heading"
	}
	if strings.HasPrefix(t, "•") || strings.HasPrefix(t, "-") {
		return "list_item"
	}
	if strings.Contains(t, "=") && len(t) < 160 {
		return "equation_or_code"
	}
	if len(t) < 120 && height >= 16 {
		return "subheading"
	}
	return "text_line"
}
func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0644)
}
