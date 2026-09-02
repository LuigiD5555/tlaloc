package parrotlab

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tlaloc.local/behaviorlab/internal/foldtest"
)

// P0 Generator R0 — builds the frozen end-to-end dataset (SPEC §4) from the
// existing Origami extraction path. It selects usable pages deterministically,
// generates candidate question/answer pairs whose answer is demonstrable from
// the page's own extracted text, keeps every case's exact evidence, and emits
// a text variant and an image variant per case (per the P0 modality decision).

// P0Categories and their per-category quota (SPEC §4: 6 each).
var P0Categories = []string{"locate", "entity", "factual", "numeric", "synthesis"}

// P0Options controls generation.
type P0Options struct {
	Seed        int64
	PageCount   int // default 10
	PerCategory int // default 6
	Variants    []string
}

// P0Report is the generator outcome.
type P0Report struct {
	SourceID       string          `json:"source_id"`
	PageSelection  string          `json:"page_selection"`
	SelectedPages  []int           `json:"selected_pages"`
	CasesWritten   int             `json:"cases_written"`
	ByCategory     map[string]int  `json:"by_category"`
	Shortfalls     map[string]int  `json:"shortfalls,omitempty"`
	ImagesRendered int             `json:"images_rendered"`
	ImageErrors    []string        `json:"image_errors,omitempty"`
	Dataset        string          `json:"dataset"`
	Provenance     string          `json:"provenance"`
	Scaffold       *ScaffoldReport `json:"scaffold,omitempty"`
}

type p0Candidate struct {
	Category         string
	Question         string
	Answer           string
	Aliases          []string
	Choices          []string
	TaskFamily       string
	EvidenceFragment string
	Method           string
	Page             SourcePage
}

type p0ProvenanceEntry struct {
	BaseID           string `json:"base_id"`
	Category         string `json:"category"`
	Question         string `json:"question"`
	Answer           string `json:"answer"`
	Page             int    `json:"page"`
	Address          string `json:"address"`
	PageCID          string `json:"page_cid"`
	EvidenceFragment string `json:"evidence_fragment"`
	EvidenceSHA256   string `json:"evidence_sha256"`
	Method           string `json:"method"`
}

