package parrotlab

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"testing"
)

type fakeProvider struct{ pages []SourcePage }

func (f fakeProvider) SourceID() string             { return "fake-book" }
func (f fakeProvider) Pages() ([]SourcePage, error) { return f.pages, nil }
func (f fakeProvider) RenderPNG(number int) ([]byte, error) {
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	return buffer.Bytes(), nil
}

func syntheticBook() fakeProvider {
	pages := []SourcePage{
		{Number: 1, Address: "ohf://fake/page/1", CID: "c1", Text: "Cover"},
		{Number: 2, Address: "ohf://fake/page/2", CID: "c2", Text: "Table of Contents\nIntroduction ...... 1\nChapter One ...... 12\nChapter Two ...... 40\nChapter Three ...... 61\nIndex ...... 200\nReferences ...... 210"},
		{Number: 3, Address: "ohf://fake/page/3", CID: "c3", Text: "   "},
	}
	// eight content pages, each rich enough for every generator.
	bodies := []string{
		"A Blackboard Swarm coordinates many small specialists through a shared workspace. The Verification Spine checks every claim against evidence. In one benchmark the swarm reached 87 percent accuracy while a single large call reached 62 percent. The Coordination Overhead stayed under 5 milliseconds. Emergent Behaviour arises from local rules, not central control.",
		"A Capability Planner turns a goal into a directed graph of workers. The Intent Representation records constraints and a risk profile. The team measured 240 tokens of context per worker and a critical path of 8 seconds. The Reference Compiler is deterministic. The Sentinel Panel blocks any action above risk class two.",
		"Tlaloque Lifecycle moves a specialist from candidate to shadow to active. A Calibration Profile must show out of distribution accuracy above 80 percent. The Repertoire Ledger keeps an append only history. Demotion happens after 5 failing runs. The Curriculum Ladder has 10 stages from clean to adversarial.",
		"An Origami Carrier packs a whole book into one image under a canonical visual profile. The Master Prompt describes the machine. The Glyph Calculus normalises evidence to a common scale. The document held 312 pages and 4180 regions. Perceptual Orthogonality keeps channels independent.",
		"The Fold Unfold Harness selects pages spaced apart with a fixed seed. Each address resolves to exact bytes. The harness ran 30 questions across 10 pages. The Evidence Packet carries a content hash. The Address Schema is stable across releases.",
		"A Deterministic Envelope authorises actions from an intent. Risk class ranges from zero to four. The Executor checks preconditions, runs the implementation once, then checks postconditions. A failed postcondition triggers a Verified Rollback. The Policy Sandbox restricts file paths.",
		"Perception Lab evaluates a tiny model on rendered glyphs. The model has 24000 parameters and beat a much larger model on decoding. The Context SIMD width was fixed at 8. The Dimensional Visual Register stores intermediate state. Macro Gestalt reads the whole field at once.",
		"Semantic Verification compares a claim to its evidence with an agreement score. A score above 0.6 counts as agreement. The Structural Level checks JSON shape and required fields. The World Level folds postcondition results into checks. Nothing is Verified on confidence alone.",
	}
	filler := []string{
		"This section reviews the design rationale and how each component connects to the others in the wider architecture, with pointers to earlier chapters where the same ideas were first introduced and discussed at length.",
		"Later material builds on these definitions, so readers unfamiliar with the terminology should study this passage carefully before continuing to the exercises and worked examples that follow in the remainder of the book.",
		"The remainder of the discussion traces how the mechanism behaves under load and what happens when one of its assumptions is violated in practice by an adversarial or simply unlucky input during operation.",
		"An extended example runs through the whole procedure step by step, showing intermediate state at each point and explaining why the deterministic checks reject inputs that would otherwise slip through unnoticed.",
	}
	for index, body := range bodies {
		number := 12 + index*7
		pages = append(pages, SourcePage{
			Number:  number,
			Address: fmt.Sprintf("ohf://fake/page/%d", number),
			CID:     fmt.Sprintf("cid-%d", number),
			Text:    body + " " + filler[index%len(filler)] + " " + filler[(index+1)%len(filler)],
		})
	}
	pages = append(pages,
		SourcePage{Number: 199, Address: "ohf://fake/page/199", CID: "cz1", Text: "Index\nblackboard 12\ncarrier 40\nswarm 12 40 61"},
		SourcePage{Number: 200, Address: "ohf://fake/page/200", CID: "cz2", Text: "References\nSmith 2019\nJones 2020"},
		SourcePage{Number: 201, Address: "ohf://fake/page/201", CID: "cz3", Text: "  "},
	)
	return fakeProvider{pages: pages}
}

