package parrotlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// P0 audit (SPEC §4 close-out gate): a per-question table + a summary the
// human signs off before freezing. Nothing is frozen until every column is
// green and the categories are balanced.

// HumanVerdict is one optional human-review entry
// (datasets/end-to-end.human-review.json: { "<base_id>": {"verdict":"PASS"} }).
type HumanVerdict struct {
	Verdict string `json:"verdict"` // PASS | REJECT
	Note    string `json:"note,omitempty"`
}

type p0AuditRow struct {
	BaseID      string
	Family      string
	Category    string
	Pages       []int
	HasEvidence bool
	HasAddress  bool
	HasText     bool
	HasImage    bool
	Validation  string
	Human       string
}

// WriteP0Audit builds datasets/P0_AUDIT.md from the authored end-to-end
// dataset. It returns the path and whether every gate is green.
func WriteP0Audit(exp *Experiment) (string, bool, error) {
	datasetPath, err := exp.StageDataset(StageEndToEnd)
	if err != nil {
		return "", false, err
	}
	cases, err := LoadCases(datasetPath)
	if err != nil {
		return "", false, err
	}

	human := map[string]HumanVerdict{}
	if raw, readErr := os.ReadFile(filepath.Join(filepath.Dir(datasetPath), "end-to-end.human-review.json")); readErr == nil {
		_ = json.Unmarshal(raw, &human)
	}

	problemByCase := map[string]string{}
	for _, problem := range append(Validate(cases), ValidateEndToEnd(cases)...) {
		text := problem.Error()
		if colon := strings.Index(text, ": "); colon > 0 {
			problemByCase[text[:colon]] = text[colon+2:]
		}
	}

	byBase := map[string]*p0AuditRow{}
	var order []string
	for _, item := range cases {
		row := byBase[item.BaseID]
		if row == nil {
			row = &p0AuditRow{BaseID: item.BaseID, Family: item.TaskFamily, Category: p0CategoryOf(item.BaseID), Pages: item.PageRefs, Human: "—"}
			byBase[item.BaseID] = row
			order = append(order, item.BaseID)
		}
		if item.Variant == "text" {
			row.HasText = true
			row.HasEvidence = strings.TrimSpace(item.EvidenceText) != "" || len(item.RequiredFacts) > 0
		}
		if item.Variant == "image" {
			row.HasImage = true
		}
		if len(item.GroundTruthAddresses) > 0 && looksLikeAddress(item.GroundTruthAddresses[0]) {
			row.HasAddress = true
		}
		if problem, bad := problemByCase[item.CaseID]; bad {
			row.Validation = "FAIL: " + problem
		}
		if verdict, ok := human[item.BaseID]; ok {
			row.Human = verdict.Verdict
		}
	}
	sort.Strings(order)

	var doc bytes.Buffer
	fmt.Fprintf(&doc, "# P0_AUDIT — %s\n\n", exp.Manifest.ExperimentID)
	doc.WriteString("| base_id | family | pages | evidence | address | text | image | validation | human |\n")
	doc.WriteString("|---|---|---|:-:|:-:|:-:|:-:|---|---|\n")

	counts := map[string]int{}
	failures, missingEvidence, missingAddress, humanRejected := 0, 0, 0, 0
	for _, baseID := range order {
		row := byBase[baseID]
		if row.Validation == "" {
			row.Validation = "PASS"
		}
		counts[row.Category]++
		if strings.HasPrefix(row.Validation, "FAIL") {
			failures++
		}
		if !row.HasEvidence {
			missingEvidence++
		}
		if !row.HasAddress {
			missingAddress++
		}
		if row.Human == "REJECT" {
			humanRejected++
		}
		fmt.Fprintf(&doc, "| %s | %s | %v | %s | %s | %s | %s | %s | %s |\n",
			row.BaseID, row.Family, row.Pages,
			check(row.HasEvidence), check(row.HasAddress), check(row.HasText), check(row.HasImage),
			row.Validation, row.Human)
	}

	base := len(order)
	fmt.Fprintf(&doc, "\n## Summary\n\n```\nBASE QUESTIONS   %3d\nTEXT VARIANTS    %3d\nIMAGE VARIANTS   %3d\nTOTAL RECORDS    %3d\n\n",
		base, countVariant(cases, "text"), countVariant(cases, "image"), len(cases))
	for _, category := range P0Categories {
		fmt.Fprintf(&doc, "%-10s %3d\n", category, counts[category])
	}
	fmt.Fprintf(&doc, "\nvalidation failures  %3d\nmissing evidence     %3d\nmissing address      %3d\nhuman rejected       %3d\n```\n\n",
		failures, missingEvidence, missingAddress, humanRejected)

	green := failures == 0 && missingEvidence == 0 && missingAddress == 0 && humanRejected == 0
	balanced := true
	for _, category := range P0Categories {
		if counts[category] != 6 {
			balanced = false
		}
	}
	ready := green && balanced && base == 30
	if ready {
		doc.WriteString("**GATE: GREEN** — ready for `freeze --scope global` + `freeze --scope stage --stage end_to_end`.\n")
	} else {
		doc.WriteString("**GATE: NOT GREEN** — do not freeze. ")
		if base != 30 {
			fmt.Fprintf(&doc, "Need 30 base questions, have %d. ", base)
		}
		if !balanced {
			doc.WriteString("Categories are not 6/6/6/6/6. ")
		}
		if !green {
			doc.WriteString("Fix the failing rows above. ")
		}
		doc.WriteString("Never force questions to hit a quota — if a category cannot be filled naturally, change the source.\n")
	}

	auditPath := filepath.Join(filepath.Dir(datasetPath), "P0_AUDIT.md")
	if err := os.WriteFile(auditPath, doc.Bytes(), 0o644); err != nil {
		return "", false, err
	}
	return auditPath, ready, nil
}

func check(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func countVariant(cases []Case, variant string) int {
	count := 0
	for _, item := range cases {
		if item.Variant == variant {
			count++
		}
	}
	return count
}