// GenerateEndToEnd runs the P0 generator against a page provider.
func GenerateEndToEnd(provider PageProvider, datasetDir string, opts P0Options) (P0Report, error) {
	if opts.PageCount <= 0 {
		opts.PageCount = 10
	}
	if opts.PerCategory <= 0 {
		opts.PerCategory = 6
	}
	if len(opts.Variants) == 0 {
		opts.Variants = []string{"text", "image"}
	}
	report := P0Report{SourceID: provider.SourceID(), ByCategory: map[string]int{}, Shortfalls: map[string]int{}}

	allPages, err := provider.Pages()
	if err != nil {
		return report, err
	}
	usable := usablePages(allPages)
	// Prefer catalog-entry pages (title + Motivation + Mechanics): every
	// generator has clean signal on them. Fall back to any usable page.
	pool := usable
	var entry []SourcePage
	for _, page := range usable {
		if isRefactoringEntryPage(page) {
			entry = append(entry, page)
		}
	}
	if len(entry) >= opts.PageCount {
		pool = entry
		report.PageSelection = "refactoring-entry-pages"
	} else {
		report.PageSelection = "usable-pages"
	}
	if len(pool) < opts.PageCount {
		return report, fmt.Errorf("only %d pages after filtering, need %d", len(pool), opts.PageCount)
	}
	positions := foldtest.SelectSpacedPages(len(pool), opts.PageCount, opts.Seed)
	selected := make([]SourcePage, 0, len(positions))
	for _, position := range positions {
		selected = append(selected, pool[position-1])
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Number < selected[j].Number })
	for _, page := range selected {
		report.SelectedPages = append(report.SelectedPages, page.Number)
	}

	// Candidate pool per category, page order shuffled deterministically so
	// the quota is spread across pages.
	byCategory := map[string][]p0Candidate{}
	for _, category := range P0Categories {
		source := rand.New(rand.NewSource(opts.Seed + int64(len(category))))
		pageOrder := append([]SourcePage(nil), selected...)
		source.Shuffle(len(pageOrder), func(i, j int) { pageOrder[i], pageOrder[j] = pageOrder[j], pageOrder[i] })
		perPage := map[int]int{}
		for round := 0; round < opts.PerCategory; round++ {
			for _, page := range pageOrder {
				if len(byCategory[category]) >= opts.PerCategory {
					break
				}
				candidates := generateCategory(category, page, selected, source)
				if perPage[page.Number] < len(candidates) {
					byCategory[category] = append(byCategory[category], candidates[perPage[page.Number]])
					perPage[page.Number]++
				}
			}
			if len(byCategory[category]) >= opts.PerCategory {
				break
			}
		}
	}

	// Cross-category dedup: the same fact tested the same way (page + answer
	// + task family) must not appear twice. A choice "is it X or Y?" and an
	// open "name it from its description" with the same answer are different
	// skills and both kept.
	usedFact := map[string]bool{}
	for _, category := range P0Categories {
		kept := byCategory[category][:0]
		for _, candidate := range byCategory[category] {
			key := fmt.Sprintf("%d|%s|%s", candidate.Page.Number, normaliseAnswer(candidate.Answer), candidate.TaskFamily)
			if usedFact[key] {
				continue
			}
			usedFact[key] = true
			kept = append(kept, candidate)
		}
		byCategory[category] = kept
		if got := len(kept); got < opts.PerCategory {
			report.Shortfalls[category] = opts.PerCategory - got
		}
	}

	var lines bytes.Buffer
	encoder := json.NewEncoder(&lines)
	var provenance []p0ProvenanceEntry
	imageDir := filepath.Join(datasetDir, "end-to-end", "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return report, err
	}

	for _, category := range P0Categories {
		for index, candidate := range byCategory[category] {
			baseID := fmt.Sprintf("e2e-%s-%02d", category, index+1)
			fragmentHash := sha256.Sum256([]byte(candidate.EvidenceFragment))
			provenance = append(provenance, p0ProvenanceEntry{
				BaseID: baseID, Category: category, Question: candidate.Question,
				Answer: candidate.Answer, Page: candidate.Page.Number, Address: candidate.Page.Address,
				PageCID: candidate.Page.CID, EvidenceFragment: candidate.EvidenceFragment,
				EvidenceSHA256: hex.EncodeToString(fragmentHash[:]), Method: candidate.Method,
			})

			imageRel := ""
			for _, variant := range opts.Variants {
				if variant == "image" {
					png, renderErr := provider.RenderPNG(candidate.Page.Number)
					if renderErr != nil {
						report.ImageErrors = append(report.ImageErrors, fmt.Sprintf("%s: %v", baseID, renderErr))
						continue
					}
					imageRel = filepath.ToSlash(filepath.Join("end-to-end", "images", baseID+".png"))
					if err := os.WriteFile(filepath.Join(datasetDir, filepath.FromSlash(imageRel)), png, 0o644); err != nil {
						return report, err
					}
					report.ImagesRendered++
				}
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
				if variant == "image" {
					if imageRel == "" {
						continue
					}
					item.ImagePath = imageRel
				}
				if err := encoder.Encode(item); err != nil {
					return report, err
				}
				report.CasesWritten++
			}
			report.ByCategory[category]++
		}
	}

	datasetFile := filepath.Join(datasetDir, "end-to-end.jsonl")
	if err := os.WriteFile(datasetFile, lines.Bytes(), 0o644); err != nil {
		return report, err
	}
	provenanceFile := filepath.Join(datasetDir, "end-to-end.provenance.json")
	provenanceBytes, _ := json.MarshalIndent(provenance, "", "  ")
	if err := os.WriteFile(provenanceFile, append(provenanceBytes, '\n'), 0o644); err != nil {
		return report, err
	}
	report.Dataset = datasetFile
	report.Provenance = provenanceFile

	scaffold, err := writeAuthoringScaffold(datasetDir, provider, selected, byCategory)
	if err != nil {
		return report, err
	}
	report.Scaffold = &scaffold
	return report, nil
}

