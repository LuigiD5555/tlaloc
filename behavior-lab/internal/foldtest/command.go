package foldtest

import (
	"fmt"
	"regexp"
	"strings"
)

// Command represents a parsed UNFOLD or GROUP command
type Command struct {
	Op       string // "UNFOLD" or "GROUP"
	Arg      string // e.g., "page:12" or "term:x depth:2"
	Address  string // extracted address for UNFOLD
	Fidelity string // optional fidelity level for UNFOLD
	Term     string // for GROUP
	Depth    int    // for GROUP
}

// ParseCommand parses a line attempting to extract UNFOLD or GROUP commands.
// Valid syntax:
//   UNFOLD page:N
//   UNFOLD block:doc/page-N/blocks/M
//   UNFOLD doc:document-id
//   GROUP term:<X> depth:<N>
//   UNFOLD <address>[:fidelity]
func ParseCommand(line string) (cmd *Command, ok bool) {
	line = strings.TrimSpace(line)

	// Try UNFOLD first
	if strings.HasPrefix(line, "UNFOLD ") {
		unfoldPart := strings.TrimPrefix(line, "UNFOLD ")
		unfoldPart = strings.TrimSpace(unfoldPart)

		if unfoldPart == "" {
			return nil, false
		}

		// Handle fidelity suffix: "address:fidelity"
		parts := strings.Split(unfoldPart, ":")
		if len(parts) >= 2 {
			// Check if last part is a known fidelity level
			lastPart := parts[len(parts)-1]
			if lastPart == "low" || lastPart == "high" {
				cmd := &Command{
					Op:       "UNFOLD",
					Arg:      unfoldPart,
					Fidelity: lastPart,
					Address:  strings.Join(parts[:len(parts)-1], ":"),
				}
				return cmd, true
			}
		}

		// No fidelity suffix
		cmd := &Command{
			Op:      "UNFOLD",
			Arg:     unfoldPart,
			Address: unfoldPart,
		}
		return cmd, true
	}

	// Try GROUP next
	if strings.HasPrefix(line, "GROUP ") {
		groupPart := strings.TrimPrefix(line, "GROUP ")
		groupPart = strings.TrimSpace(groupPart)

		// Parse "term:<X> depth:<N>"
		term, depth, ok := parseGroupArgs(groupPart)
		if !ok {
			return nil, false
		}

		cmd := &Command{
			Op:    "GROUP",
			Arg:   groupPart,
			Term:  term,
			Depth: depth,
		}
		return cmd, true
	}

	return nil, false
}

func parseGroupArgs(s string) (term string, depth int, ok bool) {
	// Expected format: term:<X> depth:<N>
	termRegex := regexp.MustCompile(`term:(\S+)`)
	depthRegex := regexp.MustCompile(`depth:(\d+)`)

	termMatch := termRegex.FindStringSubmatch(s)
	if len(termMatch) < 2 {
		return "", 0, false
	}
	term = termMatch[1]

	depthMatch := depthRegex.FindStringSubmatch(s)
	if len(depthMatch) < 2 {
		return "", 0, false
	}

	var d int
	_, err := fmt.Sscanf(depthMatch[1], "%d", &d)
	if err != nil {
		return "", 0, false
	}

	return term, d, true
}
