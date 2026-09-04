package tonalt1arms

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
	"tlaloc.local/behaviorlab/internal/parrotlab"
	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// PageStore wraps PDF rendering and caches page PNGs and layouts since many
// operands reference the same pages within and across workflows.
type PageStore struct {
	provider parrotlab.PageProvider
	storeDir string
	cache    map[int][]byte
	layouts  map[int]*canonicaldoc.Page
	manifest pdfmemory.Manifest
}

// NewPageStore creates a new page store from a PDF and its associated store directory.
func NewPageStore(storeDir, pdfPath string) (*PageStore, error) {
	provider, err := parrotlab.NewPDFMemoryProvider(storeDir, pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF provider: %w", err)
	}

	// Load the manifest to access layout paths
	manifest, _, err := pdfmemory.Load(storeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load pdfmemory manifest: %w", err)
	}

	ps := &PageStore{
		provider: provider,
		storeDir: storeDir,
		cache:    make(map[int][]byte),
		layouts:  make(map[int]*canonicaldoc.Page),
		manifest: manifest,
	}

	return ps, nil
}

// PagePNG returns the PNG bytes for a page, using the cache if available.
func (ps *PageStore) PagePNG(page int) ([]byte, error) {
	if cached, ok := ps.cache[page]; ok {
		return cached, nil
	}

	png, err := ps.provider.RenderPNG(page)
	if err != nil {
		return nil, fmt.Errorf("failed to render page %d: %w", page, err)
	}

	ps.cache[page] = png
	return png, nil
}

// Region retrieves a region's geometry from the store layout.
// The region_id must be present in the page's layout JSON.
func (ps *PageStore) Region(page int, regionID string) (*canonicaldoc.Region, *canonicaldoc.Page, error) {
	pageLayout, err := ps.pageLayout(page)
	if err != nil {
		return nil, nil, err
	}

	for i := range pageLayout.Regions {
		if pageLayout.Regions[i].ID == regionID {
			return &pageLayout.Regions[i], pageLayout, nil
		}
	}

	return nil, nil, fmt.Errorf("region %q not found on page %d", regionID, page)
}

// pageLayout loads and caches a page's geometry layout from the pdfmemory store.
func (ps *PageStore) pageLayout(pageNum int) (*canonicaldoc.Page, error) {
	if layout, ok := ps.layouts[pageNum]; ok {
		return layout, nil
	}

	// Find the page's layout path in the manifest
	var layoutPath string
	if pageNum < len(ps.manifest.Pages) {
		layoutPath = ps.manifest.Pages[pageNum].LayoutPath
	}
	if layoutPath == "" {
		return nil, fmt.Errorf("no layout path for page %d in manifest", pageNum)
	}

	// Resolve the path (may be relative to storeDir)
	if !filepath.IsAbs(layoutPath) {
		layoutPath = filepath.Join(ps.storeDir, layoutPath)
	}

	// Load and parse the layout JSON
	data, err := os.ReadFile(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout file %s: %w", layoutPath, err)
	}

	var pageLayout canonicaldoc.Page
	if err := json.Unmarshal(data, &pageLayout); err != nil {
		return nil, fmt.Errorf("failed to parse layout file %s: %w", layoutPath, err)
	}

	ps.layouts[pageNum] = &pageLayout
	return &pageLayout, nil
}