func buildExpected(candidate p0Candidate) Expected {
	if candidate.TaskFamily == "numeric" {
		if value, ok := firstNumber(candidate.Answer); ok {
			return Expected{Value: candidate.Answer, Number: &value, Tolerance: 0}
		}
	}
	return Expected{Value: candidate.Answer, Aliases: candidate.Aliases}
}

// --- usable-page filtering (SPEC §2 exclusions) ---

var dotLeader = regexp.MustCompile(`\.{4,}`)

func usablePages(pages []SourcePage) []SourcePage {
	total := len(pages)
	pick := func(relaxEnds bool) []SourcePage {
		var out []SourcePage
		for index, page := range pages {
			if !relaxEnds && (index < 3 || index >= total-3) {
				continue
			}
			text := strings.TrimSpace(page.Text)
			if len(text) < 320 {
				continue
			}
			if letterRatio(text) < 0.55 {
				continue
			}
			head := strings.ToLower(firstNonEmptyLine(text))
			if containsAnyWord(head, "references", "bibliography", "index", "contents", "acknowledgements", "acknowledgments", "copyright", "colophon") {
				continue
			}
			if len(dotLeader.FindAllString(text, -1)) > 15 {
				continue
			}
			if shortLineRatio(text) > 0.6 {
				continue
			}
			out = append(out, page)
		}
		return out
	}
	if out := pick(false); len(out) >= 1 {
		return out
	}
	return pick(true)
}

func letterRatio(text string) float64 {
	letters := 0
	for _, runeValue := range text {
		if (runeValue >= 'a' && runeValue <= 'z') || (runeValue >= 'A' && runeValue <= 'Z') || runeValue == ' ' {
			letters++
		}
	}
	if len(text) == 0 {
		return 0
	}
	return float64(letters) / float64(len([]rune(text)))
}

func shortLineRatio(text string) float64 {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return 0
	}
	short := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) < 25 {
			short++
		}
	}
	return float64(short) / float64(len(lines))
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func containsAnyWord(haystack string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(haystack, word) {
			return true
		}
	}
	return false
}

// --- question generators ---

var (
	sentenceSplit = regexp.MustCompile(`(?s)(.+?[.!?])(?:\s|$)`)
	numberSpan    = regexp.MustCompile(`\$?\d[\d,]*(?:\.\d+)?%?`)
	definitionRe  = regexp.MustCompile(`([A-Z][A-Za-z0-9-]+(?: [A-Za-z0-9-]+){0,3}) ((?:is|are|was|were|refers to|means|is defined as|coordinates|provides|turns|records|checks|keeps|moves|packs|selects|authorises|authorizes|evaluates|compares|blocks|triggers|restricts|stores|reads|resolves|carries|normalises|normalizes) [^.]{15,150})\.`)
	capPhraseRe   = regexp.MustCompile(`\b([A-Z][A-Za-z0-9-]+(?: [A-Z][A-Za-z0-9-]+){1,3})\b`)
)

func generateCategory(category string, page SourcePage, allSelected []SourcePage, source *rand.Rand) []p0Candidate {
	model := buildHeadingModel(page)
	catalogPage := model.PatternName != "" || len(headingNames(model)) > 0
	switch category {
	case "numeric":
		return genNumeric(page)
	case "factual":
		return genFactual(page)
	case "locate":
		return genLocate(page, allSelected, model, source)
	case "entity":
		if fromHeading := genEntityFromHeading(page, model); len(fromHeading) > 0 {
			return fromHeading
		}
		if catalogPage {
			return nil // generic definition-mining produces fragments on catalog pages
		}
		return genEntity(page)
	case "synthesis":
		if fromHeading := genSynthesisFromHeadings(page, model); len(fromHeading) > 0 {
			return fromHeading
		}
		if catalogPage {
			return nil
		}
		return genSynthesis(page)
	}
	return nil
}

