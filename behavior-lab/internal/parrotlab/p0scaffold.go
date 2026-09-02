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

// The authoring scaffold: for each selected P0 page it writes the verbatim
// evidence a human needs to hand-write the entity / factual / numeric /
// synthesis questions the deterministic generators can't produce from a
// catalog-structured source. It never invents question text.

// ScaffoldReport summarises a scaffold write.
type ScaffoldReport struct {
	ScaffoldFile string `json:"scaffold_file"`
	DraftFile    string `json:"draft_file"`
	Pages        int    `json:"pages"`
	DraftRows    int    `json:"draft_rows"`
}

func writeAuthoringScaffold(datasetDir string, provider PageProvider, selected []SourcePage, byCategory map[string][]p0Candidate) (ScaffoldReport, error) {
	report := ScaffoldReport{Pages: len(selected)}
	imageDir := filepath.Join(datasetDir, "end-to-end", "scaffold-images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return report, err
	}

	var doc bytes.Buffer
	fmt.Fprintf(&doc, "# P0 authoring scaffold — %s\n\n", provider.SourceID())
	doc.WriteString("Hand-write the remaining questions into `end-to-end.jsonl` (start from\n")
	doc.WriteString("`end-to-end.draft.jsonl`, which holds the auto-generated ones). Each question\n")
	doc.WriteString("must be answerable **only** from the evidence quoted below. Schema:\n")
	doc.WriteString("`datasets/SCHEMA.md`. Then: `validate --stage end_to_end` → `freeze --scope stage`.\n\n")
	doc.WriteString("Target: 30 questions over these 10 pages — 6 each of locate / entity / factual / numeric / synthesis.\n\n")

	autoByPage := map[int][]p0Candidate{}
	for _, list := range byCategory {
		for _, candidate := range list {
			autoByPage[candidate.Page.Number] = append(autoByPage[candidate.Page.Number], candidate)
		}
	}

	for _, page := range selected {
		model := buildHeadingModel(page)
		imageRel := filepath.ToSlash(filepath.Join("end-to-end", "scaffold-images", fmt.Sprintf("p%d.png", page.Number)))
		if png, err := provider.RenderPNG(page.Number); err == nil {
			_ = os.WriteFile(filepath.Join(datasetDir, filepath.FromSlash(imageRel)), png, 0o644)
		}

		fmt.Fprintf(&doc, "## Page %d — %q\n\n", page.Number, model.PatternName)
		fmt.Fprintf(&doc, "- address: `%s`  ·  page cid: `%s`\n", page.Address, page.CID)
		fmt.Fprintf(&doc, "- rendered page: `%s`\n", imageRel)
		if model.Motivation != "" {
			fmt.Fprintf(&doc, "- **Motivation** (verbatim): %s\n", truncate(model.Motivation, 500))
		}
		if model.Mechanics != "" {
			fmt.Fprintf(&doc, "- **Mechanics** (verbatim): %s\n", truncate(model.Mechanics, 500))
		}
		if numbers := numberContexts(proseText(page), page.Number); len(numbers) > 0 {
			doc.WriteString("- numbers in the prose (skip code):\n")
			for _, entry := range numbers {
				fmt.Fprintf(&doc, "    - `%s` — %s\n", entry.number, entry.sentence)
			}
		}
		if terms := distinctiveTerms(proseText(page)); len(terms) > 0 {
			fmt.Fprintf(&doc, "- candidate noun phrases: %s\n", strings.Join(quoteAll(terms), ", "))
		}
		if drafts := autoByPage[page.Number]; len(drafts) > 0 {
			doc.WriteString("- auto-generated for this page:\n")
			for _, candidate := range drafts {
				fmt.Fprintf(&doc, "    - [%s] %s → %q\n", candidate.Category, truncate(candidate.Question, 120), candidate.Answer)
			}
		}
		pageText := page.Text
		if len(pageText) > 2600 {
			pageText = pageText[:2600] + "\n…"
		}
		fmt.Fprintf(&doc, "\n<details><summary>full page text (verbatim — the authoring evidence)</summary>\n\n```\n%s\n```\n</details>\n\n", pageText)
	}

	scaffoldFile := filepath.Join(datasetDir, "end-to-end.authoring-scaffold.md")
	if err := os.WriteFile(scaffoldFile, doc.Bytes(), 0o644); err != nil {
		return report, err
	}
	report.ScaffoldFile = scaffoldFile

	// Draft JSONL: the auto-generated cases, both variants, as a starting point.
	var draft bytes.Buffer
	encoder := json.NewEncoder(&draft)
	for _, category := range P0Categories {
		for index, candidate := range byCategory[category] {
			baseID := fmt.Sprintf("e2e-%s-%02d", category, index+1)
			for _, variant := range []string{"text", "image"} {
				item := draftCase(baseID, variant, candidate)
				_ = encoder.Encode(item)
				report.DraftRows++
			}
		}
	}
	draftFile := filepath.Join(datasetDir, "end-to-end.draft.jsonl")
	if err := os.WriteFile(draftFile, draft.Bytes(), 0o644); err != nil {
		return report, err
	}
	report.DraftFile = draftFile
	return report, nil
}

func draftCase(baseID, variant string, candidate p0Candidate) Case {
	item := Case{
		CaseID:               fmt.Sprintf("%s-%s", baseID, variant),
		Stage:                StageEndToEnd,
		BaseID:               baseID,
		Capabilities:         []string{"ANSWER_FROM_EVIDENCE"},
		TaskFamily:           candidate.TaskFamily,
		Choices:              candidate.Choices,
		Instruction:          candidate.Question,
		Variant:              variant,
		PageRefs:             []int{candidate.Page.Number},
		GroundTruthAddresses: []string{candidate.Page.Address},
		EvidenceCID:          candidate.Page.CID,
		SourceMethod:         candidate.Method,
		RequiredFacts:        []string{candidate.EvidenceFragment},
		Expected:             buildExpected(candidate),
	}
	if variant == "text" {
		item.EvidenceText = candidate.Page.Text
	}
	return item
}

type numberContext struct {
	number   string
	sentence string
}

func numberContexts(text string, pageNumber int) []numberContext {
	var out []numberContext
	seen := map[string]bool{}
	for _, sentence := range sentencesOf(text) {
		for _, span := range numberSpan.FindAllString(sentence, -1) {
			if span == fmt.Sprintf("%d", pageNumber) || seen[span] {
				continue
			}
			if _, ok := firstNumber(span); !ok {
				continue
			}
			seen[span] = true
			out = append(out, numberContext{number: span, sentence: truncate(sentence, 160)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].number < out[j].number })
	return out
}

func quoteAll(values []string) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = fmt.Sprintf("%q", value)
	}
	return out
}

func truncate(text string, limit int) string {
	text = collapseSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "…"
}
