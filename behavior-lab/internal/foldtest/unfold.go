package foldtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
)

// UnfoldToTempFile calls pdfmemory.Expand unchanged, writes the retrieved content
// to a temp file under the session's work directory, and returns the path plus the
// full Packet (budget/uncertainty/Merkle-verified metrics included for free).
// The unfold is deterministic and content-addressed, so the temp file is always
// reproducible from (storeDir, address) alone.
func UnfoldToTempFile(workDir, storeDir string, m pdfmemory.Manifest, address, fidelity string, maxTokens int) (path string, packet pdfmemory.Packet, err error) {
	// Ensure work directory exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", pdfmemory.Packet{}, fmt.Errorf("creating work dir: %w", err)
	}

	// Call pdfmemory.Expand to retrieve content
	packet, err = pdfmemory.Expand(storeDir, m, address, fidelity, maxTokens)
	if err != nil {
		return "", pdfmemory.Packet{}, fmt.Errorf("expanding address %q: %w", address, err)
	}

	// Write evidence content to temp file
	// Filename: sanitized address as base, with content hash suffix for reproducibility
	filename := sanitizeAddress(address)
	if fidelity != "" {
		filename = filename + "_" + fidelity
	}
	tempPath := filepath.Join(workDir, filename+".txt")

	var contentBuf strings.Builder
	for i, ev := range packet.Evidence {
		contentBuf.WriteString(fmt.Sprintf("--- Evidence %d ---\n", i))
		contentBuf.WriteString(fmt.Sprintf("Address: %s\n", ev.Address))
		if ev.CID != "" {
			contentBuf.WriteString(fmt.Sprintf("CID: %s\n", ev.CID))
		}
		contentBuf.WriteString("\n")
		contentBuf.WriteString(ev.Content)
		contentBuf.WriteString("\n\n")
	}

	if err := os.WriteFile(tempPath, []byte(contentBuf.String()), 0644); err != nil {
		return "", pdfmemory.Packet{}, fmt.Errorf("writing temp file: %w", err)
	}

	return tempPath, packet, nil
}

// sanitizeAddress converts an address like "doc/page-12/blocks/3" to a filesystem-safe name
func sanitizeAddress(address string) string {
	// Replace problematic characters with underscores
	return strings.NewReplacer(
		"/", "_",
		":", "_",
		" ", "_",
	).Replace(address)
}