// genEntityFromHeading: name the refactoring given its stated motivation.
func genEntityFromHeading(page SourcePage, model headingModel) []p0Candidate {
	if model.PatternName == "" || len(model.Motivation) < 40 {
		return nil
	}
	sentences := sentencesOf(model.Motivation)
	if len(sentences) == 0 {
		return nil
	}
	motivation := sentences[0]
	return []p0Candidate{{
		Category:         "entity",
		Question:         fmt.Sprintf("This page describes one refactoring. Its motivation section begins: %q. Name the refactoring, and nothing else.", motivation),
		Answer:           model.PatternName,
		TaskFamily:       "exact",
		EvidenceFragment: model.PatternName + " — Motivation: " + motivation,
		Method:           "heading-motivation-r0",
		Page:             page,
	}}
}

// genSynthesisFromHeadings: reading-order of the two catalog headings on the page.
func genSynthesisFromHeadings(page SourcePage, model headingModel) []p0Candidate {
	var names []string
	seen := map[string]bool{}
	for _, heading := range model.Headings {
		lower := strings.ToLower(heading)
		if sectionWords[lower] || strings.Contains(lower, "example") || bareNumberLine.MatchString(heading) {
			continue
		}
		if len(strings.Fields(heading)) < 2 || seen[lower] {
			continue
		}
		seen[lower] = true
		names = append(names, heading)
	}
	if len(names) < 2 {
		return nil
	}
	first, second := names[0], names[1]
	return []p0Candidate{{
		Category:         "synthesis",
		Question:         fmt.Sprintf("The page shows two refactoring sections. Which heading appears first on the page: %q or %q? Answer with that heading only.", first, second),
		Answer:           first,
		Choices:          []string{first, second},
		TaskFamily:       "choice",
		EvidenceFragment: first + " (before) / " + second + " (after)",
		Method:           "heading-order-r0",
		Page:             page,
	}}
}

func sentencesOf(text string) []string {
	var out []string
	for _, match := range sentenceSplit.FindAllStringSubmatch(collapseSpace(text), -1) {
		sentence := strings.TrimSpace(match[1])
		words := strings.Fields(sentence)
		if len(words) >= 6 && len(words) <= 40 && !looksLikeCode(sentence) {
			out = append(out, sentence)
		}
	}
	return out
}

// looksLikeCode rejects a "sentence" that is really a snippet of source
// code (operators, braces, semicolons, camelCase call syntax).
func looksLikeCode(sentence string) bool {
	if strings.ContainsAny(sentence, "{}=;") {
		return true
	}
	if strings.Count(sentence, "(") != strings.Count(sentence, ")") {
		return true
	}
	symbols := strings.Count(sentence, "*") + strings.Count(sentence, "+") +
		strings.Count(sentence, "/") + strings.Count(sentence, "_")
	return symbols >= 2
}

func collapseSpace(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "\n", " ")), " ")
}

func occurrences(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	return strings.Count(strings.ToLower(haystack), strings.ToLower(needle))
}

func genNumeric(page SourcePage) []p0Candidate {
	var out []p0Candidate
	text := proseText(page)
	for _, sentence := range sentencesOf(text) {
		spans := numberSpan.FindAllString(sentence, -1)
		if len(spans) != 1 {
			continue
		}
		number := spans[0]
		if number == fmt.Sprintf("%d", page.Number) {
			continue // running page number, not a fact
		}
		if occurrences(text, number) != 1 {
			continue
		}
		if _, ok := firstNumber(number); !ok {
			continue
		}
		blanked := strings.Replace(sentence, number, "_____", 1)
		if strings.HasPrefix(strings.TrimSpace(blanked), "_____") {
			continue // blank at the very start reads as nonsense
		}
		out = append(out, p0Candidate{
			Category:         "numeric",
			Question:         fmt.Sprintf("Fill the blank using only the page. Answer with the value only.\n\"%s\"", blanked),
			Answer:           strings.TrimPrefix(number, "$"),
			TaskFamily:       "numeric",
			EvidenceFragment: sentence,
			Method:           "numeric-cloze-r0",
			Page:             page,
		})
	}
	return out
}

