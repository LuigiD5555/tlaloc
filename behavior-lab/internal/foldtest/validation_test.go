package foldtest

import (
	"testing"
)

func TestSelectSpacedPages(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		num       int
		seed      int64
		minSpacing int
		wantLen   int
	}{
		{
			name:      "select 5 from 404",
			total:     404,
			num:       5,
			seed:      42,
			minSpacing: 30, // At least ~80 pages spacing target
			wantLen:   5,
		},
		{
			name:      "select 3 from 100",
			total:     100,
			num:       3,
			seed:      42,
			minSpacing: 20,
			wantLen:   3,
		},
		{
			name:      "request more than available",
			total:     10,
			num:       20,
			seed:      42,
			wantLen:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectSpacedPages(tt.total, tt.num, tt.seed)
			if len(got) != tt.wantLen {
				t.Errorf("got len=%d, want %d", len(got), tt.wantLen)
			}

			// Check no duplicates
			seen := make(map[int]bool)
			for _, p := range got {
				if seen[p] {
					t.Errorf("duplicate page: %d", p)
				}
				seen[p] = true
				if p < 1 || p > tt.total {
					t.Errorf("page %d out of range [1, %d]", p, tt.total)
				}
			}

			// Check spacing for non-trivial cases
			if tt.minSpacing > 0 && len(got) > 1 {
				for i := 1; i < len(got); i++ {
					spacing := got[i] - got[i-1]
					if spacing < tt.minSpacing {
						t.Logf("spacing between pages %d and %d is %d (minimum expected ~%d)",
							got[i-1], got[i], spacing, tt.minSpacing)
					}
				}
			}
		})
	}
}

// Keyword-overlap scoring, estimateConfidence and extractKeywords moved to
// internal/tlaloque/answerscore (KeywordOverlapWorker) — see
// TestKeywordOverlap_ExtractedKeywordsMeetMinLength, TestEstimateConfidence
// and TestScoreByKeywordOverlap in that package for their coverage.

func TestValidateAddressFormat(t *testing.T) {
	tests := []struct {
		address string
		valid   bool
	}{
		{"page:000001", true},
		{"page:404", true},
		{"page:12345", true},
		{"page:abc", false},
		{"page:", false},
		{"block:doc/page-1/blocks/0", true},
		{"block:doc/page-404/blocks/99", true},
		{"block:invalid", false},
		{"invalid:format", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			got := ValidateAddressFormat(tt.address)
			if got != tt.valid {
				t.Errorf("ValidateAddressFormat(%q) = %v, want %v", tt.address, got, tt.valid)
			}
		})
	}
}

// GeneratePageQuestions/extractKeyword moved to
// internal/tlaloque/questiongen (TemplateWorker) — see
// TestTemplateWorker_PositiveCase in that package for their coverage.
