package parrotlab

import (
	"regexp"
	"strings"
)

// Heading-aware layer for catalog-structured sources (e.g. a refactoring
// catalog): pages are "<Pattern Name> / Motivation / Mechanics / Example",
// not flowing prose. The reliable signal is the merged heading regions and
// the prose *under* the named sections — not the whole page, which is mostly
// code listings.

var sectionWords = map[string]bool{
	"motivation": true, "mechanics": true, "example": true, "examples": true,
	"summary": true, "introduction": true, "conclusion": true,
}

var headingLineRe = regexp.MustCompile(`^([A-Z][A-Za-z0-9/&'-]*)(\s+[A-Za-z0-9/&'()-]+){0,7}[:]?$`)

var captionPrefix = regexp.MustCompile(`^(?i)(figure|fig\.?|table|appendix|plate|equation|eq\.?|box)\b`)

// isCleanHeading rejects figure captions, equations, running-header debris
// and prose fragments masquerading as section titles.
func isCleanHeading(text string) bool {
	text = strings.TrimSpace(text)
	fields := strings.Fields(text)
	if len(fields) < 2 || len(fields) > 9 || len(text) < 8 || len(text) > 64 {
		return false
	}
	if captionPrefix.MatchString(text) || strings.HasPrefix(text, "=") {
		return false
	}
	if strings.HasSuffix(text, ".") || strings.HasSuffix(text, ",") || strings.Contains(text, "(") {
		return false
	}
	letters := 0
	for _, runeValue := range text {
		if (runeValue >= 'a' && runeValue <= 'z') || (runeValue >= 'A' && runeValue <= 'Z') {
			letters++
		}
	}
	return float64(letters)/float64(len([]rune(text))) >= 0.75
}

// mergedHeadings joins consecutive heading/subheading regions (a real
// heading is often split across regions) into logical heading strings, in
// reading order. Bare page numbers and caption/equation regions are dropped.
func mergedHeadings(page SourcePage) []headingSpan {
	var spans []headingSpan
	var current *headingSpan
	for _, region := range page.Regions {
		isHeading := region.Kind == "heading" || region.Kind == "subheading"
		if !isHeading || region.Text == "" || bareNumberLine.MatchString(region.Text) || region.FontSize < 14 ||
			captionPrefix.MatchString(strings.TrimSpace(region.Text)) || strings.HasPrefix(strings.TrimSpace(region.Text), "=") {
			current = nil
			continue
		}
		if current != nil && region.ReadingOrder == current.lastOrder+1 && abs(region.FontSize-current.font) <= 2 {
			current.Text += " " + region.Text
			current.lastOrder = region.ReadingOrder
			if region.FontSize > current.font {
				current.font = region.FontSize
			}
			continue
		}
		spans = append(spans, headingSpan{Text: region.Text, font: region.FontSize, order: region.ReadingOrder, lastOrder: region.ReadingOrder})
		current = &spans[len(spans)-1]
	}
	return spans
}

type headingSpan struct {
	Text      string
	font      int
	order     int
	lastOrder int
}

// headingModel is the structured view of one catalog page.
type headingModel struct {
	PatternName string   // largest heading that is not a section word
	Headings    []string // all merged headings, in reading order
	Motivation  string   // prose under the "Motivation" heading (from page.Text)
	Mechanics   string   // prose under the "Mechanics" heading
}

func buildHeadingModel(page SourcePage) headingModel {
	model := headingModel{}
	bestFont := 0
	beforeSections := true
	for _, span := range mergedHeadings(page) {
		text := strings.TrimSpace(span.Text)
		model.Headings = append(model.Headings, text)
		lower := strings.ToLower(text)
		if sectionWords[lower] || strings.Contains(lower, "example") {
			beforeSections = false
			continue
		}
		// The title is a big clean heading that comes before the Motivation/
		// Mechanics section headings; later big "headings" are inline refs.
		if beforeSections && span.font > bestFont && isCleanHeading(text) {
			bestFont = span.font
			model.PatternName = text
		}
	}
	model.Motivation = sectionBody(page.Text, "motivation")
	model.Mechanics = sectionBody(page.Text, "mechanics")
	return model
}