var termStopwords = map[string]bool{
	"the": true, "a": true, "an": true, "this": true, "that": true, "these": true,
	"those": true, "it": true, "there": true, "nothing": true, "one": true, "each": true,
	"every": true, "some": true, "any": true, "no": true, "he": true, "she": true,
	"they": true, "we": true, "you": true, "its": true, "their": true,
	// clause-fragment openers — real concept names never start with these
	"if": true, "when": true, "unless": true, "actually": true, "so": true,
	"then": true, "because": true, "although": true, "while": true, "since": true,
	"however": true, "therefore": true, "in": true, "on": true, "for": true,
	"at": true, "by": true, "with": true, "as": true, "but": true, "and": true,
	"or": true, "to": true, "of": true, "from": true, "after": true, "before": true,
	"here": true, "now": true, "thus": true, "hence": true,
	"another": true, "both": true, "such": true, "either": true, "neither": true,
	"first": true, "second": true, "third": true, "next": true, "last": true,
}

// looksLikeClauseFragment rejects a candidate "term" that is really a snippet
// of a sentence (contains a lowercase function word mid-phrase).
func looksLikeClauseFragment(term string) bool {
	fields := strings.Fields(term)
	if len(fields) == 0 {
		return true
	}
	if termStopwords[strings.ToLower(fields[0])] {
		return true
	}
	for _, word := range fields[1:] {
		switch strings.ToLower(word) {
		case "is", "are", "was", "were", "it", "the", "a", "an", "to", "of", "in",
			"that", "this", "and", "or", "you", "we", "they", "if", "then":
			return true
		}
	}
	return false
}

func trimArticle(phrase string) string {
	lower := strings.ToLower(phrase)
	for _, article := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(lower, article) {
			return strings.TrimSpace(phrase[len(article):])
		}
	}
	return phrase
}

func genEntity(page SourcePage) []p0Candidate {
	var out []p0Candidate
	for _, match := range definitionRe.FindAllStringSubmatch(collapseSpace(page.Text), -1) {
		term := trimArticle(strings.TrimSpace(match[1]))
		definition := strings.TrimSpace(match[2])
		if len(term) > 40 || len(strings.Fields(term)) < 2 || looksLikeClauseFragment(term) {
			continue
		}
		if occurrences(definition, term) > 0 || len(strings.Fields(definition)) < 4 {
			continue
		}
		out = append(out, p0Candidate{
			Category:         "entity",
			Question:         fmt.Sprintf("Which term from the page is described as: \"%s\"? Answer with the term only.", definition),
			Answer:           term,
			TaskFamily:       "entity",
			EvidenceFragment: fmt.Sprintf("%s ... %s", term, definition),
			Method:           "definition-pattern-r0",
			Page:             page,
		})
	}
	return out
}

func genFactual(page SourcePage) []p0Candidate {
	var out []p0Candidate
	text := proseText(page)
	for _, sentence := range sentencesOf(text) {
		phrases := capPhraseRe.FindAllString(sentence, -1)
		if len(phrases) == 0 {
			continue
		}
		phrase := phrases[len(phrases)-1]
		if strings.HasPrefix(sentence, phrase) || occurrences(text, phrase) != 1 || looksLikeClauseFragment(phrase) {
			continue
		}
		blanked := strings.Replace(sentence, phrase, "_____", 1)
		if strings.HasPrefix(strings.TrimSpace(blanked), "_____") {
			continue
		}
		out = append(out, p0Candidate{
			Category:         "factual",
			Question:         fmt.Sprintf("Fill the blank using only the page. Answer with the missing words only.\n\"%s\"", blanked),
			Answer:           phrase,
			TaskFamily:       "exact",
			EvidenceFragment: sentence,
			Method:           "phrase-cloze-r0",
			Page:             page,
		})
	}
	return out
}

