package pdfmemory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/canonicalstate"
)

func LoadCanonicalState(storeDir string, m Manifest) (canonicalstate.State, error) {
	var s canonicalstate.State
	if m.CanonicalStatePath == "" {
		return s, os.ErrNotExist
	}
	b, err := os.ReadFile(filepath.Join(storeDir, filepath.FromSlash(m.CanonicalStatePath)))
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	return s, err
}

func matchCanonicalClaims(state canonicalstate.State, terms []string, limit int) []canonicalstate.CanonicalClaim {
	if limit <= 0 {
		limit = 8
	}
	type row struct {
		c     canonicalstate.CanonicalClaim
		score int
	}
	var rows []row
	for _, c := range state.Claims {
		hay := strings.ToLower(c.Claim.Subject + " " + c.Claim.Predicate + " " + c.Claim.Object)
		score := 0
		for _, t := range terms {
			if strings.Contains(hay, strings.ToLower(t)) {
				score++
			}
		}
		if score > 0 {
			rows = append(rows, row{c, score})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score == rows[j].score {
			return rows[i].c.ID < rows[j].c.ID
		}
		return rows[i].score > rows[j].score
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]canonicalstate.CanonicalClaim, len(rows))
	for i, r := range rows {
		out[i] = r.c
	}
	return out
}