func TestUsablePagesFiltersFrontBackAndJunk(t *testing.T) {
	usable := usablePages(syntheticBook().pages)
	for _, page := range usable {
		if page.Number < 12 || page.Number > 100 {
			t.Errorf("kept a non-content page: %d (%q)", page.Number, firstNonEmptyLine(page.Text))
		}
	}
	if len(usable) < 8 {
		t.Fatalf("expected the 8 content pages, kept %d", len(usable))
	}
}

func TestGenerateEndToEndProducesValidPairedCases(t *testing.T) {
	dir := t.TempDir()
	report, err := GenerateEndToEnd(syntheticBook(), dir, P0Options{Seed: 42, PageCount: 8})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(report.SelectedPages) != 8 {
		t.Fatalf("selected %d pages, want 8", len(report.SelectedPages))
	}
	cases, err := LoadCases(report.Dataset)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no cases generated")
	}
	if problems := ValidateEndToEnd(cases); len(problems) > 0 {
		t.Fatalf("generated P0 dataset failed its own validator: %v", problems)
	}
	if problems := Validate(cases); len(problems) > 0 {
		t.Fatalf("generated P0 dataset failed structural validation: %v", problems)
	}
	// every base_id must have both a text and an image variant.
	variants := map[string]map[string]bool{}
	for _, item := range cases {
		if variants[item.BaseID] == nil {
			variants[item.BaseID] = map[string]bool{}
		}
		variants[item.BaseID][item.Variant] = true
		if item.Variant == "text" && !containsSubstr(item.UserText(), item.Instruction) {
			t.Errorf("%s: text variant UserText drops the question", item.CaseID)
		}
	}
	for base, seen := range variants {
		if !seen["text"] || !seen["image"] {
			t.Errorf("base %s missing a variant: %v", base, seen)
		}
	}

	second := t.TempDir()
	report2, err := GenerateEndToEnd(syntheticBook(), second, P0Options{Seed: 42, PageCount: 8})
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if mustRead(t, report.Dataset) != mustRead(t, report2.Dataset) {
		t.Fatal("P0 generator not deterministic for a fixed seed")
	}
}

func TestValidateEndToEndCatchesLeakAndExternalKnowledge(t *testing.T) {
	leak := Case{
		CaseID: "e2e-x-01-text", Stage: StageEndToEnd, BaseID: "e2e-x-01", Variant: "text",
		TaskFamily: "exact", Instruction: "What is the Blackboard Swarm? Answer: Blackboard Swarm",
		PageRefs: []int{12}, GroundTruthAddresses: []string{"ohf://fake/page/12"}, EvidenceCID: "c12",
		EvidenceText: "A Blackboard Swarm coordinates specialists.", Expected: Expected{Value: "Blackboard Swarm"},
	}
	external := Case{
		CaseID: "e2e-x-02-text", Stage: StageEndToEnd, BaseID: "e2e-x-02", Variant: "text",
		TaskFamily: "exact", Instruction: "Who invented the telephone?",
		PageRefs: []int{12}, GroundTruthAddresses: []string{"ohf://fake/page/12"}, EvidenceCID: "c12",
		EvidenceText: "A Blackboard Swarm coordinates specialists.", Expected: Expected{Value: "Alexander Graham Bell"},
	}
	problems := ValidateEndToEnd([]Case{leak, external})
	joined := ""
	for _, problem := range problems {
		joined += problem.Error() + "\n"
	}
	if !containsSubstr(joined, "leaks the answer") || !containsSubstr(joined, "not demonstrable") {
		t.Fatalf("validator missed leak/external-knowledge: %s", joined)
	}
}

func containsSubstr(haystack, needle string) bool { return indexOf(haystack, needle) >= 0 }

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
