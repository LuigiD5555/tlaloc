package pdfmemory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicaldoc"
)

type RegionResult struct {
	Page   PageRef
	Region canonicaldoc.Region
	Bytes  []byte
	Binary bool
}

func ReadRegion(storeDir string, m Manifest, address string) (RegionResult, error) {
	i := strings.Index(address, "/regions/")
	if i < 0 {
		return RegionResult{}, fmt.Errorf("not a region address")
	}
	pageAddr := address[:i]
	doc, n, kind, _, err := ParseAddress(m.CarrierID, pageAddr)
	if err != nil {
		return RegionResult{}, err
	}
	if kind != "page" {
		return RegionResult{}, fmt.Errorf("region parent is not page")
	}
	doc = resolveDocAlias(m, doc)

	var pref PageRef
	found := false
	for _, p := range m.Pages {
		if p.DocID == doc && p.Number == n {
			pref = p
			found = true
			break
		}
	}
	if !found {
		return RegionResult{}, fmt.Errorf("page not found")
	}
	b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(pref.LayoutPath)))
	if err != nil {
		return RegionResult{}, err
	}
	if hash(b) != pref.LayoutCID {
		return RegionResult{}, fmt.Errorf("layout CID mismatch")
	}
	var page canonicaldoc.Page
	if err := json.Unmarshal(b, &page); err != nil {
		return RegionResult{}, err
	}
	for _, r := range page.Regions {
		if r.Address != address {
			continue
		}
		if r.Kind == "figure" && r.SourceObject != "" {
			figurePath := filepath.Join(storeDir, "canonical", doc, filepath.FromSlash(r.SourceObject))
			fb, ferr := os.ReadFile(figurePath)
			if ferr == nil && hash(fb) == r.CID {
				return RegionResult{Page: pref, Region: r, Bytes: fb, Binary: true}, nil
			}
		}
		content := []byte(r.Text)
		if hash(content) != r.CID {
			return RegionResult{}, fmt.Errorf("region CID mismatch")
		}
		return RegionResult{Page: pref, Region: r, Bytes: content}, nil
	}
	return RegionResult{}, fmt.Errorf("region address not found")
}