func genLocate(page SourcePage, allSelected []SourcePage, model headingModel, source *rand.Rand) []p0Candidate {
	// Only the page's dominant heading — inline cross-references also render
	// as subheadings, so a full heading list would claim the page "covers"
	// things it only mentions.
	if model.PatternName == "" {
		return nil
	}
	onPage := []string{model.PatternName}
	seenElsewhere := map[string]bool{}
	var elsewhere []string
	for _, other := range allSelected {
		if other.Number == page.Number {
			continue
		}
		term := buildHeadingModel(other).PatternName
		lower := strings.ToLower(term)
		if term == "" || seenElsewhere[lower] || occurrences(page.Text, term) != 0 {
			continue
		}
		seenElsewhere[lower] = true
		elsewhere = append(elsewhere, term)
	}
	if len(elsewhere) == 0 {
		return nil
	}
	var out []p0Candidate
	for _, present := range onPage {
		absent := elsewhere[source.Intn(len(elsewhere))]
		choices := []string{present, absent}
		if source.Intn(2) == 0 {
			choices[0], choices[1] = choices[1], choices[0]
		}
		out = append(out, p0Candidate{
			Category:         "locate",
			Question:         fmt.Sprintf("This page has a section heading for exactly one of these refactorings: %q or %q. Which one? Answer with that heading only.", choices[0], choices[1]),
			Answer:           present,
			Choices:          choices,
			TaskFamily:       "choice",
			EvidenceFragment: "page heading: " + present,
			Method:           "heading-presence-r0",
			Page:             page,
		})
	}
	return out
}

func genSynthesis(page SourcePage) []p0Candidate {
	var out []p0Candidate
	sentences := sentencesOf(page.Text)
	var numbers []string
	for _, sentence := range sentences {
		for _, span := range numberSpan.FindAllString(sentence, -1) {
			if value, ok := firstNumber(span); ok && value != 0 && occurrences(page.Text, span) == 1 {
				numbers = append(numbers, span)
			}
		}
	}
	if len(numbers) >= 2 {
		first, second := numbers[0], numbers[1]
		valueA, _ := firstNumber(first)
		valueB, _ := firstNumber(second)
		larger := first
		if valueB > valueA {
			larger = second
		}
		choices := []string{first, second}
		out = append(out, p0Candidate{
			Category:         "synthesis",
			Question:         fmt.Sprintf("The page states both %s and %s. Which is the larger number? Answer with that number only.", first, second),
			Answer:           larger,
			Choices:          choices,
			TaskFamily:       "choice",
			EvidenceFragment: fmt.Sprintf("%s / %s", first, second),
			Method:           "two-number-compare-r0",
			Page:             page,
		})
	}
	terms := distinctiveTerms(page.Text)
	if len(terms) >= 2 {
		first, second := terms[0], terms[1]
		firstPos := strings.Index(strings.ToLower(page.Text), strings.ToLower(first))
		secondPos := strings.Index(strings.ToLower(page.Text), strings.ToLower(second))
		earlier := first
		if secondPos < firstPos {
			earlier = second
		}
		out = append(out, p0Candidate{
			Category:         "synthesis",
			Question:         fmt.Sprintf("On the page, which is mentioned first: \"%s\" or \"%s\"? Answer with that phrase only.", first, second),
			Answer:           earlier,
			Choices:          []string{first, second},
			TaskFamily:       "choice",
			EvidenceFragment: firstSentenceContaining(page.Text, earlier),
			Method:           "reading-order-r0",
			Page:             page,
		})
	}
	return out
}

func distinctiveTerms(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range capPhraseRe.FindAllString(collapseSpace(text), -1) {
		phrase := trimArticle(raw)
		key := strings.ToLower(phrase)
		if seen[key] || len(phrase) < 6 || len(phrase) > 45 || looksLikeClauseFragment(phrase) {
			continue
		}
		if occurrences(text, phrase) > 3 {
			continue
		}
		seen[key] = true
		out = append(out, phrase)
	}
	sort.Strings(out)
	return out
}

