package parrotlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/foldtest"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

var (
	bareNumberLine = regexp.MustCompile(`^\d{1,4}$`)
	// A running header: an ALL-CAPS chapter/section title with the printed
	// page number embedded, e.g. "THE ANALYSIS OF VISUAL MOTION 51".
	runningHeader = regexp.MustCompile(`([A-Z]{3}[A-Z ’'&-]{3,58}?)\s+\d{1,4}(\s|$)`)
)

// A page-start running header: "<Section Title> <printed page number> <body…>"
// where the number is flanked by a letter on the left and a lowercase word on
// the right. Only stripped near the top of the page.
var startHeaderNumber = regexp.MustCompile(`([A-Za-z][A-Za-z'’-]+)\s+\d{2,3}\s+([a-z])`)

// cleanPageText drops page-number lines, running headers and blank lines
// from the deterministically extracted page text. It does not paraphrase —
// every remaining fragment is verbatim from pdfmemory.Expand.
func cleanPageText(raw string) string {
	lines := strings.Split(raw, "\n")
	kept := make([]string, 0, len(lines))
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if bareNumberLine.MatchString(trimmed) && (index <= 1 || index >= len(lines)-2) {
			continue
		}
		trimmed = runningHeader.ReplaceAllString(trimmed, "$1$2")
		kept = append(kept, strings.TrimSpace(trimmed))
	}
	joined := strings.Join(kept, "\n")
	// Strip the page-start running-header page number from the first line.
	if cut := strings.IndexByte(joined, '\n'); cut > 0 && cut < 200 {
		joined = startHeaderNumber.ReplaceAllString(joined[:cut], "$1 $2") + joined[cut:]
	} else if len(joined) < 200 {
		joined = startHeaderNumber.ReplaceAllString(joined, "$1 $2")
	}
	return joined
}

// SourceRegion is one layout region: its kind ("heading", "subheading",
// "text_line", "list_item", …), verbatim text, reading order and font size.
type SourceRegion struct {
	Kind         string
	Text         string
	ReadingOrder int
	FontSize     int
}

// SourcePage is one page of the P0 document: its number, its Origami address,
// the content hash, the deterministically extracted text, and the layout
// regions (used for heading-aware question generation).
type SourcePage struct {
	Number  int
	Address string
	CID     string
	Text    string
	Regions []SourceRegion
}

// PageProvider is the P0 backend. The production implementation wraps the
// existing pdfmemory store + poppler rasteriser; tests inject a fake.
type PageProvider interface {
	SourceID() string
	Pages() ([]SourcePage, error)
	// RenderPNG rasterises one page. It returns an error (not a blank
	// image) when no rasteriser or source PDF is available.
	RenderPNG(number int) ([]byte, error)
}

// pdfMemoryProvider reads pages from a built pdfmemory store and rasterises
// them from the store's own source PDF with pdftoppm. It invents no new
// extraction path — Pages() is exactly foldtest.ExtractPageContent.
type pdfMemoryProvider struct {
	storeDir string
	pdfPath  string
	manifest pdfmemory.Manifest
}

// NewPDFMemoryProvider loads a pdfmemory store. pdfPath may be empty, in
// which case the store's recorded source object is used for rasterising.
func NewPDFMemoryProvider(storeDir, pdfPath string) (PageProvider, error) {
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return nil, fmt.Errorf("load pdfmemory store %s: %w", storeDir, err)
	}
	if pdfPath == "" {
		pdfPath = resolveStorePDF(storeDir, manifest)
	}
	return &pdfMemoryProvider{storeDir: storeDir, pdfPath: pdfPath, manifest: manifest}, nil
}

func resolveStorePDF(storeDir string, manifest pdfmemory.Manifest) string {
	if manifest.SourcePath != "" {
		candidate := manifest.SourcePath
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(storeDir, candidate)
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	for _, doc := range manifest.Documents {
		candidate := doc.SourcePath
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(storeDir, candidate)
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (provider *pdfMemoryProvider) SourceID() string {
	if provider.manifest.SourceName != "" {
		return provider.manifest.SourceName
	}
	return provider.manifest.CarrierID
}

func (provider *pdfMemoryProvider) Pages() ([]SourcePage, error) {
	pages := make([]SourcePage, 0, len(provider.manifest.Pages))
	for _, ref := range provider.manifest.Pages {
		// Read the full page text object directly. foldtest.ExtractPageContent
		// (via pdfmemory.Expand) returns only the first block of a multi-block
		// page — it truncates most content pages to a few hundred chars.
		text, err := provider.pageText(ref)
		if err != nil {
			return nil, fmt.Errorf("read page %d text: %w", ref.Number, err)
		}
		pages = append(pages, SourcePage{
			Number:  ref.Number,
			Address: ref.Address,
			CID:     ref.CID,
			Text:    cleanPageText(text),
			Regions: provider.loadRegions(ref.LayoutPath),
		})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })
	return pages, nil
}

func (provider *pdfMemoryProvider) pageText(ref pdfmemory.PageRef) (string, error) {
	if ref.Path != "" {
		if raw, err := os.ReadFile(filepath.Join(provider.storeDir, filepath.FromSlash(ref.Path))); err == nil {
			return string(raw), nil
		}
	}
	return foldtest.ExtractPageContent(provider.storeDir, provider.manifest, ref.Number)
}

func (provider *pdfMemoryProvider) loadRegions(layoutPath string) []SourceRegion {
	if layoutPath == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(provider.storeDir, filepath.FromSlash(layoutPath)))
	if err != nil {
		return nil
	}
	var page canonicaldoc.Page
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil
	}
	regions := make([]SourceRegion, 0, len(page.Regions))
	for _, region := range page.Regions {
		regions = append(regions, SourceRegion{
			Kind:         region.Kind,
			Text:         strings.TrimSpace(region.Text),
			ReadingOrder: region.ReadingOrder,
			FontSize:     region.FontSize,
		})
	}
	return regions
}

func (provider *pdfMemoryProvider) RenderPNG(number int) ([]byte, error) {
	if provider.pdfPath == "" {
		return nil, fmt.Errorf("no source PDF for rasterising (store %s); pass --pdf", provider.storeDir)
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil, fmt.Errorf("pdftoppm not on PATH: %w", err)
	}
	dir, err := os.MkdirTemp("", "parrotlab-raster-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	prefix := filepath.Join(dir, "page")
	page := fmt.Sprintf("%d", number)
	cmd := exec.Command("pdftoppm", "-png", "-r", "150", "-f", page, "-l", page, "-singlefile", provider.pdfPath, prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm page %d: %w (%s)", number, err, strings.TrimSpace(stderr.String()))
	}
	return os.ReadFile(prefix + ".png")
}