// isRefactoringEntryPage reports whether the page *starts* a catalog entry:
// its first heading is a large multi-word title, and a "Motivation" section
// heading follows it. A later page of the same entry (Mechanics/Example, or
// an inline title reference) does not qualify.
func isRefactoringEntryPage(page SourcePage) bool {
	titleOrder := -1
	motivationOrder := -1
	firstHeadingSeen := false
	for _, region := range page.Regions {
		if region.Kind != "heading" && region.Kind != "subheading" {
			continue
		}
		text := strings.TrimSpace(region.Text)
		lower := strings.ToLower(text)
		if bareNumberLine.MatchString(text) {
			continue
		}
		if lower == "motivation" && motivationOrder < 0 {
			motivationOrder = region.ReadingOrder
		}
		if !firstHeadingSeen {
			firstHeadingSeen = true
			if region.FontSize >= 17 && len(strings.Fields(text)) >= 2 &&
				!strings.HasPrefix(text, "Figure") && !sectionWords[lower] {
				titleOrder = region.ReadingOrder
			}
		}
	}
	return titleOrder >= 0 && motivationOrder > titleOrder
}

// sectionBody returns the prose lines following a line that is exactly the
// section name, stopping at the next heading-like line.
func sectionBody(text, section string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		if !strings.EqualFold(strings.TrimSpace(line), section) {
			continue
		}
		var body []string
		for _, rest := range lines[index+1:] {
			trimmed := strings.TrimSpace(rest)
			if trimmed == "" {
				continue
			}
			if isLikelyHeading(trimmed) {
				break
			}
			body = append(body, trimmed)
			if len(strings.Join(body, " ")) > 600 {
				break
			}
		}
		return collapseSpace(strings.Join(body, " "))
	}
	// Fallback: the section word is present but not on its own line (the
	// flattened text glued it to the following sentence). Split on it.
	flat := collapseSpace(text)
	marker := regexp.MustCompile(`(?i)(^|[.\s])` + regexp.QuoteMeta(section) + `\s`)
	if location := marker.FindStringIndex(flat); location != nil {
		after := strings.TrimSpace(flat[location[1]:])
		if next := nextSectionIndex(after); next > 0 {
			after = after[:next]
		}
		if len(after) > 600 {
			after = after[:600]
		}
		return strings.TrimSpace(after)
	}
	return ""
}

// nextSectionIndex finds where the following named section starts, so a
// section body does not bleed into the next one.
func nextSectionIndex(text string) int {
	best := -1
	for word := range sectionWords {
		re := regexp.MustCompile(`(?i)[.\s]` + regexp.QuoteMeta(word) + `\s`)
		if location := re.FindStringIndex(text); location != nil {
			if best < 0 || location[0] < best {
				best = location[0]
			}
		}
	}
	return best
}

func isLikelyHeading(line string) bool {
	if sectionWords[strings.ToLower(line)] {
		return true
	}
	fields := strings.Fields(line)
	if len(fields) == 0 || len(fields) > 8 {
		return false
	}
	if strings.ContainsAny(line[len(line)-1:], ".!?,;:") && !strings.HasSuffix(line, ":") {
		return false
	}
	capitalised := 0
	for _, field := range fields {
		runes := []rune(field)
		if len(runes) > 0 && runes[0] >= 'A' && runes[0] <= 'Z' {
			capitalised++
		}
	}
	return float64(capitalised)/float64(len(fields)) >= 0.7
}

// proseText is the text the generic generators should mine: the named
// section bodies when the page is a catalog entry, otherwise the whole
// extracted page. A catalog entry has real "Motivation"/"Mechanics" heading
// regions — a stray "motivation" in running prose must not truncate the page.
func proseText(page SourcePage) string {
	if !isRefactoringEntryPage(page) {
		return page.Text
	}
	model := buildHeadingModel(page)
	parts := []string{}
	for _, section := range []string{model.Motivation, model.Mechanics} {
		if strings.TrimSpace(section) != "" {
			parts = append(parts, section)
		}
	}
	if len(parts) == 0 {
		return page.Text
	}
	return strings.Join(parts, "\n")
}

// headingNames returns the merged headings that name a topic (not a section
// word, not "Example…", at least two words).
func headingNames(model headingModel) []string {
	var out []string
	seen := map[string]bool{}
	for _, heading := range model.Headings {
		lower := strings.ToLower(heading)
		if sectionWords[lower] || strings.Contains(lower, "example") || seen[lower] || !isCleanHeading(heading) {
			continue
		}
		seen[lower] = true
		out = append(out, heading)
	}
	return out
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
