package foldtest

import (
	"strings"
	"testing"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// ExtractPageContent must resolve the page's address from the manifest
// itself (manifest.Pages[i].Address) rather than guessing a hardcoded
// carrier-specific string format — it must work for any carrier_id and for
// the multi-document address shape, not just the single-document
// "ohf://fold-bench/pages/NNNNNN" pattern the old code assumed.
func TestExtractPageContent_ResolvesAddressFromManifest(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{name: "single-document carrier, non-default carrier_id", address: "ohf://some-other-carrier/pages/000005"},
		{name: "multi-document carrier", address: "ohf://some-other-carrier/docs/doc1/pages/000005"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := pdfmemory.Manifest{
				Pages: []pdfmemory.PageRef{
					{Number: 4, Address: "ohf://some-other-carrier/pages/000004"},
					{Number: 5, Address: tt.address},
					{Number: 6, Address: "ohf://some-other-carrier/pages/000006"},
				},
			}

			_, err := ExtractPageContent("/nonexistent-store", manifest, 5)
			// pdfmemory.Expand will fail because the store directory does not
			// exist, but the failure must happen *after* successfully
			// resolving page 5's address — not with "page not found".
			if err == nil {
				t.Fatal("expected an error since the store directory does not exist")
			}
			if strings.Contains(err.Error(), "not found in manifest") {
				t.Errorf("expected address resolution to succeed for %q, got: %v", tt.address, err)
			}
		})
	}
}

// A page number absent from the manifest must fail clearly, not panic or
// silently fall through to a wrong page.
func TestExtractPageContent_PageNotFound(t *testing.T) {
	manifest := pdfmemory.Manifest{
		Pages: []pdfmemory.PageRef{
			{Number: 1, Address: "ohf://carrier/pages/000001"},
		},
	}

	_, err := ExtractPageContent("/nonexistent-store", manifest, 99)
	if err == nil {
		t.Fatal("expected an error for a page number absent from the manifest")
	}
	if !strings.Contains(err.Error(), "not found in manifest") {
		t.Errorf("expected a 'not found in manifest' error, got: %v", err)
	}
}