func firstSentenceContaining(text, needle string) string {
	for _, sentence := range sentencesOf(text) {
		if strings.Contains(strings.ToLower(sentence), strings.ToLower(needle)) {
			return sentence
		}
	}
	return needle
}

// --- P0 validation (SPEC §4 automatic checks) ---

// ValidateEndToEnd checks every P0 case: the answer must be demonstrable from
// its own evidence, the address and evidence must exist, the model input must
// not leak the answer, and questions must be unique and unambiguous.
func ValidateEndToEnd(cases []Case) []error {
	var problems []error
	add := func(format string, args ...any) { problems = append(problems, fmt.Errorf(format, args...)) }
	seenQuestion := map[string]string{}

	for _, item := range cases {
		id := item.CaseID
		if item.Stage != StageEndToEnd {
			continue
		}
		// The precise evidence fragment is the proof; the full page text
		// (what Parrot sees) is often reformatted so a heading/phrase is not
		// a contiguous substring of it.
		evidence := strings.Join(item.RequiredFacts, "\n")
		if strings.TrimSpace(evidence) == "" {
			evidence = item.EvidenceText
		}
		answer := normaliseAnswer(item.Expected.Value)

		if strings.TrimSpace(evidence) == "" {
			add("%s: no evidence to check the answer against", id)
		} else if !answerDemonstrable(item, evidence) {
			add("%s: answer %q is not demonstrable from its evidence (external knowledge?)", id, item.Expected.Value)
		}
		if item.EvidenceCID == "" {
			add("%s: missing evidence_cid", id)
		}
		if len(item.GroundTruthAddresses) == 0 || !looksLikeAddress(item.GroundTruthAddresses[0]) {
			add("%s: missing or malformed Origami address", id)
		}
		// For a choice question the options (one of them correct) are meant to
		// appear in the prompt; the leak test only applies to open answers.
		if item.TaskFamily != "choice" && answer != "" && strings.Contains(normaliseAnswer(item.Instruction), answer) {
			add("%s: the question leaks the answer", id)
		}
		questionKey := collapseSpace(strings.ToLower(item.Instruction)) + "|" + answer
		if prior, dup := seenQuestion[questionKey]; dup && prior != item.BaseID {
			add("%s: duplicate question (also %s)", id, prior)
		}
		seenQuestion[questionKey] = item.BaseID
		if item.EvidenceText != "" && ambiguousClozeAnswer(item) {
			add("%s: cloze answer appears %d times in the page (ambiguous)", id, occurrences(item.EvidenceText, item.Expected.Value))
		}
		if item.TaskFamily == "choice" && len(item.Choices) < 2 {
			add("%s: choice case without a 2-option universe", id)
		}
		if (item.TaskFamily == "entity" || item.TaskFamily == "exact") && looksLikeClauseFragment(item.Expected.Value) {
			add("%s: answer %q is a sentence fragment, not a term", id, item.Expected.Value)
		}
		if item.TaskFamily == "numeric" {
			for _, page := range item.PageRefs {
				if item.Expected.Value == fmt.Sprintf("%d", page) {
					add("%s: numeric answer equals the page number (running header, not a fact)", id)
				}
			}
		}
	}
	return problems
}

func answerDemonstrable(item Case, evidence string) bool {
	if item.TaskFamily == "numeric" {
		want, ok := firstNumber(item.Expected.Value)
		if !ok {
			return false
		}
		for _, span := range numberSpan.FindAllString(evidence, -1) {
			if got, ok := firstNumber(span); ok && got == want {
				return true
			}
		}
		return false
	}
	return tokenSubset(item.Expected.Value, strings.ToLower(evidence))
}

func ambiguousClozeAnswer(item Case) bool {
	if item.TaskFamily != "exact" && item.TaskFamily != "numeric" {
		return false
	}
	return occurrences(item.EvidenceText, item.Expected.Value) > 1
}

func looksLikeAddress(address string) bool {
	if strings.Contains(address, "://") {
		return true
	}
	return foldtest.ValidateAddressFormat(address)
}
