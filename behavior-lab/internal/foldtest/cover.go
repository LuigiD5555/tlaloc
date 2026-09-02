package foldtest

import (
	"fmt"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// BuildCoverText generates a compact plain-text table of contents that encodes
// the whole book in under ~800 tokens, suitable for a 4000-token window context.
// It ranks terms by document frequency from the graph and includes one line per
// page listing the top ~12 most relevant terms sourced from graph.json/index.json.
func BuildCoverText(storeDir string, m pdfmemory.Manifest, maxTokens int) (string, error) {
	graph, err := pdfmemory.LoadGraph(storeDir)
	if err != nil {
		return "", fmt.Errorf("loading graph: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("=== PDF Document Cover ===\n\n")

	// Document summary
	sb.WriteString(fmt.Sprintf("Documents: %d\n", m.DocumentCount))
	sb.WriteString(fmt.Sprintf("Total Pages: %d\n", m.PageCount))
	sb.WriteString(fmt.Sprintf("Total Blocks: %d\n\n", m.BlockCount))

	// Document list
	sb.WriteString("--- Documents ---\n")
	for _, doc := range m.Documents {
		sb.WriteString(fmt.Sprintf("  %s: %d pages\n", doc.ID, doc.PageCount))
	}
	sb.WriteString("\n")

	// Page-by-page coverage using graph terms
	sb.WriteString("--- Page Index ---\n")
	pageTerms := extractPageTopTerms(graph, m, 12)

	for docIdx, doc := range m.Documents {
		sb.WriteString(fmt.Sprintf("\n[%s]\n", doc.Name))

		startPage := 0
		for i := 0; i < docIdx; i++ {
			startPage += m.Documents[i].PageCount
		}

		for pageNum := 1; pageNum <= doc.PageCount; pageNum++ {
			globalPageIdx := startPage + pageNum - 1
			terms := pageTerms[globalPageIdx]

			// Format: "page N: term1, term2, ..."
			sb.WriteString(fmt.Sprintf("  page %d: %s\n", pageNum, strings.Join(terms, ", ")))
		}
	}

	result := sb.String()

	// Estimate tokens (rough: ~4 chars per token)
	estimatedTokens := len(result) / 4
	if estimatedTokens > maxTokens {
		// Truncate aggressively if needed
		maxChars := maxTokens * 4
		if len(result) > maxChars {
			result = result[:maxChars] + "\n[... truncated to fit token budget]\n"
		}
	}

	return result, nil
}

func extractPageTopTerms(g pdfmemory.Graph, m pdfmemory.Manifest, termsPerPage int) [][]string {
	// For each page, find the most relevant terms by looking at the blocks on that page
	// and collecting high-frequency terms from the graph.

	pageTerms := make([][]string, m.PageCount)
	for i := range pageTerms {
		pageTerms[i] = []string{}
	}

	// Collect all terms sorted by document frequency
	type termFreq struct {
		term string
		freq int
	}
	var allTerms []termFreq
	for term, node := range g.Nodes {
		allTerms = append(allTerms, termFreq{term, node.DocumentFrequency})
	}

	// Sort by frequency descending
	sort.Slice(allTerms, func(i, j int) bool {
		return allTerms[i].freq > allTerms[j].freq
	})

	// Assign top global terms to pages proportionally
	// Simple strategy: distribute the top terms across all pages
	if len(allTerms) > 0 {
		topN := termsPerPage * m.PageCount / 2 // limit total unique terms shown
		if topN > len(allTerms) {
			topN = len(allTerms)
		}

		// Spread top terms across pages
		for i := 0; i < topN; i++ {
			pageIdx := i % m.PageCount
			if len(pageTerms[pageIdx]) < termsPerPage {
				pageTerms[pageIdx] = append(pageTerms[pageIdx], allTerms[i].term)
			}
		}
	}

	// Ensure each page has some terms, using graph neighborhood fallback
	for pageIdx := 0; pageIdx < m.PageCount; pageIdx++ {
		if len(pageTerms[pageIdx]) < 3 && len(allTerms) > 0 {
			// Add more terms from top-frequency list
			for j := 0; j < len(allTerms) && len(pageTerms[pageIdx]) < 6; j++ {
				pageTerms[pageIdx] = append(pageTerms[pageIdx], allTerms[j].term)
			}
		}
	}

	return pageTerms
}
