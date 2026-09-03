package parrotlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// microisagen builds the parrot-microisa-r0 `microisa_visual` dataset: four
// one-variable sub-experiments (A1 canonical baseline, A2 input-size ladders
// + reference-type block, A3 visual-field curve, A4 real-PDF transfer) that
// each hold every variable at its frozen CANONICAL_MICRO_ISA_CONDITION
// except the one under study. Ladders are structurally nested (one master
// stimulus per base; rungs are deterministic subsets), not merely
// BaseID-matched. Synthetic images are dependency-free; A4 crops come from
// real rendered PDF pages listed in microisa-visual.crops.json.

const (
	miWidth  = 640
	miHeight = 400
)

var (
	miInk   = color.RGBA{20, 20, 20, 255}
	miPaper = color.RGBA{255, 255, 255, 255}
)

// nounBank supplies EXTRACT_ENTITY / SELECT_ONE_OF_N words. Every word uses
// only the conservative unambiguous letter set (letterAlphabet =
// "ACDEFHKMNPRTUVXY"), so a misread cannot be blamed on a confusable glyph.
var nounBank = []string{
	"CART", "DART", "PART", "MART", "PARK", "MARK", "DARK", "CAKE", "RAKE", "FARM",
	"HARM", "YARN", "CANE", "MANE", "PANE", "DANDY", "CRANE", "FRAME", "PARTY", "CANDY",
	"HANDY", "NECTAR", "MATTER", "HAMMER", "PEPPER", "RUNNER", "DANCER", "MARKET", "CARPET", "MARRY",
	"HATTER", "PATTERN", "PARTNER", "MARKER", "PACKER", "TRACKER", "CRACKER", "HARDEN", "CAMERA", "HYDRANT",
}

// fillerWords are non-lexical visual noise for the field-size experiment:
// deterministic pseudo-random groups of safe-alphabet letters, never English
// prose (parrot-microisa-r0 aborted because instruction-like filler was read
// as a prompt). 60 tokens is enough for the widest "page" layout.
var fillerWords = buildFillerTokens()

func buildFillerTokens() []string {
	source := rand.New(rand.NewSource(20260903))
	tokens := make([]string, 60)
	for index := range tokens {
		tokens[index] = randomString(source, letterAlphabet, 3+source.Intn(2))
	}
	return tokens
}

// microISALadders is the frozen A2 ladder definition.
var microISALadders = []struct {
	capability string
	dim        string
	rungs      []int
	bases      int
}{
	{"READ_SHORT_TEXT", "visual_text_chars", []int{2, 4, 8, 16, 32}, 10},
	{"SELECT_ONE_OF_N", "choice_width", []int{2, 3, 4, 6, 8}, 10},
	{"VISUAL_LOCATE", "region_count", []int{2, 4, 6, 9}, 10},
}

var microISAReferenceTypes = []string{"arrow", "label2region", "number2box", "glyph2target"}

const (
	microISARefBases   = 8
	microISAFieldBases = 12
	microISAFieldCaps  = "READ_SHORT_TEXT,EXTRACT_NUMBER,EXTRACT_ENTITY"
)

var microISAFieldSizes = []string{"tight", "medium", "block", "page"}

// a1Sizes is the frozen A1 base-stimulus count per capability.
var a1Sizes = map[string]int{
	"VISUAL_IDENTIFY": 20, "VISUAL_LOCATE": 30, "READ_SHORT_LABEL": 30, "READ_SHORT_TEXT": 30,
	"EXTRACT_NUMBER": 30, "EXTRACT_ENTITY": 30, "SELECT_ONE_OF_N": 20, "FOLLOW_ONE_REFERENCE": 30,
	"COMPARE_TWO_VALUES": 20, "SAME_DIFFERENT": 20,
}

// MicroISAGenReport summarises a generation run.
type MicroISAGenReport struct {
	Dataset         string         `json:"dataset"`
	CasesWritten    int            `json:"cases_written"`
	BaseStimuli     int            `json:"base_stimuli"`
	BySubExperiment map[string]int `json:"by_sub_experiment"`
	A4Exploratory   []string       `json:"a4_exploratory,omitempty"`
	CropSpecFound   bool           `json:"crop_spec_found"`
}

type microISAProvenance struct {
	CaseID        string `json:"case_id"`
	BaseID        string `json:"base_id"`
	Capability    string `json:"capability"`
	SubExperiment string `json:"sub_experiment"`
	Condition     string `json:"condition"`
	VariedDim     string `json:"varied_dim"`
	Source        string `json:"source"`
	Expected      string `json:"expected"`
	Method        string `json:"method"`
}

type microISABuilder struct {
	datasetDir string
	imageDir   string
	seed       int64
	baseSeq    int64
	cases      []Case
	prov       []microISAProvenance
	report     MicroISAGenReport
}

// GenerateMicroISAVisual writes datasets/microisa-visual.jsonl, the rendered
// images and datasets/microisa-visual.provenance.json. It refuses to
// overwrite an existing dataset unless force is set.
func GenerateMicroISAVisual(datasetDir string, seed int64, force bool) (MicroISAGenReport, error) {
	datasetFile := filepath.Join(datasetDir, "microisa-visual.jsonl")
	if _, err := os.Stat(datasetFile); err == nil && !force {
		return MicroISAGenReport{}, fmt.Errorf("%s exists; pass --force to regenerate", datasetFile)
	}
	imageDir := filepath.Join(datasetDir, "microisa-visual", "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return MicroISAGenReport{}, err
	}
	builder := &microISABuilder{
		datasetDir: datasetDir,
		imageDir:   imageDir,
		seed:       seed,
		report:     MicroISAGenReport{BySubExperiment: map[string]int{}},
	}

	builder.buildA1()
	builder.buildA2Ladders()
	builder.buildA2References()
	builder.buildA3Field()
	if err := builder.buildA4Transfer(); err != nil {
		return MicroISAGenReport{}, err
	}

	if problems := Validate(builder.cases); len(problems) > 0 {
		return MicroISAGenReport{}, fmt.Errorf("generated dataset invalid: %d problem(s); first: %v", len(problems), problems[0])
	}

	var jsonl bytes.Buffer
	encoder := json.NewEncoder(&jsonl)
	for _, item := range builder.cases {
		if err := encoder.Encode(item); err != nil {
			return MicroISAGenReport{}, err
		}
	}
	if err := os.WriteFile(datasetFile, jsonl.Bytes(), 0o644); err != nil {
		return MicroISAGenReport{}, err
	}
	provBytes, _ := json.MarshalIndent(builder.prov, "", "  ")
	if err := os.WriteFile(filepath.Join(datasetDir, "microisa-visual.provenance.json"), append(provBytes, '\n'), 0o644); err != nil {
		return MicroISAGenReport{}, err
	}

	builder.report.Dataset = datasetFile
	builder.report.CasesWritten = len(builder.cases)
	baseSet := map[string]bool{}
	for _, item := range builder.cases {
		key := item.BaseID
		if key == "" {
			key = item.CaseID
		}
		baseSet[item.SubExperiment+"/"+key] = true
	}
	builder.report.BaseStimuli = len(baseSet)
	return builder.report, nil
}

func (builder *microISABuilder) rng() *rand.Rand {
	source := rand.New(rand.NewSource(builder.seed + builder.baseSeq))
	builder.baseSeq++
	return source
}

func (builder *microISABuilder) emit(item Case, expected, method string) {
	builder.cases = append(builder.cases, item)
	builder.prov = append(builder.prov, microISAProvenance{
		CaseID: item.CaseID, BaseID: item.BaseID, Capability: item.Capabilities[0],
		SubExperiment: item.SubExperiment, Condition: item.Condition, VariedDim: item.VariedDim,
		Source: item.Source, Expected: expected, Method: method,
	})
	builder.report.BySubExperiment[item.SubExperiment]++
}

func (builder *microISABuilder) writeImage(caseID string, data []byte) string {
	rel := filepath.ToSlash(filepath.Join("microisa-visual", "images", caseID+".png"))
	_ = os.WriteFile(filepath.Join(builder.datasetDir, filepath.FromSlash(rel)), data, 0o644)
	return rel
}

func newCanvas(width, height int) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(canvas, miPaper)
	return canvas
}

func encodePNG(canvas *image.RGBA) []byte {
	var out bytes.Buffer
	_ = png.Encode(&out, canvas)
	return out.Bytes()
}

func slug(capability string) string { return strings.ToLower(capability) }

func randomString(source *rand.Rand, alphabet []rune, length int) string {
	runes := make([]rune, length)
	for index := range runes {
		runes[index] = alphabet[source.Intn(len(alphabet))]
	}
	return string(runes)
}

// ---------------------------------------------------------------------------
// A1 — canonical atomic baseline (one observation per independent base stimulus)
// ---------------------------------------------------------------------------

func (builder *microISABuilder) buildA1() {
	for _, capability := range MicroISACapabilities {
		count := a1Sizes[capability]
		for index := 0; index < count; index++ {
			source := builder.rng()
			caseID := fmt.Sprintf("mi-a1-%s-%02d", slug(capability), index+1)
			item := Case{
				CaseID: caseID, Stage: StageMicroISAVisual, Capabilities: []string{capability},
				BaseID: caseID, SubExperiment: "A1", Condition: "canonical", Source: "synthetic",
			}
			image, expected := builder.renderCanonical(capability, source, &item)
			item.ImagePath = builder.writeImage(caseID, image)
			builder.emit(item, expected, "synthetic-r0")
		}
	}
}

// renderCanonical draws capability's frozen canonical condition and fills the
// task fields on item, returning the PNG and a printable expected answer.
func (builder *microISABuilder) renderCanonical(capability string, source *rand.Rand, item *Case) ([]byte, string) {
	switch capability {
	case "VISUAL_IDENTIFY":
		return builder.renderIdentify(source, item)
	case "VISUAL_LOCATE":
		return builder.renderLocate(source, item, 4)
	case "READ_SHORT_LABEL":
		return builder.renderShortLabel(source, item)
	case "READ_SHORT_TEXT":
		master := randomString(source, letterAlphabet, 32)
		return builder.renderShortText(item, master, 8, "medium")
	case "EXTRACT_NUMBER":
		return builder.renderExtractNumber(source, item, "medium")
	case "EXTRACT_ENTITY":
		return builder.renderExtractEntity(source, item, "medium")
	case "SELECT_ONE_OF_N":
		master := builder.pickWords(source, 8)
		return builder.renderSelect(item, master, 4)
	case "FOLLOW_ONE_REFERENCE":
		return builder.renderReference(source, item, "arrow")
	case "COMPARE_TWO_VALUES":
		return builder.renderCompare(source, item)
	case "SAME_DIFFERENT":
		return builder.renderSameDifferent(source, item)
	}
	return encodePNG(newCanvas(miWidth, miHeight)), ""
}

func (builder *microISABuilder) renderIdentify(source *rand.Rand, item *Case) ([]byte, string) {
	order := source.Perm(len(scenePalette))
	circleColour := order[0]
	squareColour := order[1]
	canvas := newCanvas(miWidth, miHeight)
	drawDisc(canvas, image.Point{X: 200, Y: 200}, 70, scenePalette[circleColour].rgb)
	drawSquare(canvas, image.Point{X: 440, Y: 200}, 130, scenePalette[squareColour].rgb)
	item.TaskFamily = "choice"
	item.Choices = paletteWords()
	item.Expected = Expected{Value: scenePalette[circleColour].word}
	item.Instruction = "What colour is the circle? Answer with exactly one of: red, green, blue, orange."
	return encodePNG(canvas), scenePalette[circleColour].word
}

func paletteWords() []string {
	words := make([]string, len(scenePalette))
	for index, entry := range scenePalette {
		words[index] = entry.word
	}
	return words
}

var locatePositionNames = map[int][]string{
	2: {"left", "right"},
	4: {"top-left", "top-right", "bottom-left", "bottom-right"},
	6: {"top-left", "top-middle", "top-right", "bottom-left", "bottom-middle", "bottom-right"},
	9: {"top-left", "top-middle", "top-right", "middle-left", "center", "middle-right", "bottom-left", "bottom-middle", "bottom-right"},
}

func locateGridShape(regions int) (cols, rows int) {
	switch regions {
	case 2:
		return 2, 1
	case 4:
		return 2, 2
	case 6:
		return 3, 2
	default:
		return 3, 3
	}
}

// renderLocate places a marker at a fractional position (carried on item via
// a stashed fraction) and overlays a grid of `regions` cells; the answer is
// the position name of the cell containing the marker. Nested across the A2
// ladder: the same fractional position is reused for every rung.
func (builder *microISABuilder) renderLocate(source *rand.Rand, item *Case, regions int) ([]byte, string) {
	fx, fy := builder.locateFraction(source, item)
	cols, rows := locateGridShape(regions)
	canvas := newCanvas(miWidth, miHeight)
	marginX, marginY := 40, 40
	gridW, gridH := miWidth-2*marginX, miHeight-2*marginY
	for column := 0; column <= cols; column++ {
		x := marginX + column*gridW/cols
		drawLine(canvas, image.Point{X: x, Y: marginY}, image.Point{X: x, Y: marginY + gridH}, miInk)
	}
	for row := 0; row <= rows; row++ {
		y := marginY + row*gridH/rows
		drawLine(canvas, image.Point{X: marginX, Y: y}, image.Point{X: marginX + gridW, Y: y}, miInk)
	}
	column := int(fx * float64(cols))
	row := int(fy * float64(rows))
	if column >= cols {
		column = cols - 1
	}
	if row >= rows {
		row = rows - 1
	}
	// Draw the marker at the cell centre so it is unambiguous at every rung.
	markerX := marginX + (2*column+1)*gridW/(2*cols)
	markerY := marginY + (2*row+1)*gridH/(2*rows)
	drawDisc(canvas, image.Point{X: markerX, Y: markerY}, 16, scenePalette[2].rgb)
	names := locatePositionNames[regions]
	answer := names[row*cols+column]
	item.TaskFamily = "choice"
	item.Choices = append([]string(nil), names...)
	item.Expected = Expected{Value: answer, Aliases: []string{strings.ReplaceAll(answer, "-", " "), strings.ReplaceAll(answer, "-", "")}}
	item.Instruction = "A grid is drawn over the image with one blue dot. Which cell contains the dot? Answer with one of: " + strings.Join(names, ", ") + "."
	return encodePNG(canvas), answer
}

// locateFraction returns a stable fractional marker position for item's base
// stimulus, seeded so A1 and every A2 rung of the same base agree.
func (builder *microISABuilder) locateFraction(source *rand.Rand, item *Case) (float64, float64) {
	// Keep the marker away from grid lines for the coarsest and finest grids.
	choicesX := []float64{0.16, 0.5, 0.83}
	choicesY := []float64{0.16, 0.5, 0.83}
	return choicesX[source.Intn(len(choicesX))], choicesY[source.Intn(len(choicesY))]
}

func (builder *microISABuilder) renderShortLabel(source *rand.Rand, item *Case) ([]byte, string) {
	length := 2
	text := randomString(source, letterAlphabet, length)
	canvas := newCanvas(miWidth, miHeight)
	scale := 8
	width := textPixelWidth(text, scale)
	originX := (miWidth - width) / 2
	originY := (miHeight - glyphHeight*scale) / 2
	drawText(canvas, text, image.Point{X: originX, Y: originY}, scale, miInk)
	box := image.Rect(originX-24, originY-24, originX+width+24, originY+glyphHeight*scale+24)
	drawRectOutline(canvas, box, 4, miInk)
	item.TaskFamily = "entity"
	item.Expected = Expected{Value: text}
	item.Instruction = "Read the characters inside the box. Answer with just those characters."
	return encodePNG(canvas), text
}

// renderShortText renders the first `chars` characters of master at a given
// field size. Used by A1 (chars=8, field=medium), A2 ladder (varying chars,
// field=medium) and A3 (chars=8, varying field).
func (builder *microISABuilder) renderShortText(item *Case, master string, chars int, field string) ([]byte, string) {
	if chars > len(master) {
		chars = len(master)
	}
	text := master[:chars]
	scale := 3
	draw := func(canvas *image.RGBA, area image.Rectangle) {
		width := textPixelWidth(text, scale)
		originX := area.Min.X + (area.Dx()-width)/2
		originY := area.Min.Y + (area.Dy()-glyphHeight*scale)/2
		drawText(canvas, text, image.Point{X: originX, Y: originY}, scale, miInk)
	}
	canvas := composeField(field, textPixelWidth(text, scale), draw)
	item.TaskFamily = "entity"
	item.Expected = Expected{Value: text}
	item.Instruction = "What text is shown in the image? Answer with just that text."
	return encodePNG(canvas), text
}

func (builder *microISABuilder) renderExtractNumber(source *rand.Rand, item *Case, field string) ([]byte, string) {
	number := randomString(source, digitAlphabet, 3)
	distractorA := randomString(source, digitAlphabet, 3)
	distractorB := randomString(source, digitAlphabet, 2)
	scale := 4
	draw := func(canvas *image.RGBA, area image.Rectangle) {
		centerX := area.Min.X + area.Dx()/2
		centerY := area.Min.Y + area.Dy()/2
		width := textPixelWidth(number, scale)
		originX := centerX - width/2
		originY := centerY - glyphHeight*scale/2
		drawText(canvas, number, image.Point{X: originX, Y: originY}, scale, miInk)
		drawRectOutline(canvas, image.Rect(originX-18, originY-18, originX+width+18, originY+glyphHeight*scale+18), 3, miInk)
		if area.Dy() > 90 {
			drawText(canvas, distractorA, image.Point{X: area.Min.X + 12, Y: area.Min.Y + 10}, 2, miInk)
			drawText(canvas, distractorB, image.Point{X: area.Max.X - 90, Y: area.Max.Y - 24}, 2, miInk)
		}
	}
	canvas := composeField(field, textPixelWidth(number, scale)+40, draw)
	item.TaskFamily = "numeric"
	value := float64(mustParseDigits(number))
	item.Expected = Expected{Number: &value}
	item.Instruction = "What number is inside the outlined box? Answer with just the number."
	return encodePNG(canvas), number
}

func (builder *microISABuilder) renderExtractEntity(source *rand.Rand, item *Case, field string) ([]byte, string) {
	word := nounBank[source.Intn(len(nounBank))]
	other := nounBank[source.Intn(len(nounBank))]
	scale := 3
	draw := func(canvas *image.RGBA, area image.Rectangle) {
		centerX := area.Min.X + area.Dx()/2
		centerY := area.Min.Y + area.Dy()/2
		width := textPixelWidth(word, scale)
		originX := centerX - width/2
		originY := centerY - glyphHeight*scale/2
		drawText(canvas, word, image.Point{X: originX, Y: originY}, scale, miInk)
		drawRectOutline(canvas, image.Rect(originX-16, originY-14, originX+width+16, originY+glyphHeight*scale+14), 3, miInk)
		if area.Dy() > 90 {
			drawText(canvas, other, image.Point{X: area.Min.X + 12, Y: area.Min.Y + 10}, 2, miInk)
		}
	}
	canvas := composeField(field, textPixelWidth(word, scale)+40, draw)
	item.TaskFamily = "entity"
	item.Expected = Expected{Value: word}
	item.Instruction = "What word is inside the outlined box? Answer with just that word."
	return encodePNG(canvas), word
}

func (builder *microISABuilder) pickWords(source *rand.Rand, count int) []string {
	order := source.Perm(len(nounBank))
	words := make([]string, count)
	for index := 0; index < count; index++ {
		words[index] = nounBank[order[index]]
	}
	return words
}

// renderSelect lists the first `width` words of master and boxes master[0]
// (the target is present at every rung; larger rungs only add distractors).
func (builder *microISABuilder) renderSelect(item *Case, master []string, width int) ([]byte, string) {
	if width > len(master) {
		width = len(master)
	}
	options := append([]string(nil), master[:width]...)
	target := master[0]
	canvas := newCanvas(miWidth, miHeight)
	scale := 4
	lineHeight := glyphHeight*scale + 22
	startY := (miHeight - lineHeight*width) / 2
	for index, word := range options {
		originY := startY + index*lineHeight
		drawText(canvas, word, image.Point{X: 120, Y: originY}, scale, miInk)
		if word == target {
			drawRectOutline(canvas, image.Rect(104, originY-10, 104+textPixelWidth(word, scale)+24, originY+glyphHeight*scale+10), 3, miInk)
		}
	}
	item.TaskFamily = "choice"
	item.Choices = options
	item.Expected = Expected{Value: target}
	item.Instruction = "Several words are listed. Which word is inside the box? Answer with that word."
	return encodePNG(canvas), target
}

func (builder *microISABuilder) renderReference(source *rand.Rand, item *Case, refType string) ([]byte, string) {
	labels := []string{"A", "B", "C"}
	targetIndex := source.Intn(3)
	canvas := newCanvas(miWidth, miHeight)
	positions := []image.Point{{X: 120, Y: 90}, {X: 320, Y: 300}, {X: 540, Y: 110}}
	for index, label := range labels {
		point := positions[index]
		drawRectOutline(canvas, image.Rect(point.X-30, point.Y-30, point.X+30, point.Y+30), 3, miInk)
		drawText(canvas, label, image.Point{X: point.X - glyphWidth*3/2*3, Y: point.Y - glyphHeight*3/2}, 3, miInk)
	}
	answer := labels[targetIndex]
	switch refType {
	case "arrow":
		origin := image.Point{X: 320, Y: 40}
		tip := positions[targetIndex]
		drawLine(canvas, origin, tip, scenePalette[0].rgb)
		drawLine(canvas, image.Point{X: origin.X + 1, Y: origin.Y}, tip, scenePalette[0].rgb)
		drawLine(canvas, tip, image.Point{X: tip.X - 12, Y: tip.Y - 14}, scenePalette[0].rgb)
		drawLine(canvas, tip, image.Point{X: tip.X + 12, Y: tip.Y - 14}, scenePalette[0].rgb)
		item.Instruction = "A red arrow starts near the top and points to one target box. Which target (A, B or C) does it point to?"
	case "label2region":
		drawText(canvas, "GO TO BOX "+answer, image.Point{X: 40, Y: 360}, 3, miInk)
		item.Instruction = "Read the instruction text in the image and answer with the target letter it names (A, B or C)."
	case "number2box":
		numbered := map[int]string{0: "2", 1: "3", 2: "4"}
		for index, point := range positions {
			drawText(canvas, numbered[index], image.Point{X: point.X - 6, Y: point.Y + 40}, 2, miInk)
		}
		drawText(canvas, "TARGET NUMBER "+map[string]string{"A": "2", "B": "3", "C": "4"}[answer], image.Point{X: 40, Y: 360}, 3, miInk)
		item.Instruction = "Each target has a number beneath it. Read the target-number instruction and answer with the letter (A, B or C) of that target."
	default: // glyph2target
		drawText(canvas, "KEY  X EQUALS "+answer, image.Point{X: 40, Y: 350}, 3, miInk)
		drawText(canvas, "X", image.Point{X: 300, Y: 40}, 5, scenePalette[0].rgb)
		item.Instruction = "Use the KEY line in the image to resolve the large red symbol to a target letter (A, B or C)."
	}
	item.TaskFamily = "choice"
	item.Choices = labels
	item.Expected = Expected{Value: answer}
	return encodePNG(canvas), answer
}

func (builder *microISABuilder) renderCompare(source *rand.Rand, item *Case) ([]byte, string) {
	left := twoLegibleDigits(source)
	right := twoLegibleDigits(source)
	for right == left {
		right = twoLegibleDigits(source)
	}
	canvas := newCanvas(miWidth, miHeight)
	drawText(canvas, fmt.Sprintf("%d", left), image.Point{X: 130, Y: 170}, 6, miInk)
	drawText(canvas, fmt.Sprintf("%d", right), image.Point{X: 420, Y: 170}, 6, miInk)
	larger := left
	if right > left {
		larger = right
	}
	value := float64(larger)
	item.TaskFamily = "numeric"
	item.Expected = Expected{Number: &value}
	item.Instruction = "Two numbers are shown. What is the larger number? Answer with just that number."
	return encodePNG(canvas), fmt.Sprintf("%d", larger)
}

func (builder *microISABuilder) renderSameDifferent(source *rand.Rand, item *Case) ([]byte, string) {
	order := source.Perm(len(scenePalette))
	same := source.Intn(2) == 0
	leftColour := order[0]
	rightColour := order[0]
	if !same {
		rightColour = order[1]
	}
	canvas := newCanvas(miWidth, miHeight)
	drawSquare(canvas, image.Point{X: 190, Y: 200}, 130, scenePalette[leftColour].rgb)
	drawSquare(canvas, image.Point{X: 450, Y: 200}, 130, scenePalette[rightColour].rgb)
	answer := "no"
	if same {
		answer = "yes"
	}
	item.TaskFamily = "choice"
	item.Choices = []string{"yes", "no"}
	item.Expected = Expected{Value: answer}
	item.Instruction = "Are the two squares the same colour? Answer yes or no."
	return encodePNG(canvas), answer
}

// composeField wraps a draw callback in a canvas whose size / surrounding
// filler encodes the field size (tight, medium, block, page).
func composeField(field string, evidenceWidth int, draw func(canvas *image.RGBA, area image.Rectangle)) *image.RGBA {
	switch field {
	case "tight":
		width := evidenceWidth + 32
		if width < 96 {
			width = 96
		}
		canvas := newCanvas(width, 72)
		draw(canvas, canvas.Bounds())
		return canvas
	case "medium":
		canvas := newCanvas(miWidth, 150)
		drawText(canvas, fillerLine(0), image.Point{X: 20, Y: 12}, 2, miInk)
		drawText(canvas, fillerLine(1), image.Point{X: 20, Y: 120}, 2, miInk)
		draw(canvas, image.Rect(0, 34, miWidth, 108))
		return canvas
	case "block":
		canvas := newCanvas(miWidth, 300)
		for line := 0; line < 4; line++ {
			drawText(canvas, fillerLine(line), image.Point{X: 20, Y: 14 + line*24}, 2, miInk)
		}
		draw(canvas, image.Rect(0, 120, miWidth, 190))
		for line := 0; line < 4; line++ {
			drawText(canvas, fillerLine(line+4), image.Point{X: 20, Y: 210 + line*24}, 2, miInk)
		}
		return canvas
	default: // page
		canvas := newCanvas(miWidth, miHeight)
		for line := 0; line < 7; line++ {
			drawText(canvas, fillerLine(line), image.Point{X: 20, Y: 14 + line*24}, 2, miInk)
		}
		draw(canvas, image.Rect(0, 190, miWidth, 260))
		for line := 0; line < 5; line++ {
			drawText(canvas, fillerLine(line+7), image.Point{X: 20, Y: 280 + line*24}, 2, miInk)
		}
		return canvas
	}
}

func fillerLine(index int) string {
	if len(fillerWords) == 0 {
		return ""
	}
	start := (index * 9) % len(fillerWords)
	end := start + 9
	if end > len(fillerWords) {
		end = len(fillerWords)
	}
	return strings.Join(fillerWords[start:end], " ")
}

// twoLegibleDigits returns a two-digit number whose digits are all in the
// restricted digit alphabet {2,3,4,5,6,7,9}.
func twoLegibleDigits(source *rand.Rand) int {
	digit := func() int { return int(digitAlphabet[source.Intn(len(digitAlphabet))] - '0') }
	return digit()*10 + digit()
}

func mustParseDigits(text string) int {
	value := 0
	for _, symbol := range text {
		value = value*10 + int(symbol-'0')
	}
	return value
}

// ---------------------------------------------------------------------------
// A2 — input-size ladders (nested; only the ladder dimension varies)
// ---------------------------------------------------------------------------

func (builder *microISABuilder) buildA2Ladders() {
	for _, ladder := range microISALadders {
		for index := 0; index < ladder.bases; index++ {
			source := builder.rng()
			baseID := fmt.Sprintf("mi-a2-%s-%02d", ladder.dim, index+1)
			switch ladder.capability {
			case "READ_SHORT_TEXT":
				master := randomString(source, letterAlphabet, 32)
				for _, chars := range ladder.rungs {
					item := builder.newLadderCase(ladder.capability, baseID, ladder.dim, fmt.Sprintf("chars=%d", chars))
					image, expected := builder.renderShortText(&item, master, chars, "medium")
					item.ImagePath = builder.writeImage(item.CaseID, image)
					builder.emit(item, expected, "synthetic-r0")
				}
			case "SELECT_ONE_OF_N":
				master := builder.pickWords(source, 8)
				for _, width := range ladder.rungs {
					item := builder.newLadderCase(ladder.capability, baseID, ladder.dim, fmt.Sprintf("width=%d", width))
					image, expected := builder.renderSelect(&item, master, width)
					item.ImagePath = builder.writeImage(item.CaseID, image)
					builder.emit(item, expected, "synthetic-r0")
				}
			case "VISUAL_LOCATE":
				// One master fractional position, reused at every rung.
				fixed := rand.New(rand.NewSource(builder.seed + builder.baseSeq + int64(1<<20)))
				fx, fy := builder.locateFraction(fixed, &Case{})
				for _, regions := range ladder.rungs {
					item := builder.newLadderCase(ladder.capability, baseID, ladder.dim, fmt.Sprintf("regions=%d", regions))
					image, expected := builder.renderLocateFixed(&item, regions, fx, fy)
					item.ImagePath = builder.writeImage(item.CaseID, image)
					builder.emit(item, expected, "synthetic-r0")
				}
			}
		}
	}
}

func (builder *microISABuilder) newLadderCase(capability, baseID, dim, condition string) Case {
	return Case{
		CaseID:        fmt.Sprintf("%s-%s", baseID, strings.ReplaceAll(condition, "=", "")),
		Stage:         StageMicroISAVisual,
		Capabilities:  []string{capability},
		BaseID:        baseID,
		SubExperiment: "A2",
		Condition:     condition,
		VariedDim:     dim,
		Source:        "synthetic",
	}
}

// renderLocateFixed is renderLocate with an externally supplied fractional
// marker position (so every ladder rung of one base stimulus is nested).
func (builder *microISABuilder) renderLocateFixed(item *Case, regions int, fx, fy float64) ([]byte, string) {
	cols, rows := locateGridShape(regions)
	canvas := newCanvas(miWidth, miHeight)
	marginX, marginY := 40, 40
	gridW, gridH := miWidth-2*marginX, miHeight-2*marginY
	for column := 0; column <= cols; column++ {
		x := marginX + column*gridW/cols
		drawLine(canvas, image.Point{X: x, Y: marginY}, image.Point{X: x, Y: marginY + gridH}, miInk)
	}
	for row := 0; row <= rows; row++ {
		y := marginY + row*gridH/rows
		drawLine(canvas, image.Point{X: marginX, Y: y}, image.Point{X: marginX + gridW, Y: y}, miInk)
	}
	column := int(fx * float64(cols))
	row := int(fy * float64(rows))
	if column >= cols {
		column = cols - 1
	}
	if row >= rows {
		row = rows - 1
	}
	drawDisc(canvas, image.Point{X: marginX + (2*column+1)*gridW/(2*cols), Y: marginY + (2*row+1)*gridH/(2*rows)}, 16, scenePalette[2].rgb)
	names := locatePositionNames[regions]
	answer := names[row*cols+column]
	item.TaskFamily = "choice"
	item.Choices = append([]string(nil), names...)
	item.Expected = Expected{Value: answer, Aliases: []string{strings.ReplaceAll(answer, "-", " "), strings.ReplaceAll(answer, "-", "")}}
	item.Instruction = "A grid is drawn over the image with one blue dot. Which cell contains the dot? Answer with one of: " + strings.Join(names, ", ") + "."
	return encodePNG(canvas), answer
}

// ---------------------------------------------------------------------------
// A2-ref — FOLLOW_ONE_REFERENCE by reference type (categorical, no max rung)
// ---------------------------------------------------------------------------

func (builder *microISABuilder) buildA2References() {
	for index := 0; index < microISARefBases; index++ {
		source := builder.rng()
		targetIndex := source.Intn(3)
		baseID := fmt.Sprintf("mi-a2ref-%02d", index+1)
		for _, refType := range microISAReferenceTypes {
			// Reseed per type from the base's target so the resolved target
			// letter is identical across all four reference types (nested).
			typed := rand.New(rand.NewSource(int64(targetIndex)))
			_ = typed
			item := Case{
				CaseID:        fmt.Sprintf("%s-%s", baseID, refType),
				Stage:         StageMicroISAVisual,
				Capabilities:  []string{"FOLLOW_ONE_REFERENCE"},
				BaseID:        baseID,
				SubExperiment: "A2",
				Condition:     "reftype=" + refType,
				VariedDim:     "reference_type",
				Source:        "synthetic",
			}
			image, expected := builder.renderReferenceTarget(item.Capabilities[0], &item, refType, targetIndex)
			item.ImagePath = builder.writeImage(item.CaseID, image)
			builder.emit(item, expected, "synthetic-r0")
		}
	}
}

// renderReferenceTarget is renderReference with a fixed target index.
func (builder *microISABuilder) renderReferenceTarget(_ string, item *Case, refType string, targetIndex int) ([]byte, string) {
	stub := rand.New(rand.NewSource(0))
	_ = stub
	// Reuse renderReference's drawing by temporarily forcing the target.
	labels := []string{"A", "B", "C"}
	answer := labels[targetIndex]
	// Build a deterministic single-value source: Intn(3) must return targetIndex.
	image, _ := builder.renderReference(fixedIntnSource(targetIndex), item, refType)
	item.Expected = Expected{Value: answer}
	return image, answer
}

// fixedIntnSource returns a *rand.Rand whose first Intn(3) yields value; good
// enough for the single draw renderReference makes.
func fixedIntnSource(value int) *rand.Rand {
	for seed := int64(0); seed < 64; seed++ {
		candidate := rand.New(rand.NewSource(seed))
		if candidate.Intn(3) == value {
			return rand.New(rand.NewSource(seed))
		}
	}
	return rand.New(rand.NewSource(0))
}

// ---------------------------------------------------------------------------
// A3 — visual-field curve (content fixed at canonical; only field_size varies)
// ---------------------------------------------------------------------------

func (builder *microISABuilder) buildA3Field() {
	for _, capability := range strings.Split(microISAFieldCaps, ",") {
		for index := 0; index < microISAFieldBases; index++ {
			source := builder.rng()
			baseID := fmt.Sprintf("mi-a3-%s-%02d", slug(capability), index+1)
			var master string
			if capability == "READ_SHORT_TEXT" {
				master = randomString(source, letterAlphabet, 32)
			}
			for _, field := range microISAFieldSizes {
				item := Case{
					CaseID:        fmt.Sprintf("%s-%s", baseID, field),
					Stage:         StageMicroISAVisual,
					Capabilities:  []string{capability},
					BaseID:        baseID,
					SubExperiment: "A3",
					Condition:     "field=" + field,
					VariedDim:     "field_size",
					Source:        "synthetic",
				}
				var image []byte
				var expected string
				switch capability {
				case "READ_SHORT_TEXT":
					image, expected = builder.renderShortText(&item, master, 8, field)
				case "EXTRACT_NUMBER":
					fixed := rand.New(rand.NewSource(builder.seed + int64(index) + int64(1<<24)))
					image, expected = builder.renderExtractNumber(fixed, &item, field)
				case "EXTRACT_ENTITY":
					fixed := rand.New(rand.NewSource(builder.seed + int64(index) + int64(1<<25)))
					image, expected = builder.renderExtractEntity(fixed, &item, field)
				}
				item.ImagePath = builder.writeImage(item.CaseID, image)
				builder.emit(item, expected, "synthetic-r0")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// A4 — real-PDF transfer (crops from rendered pages; microisa-visual.crops.json)
// ---------------------------------------------------------------------------

type cropCondition struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Rect   []int  `json:"rect"` // x,y,w,h; nil/empty = whole page
}

type cropSpec struct {
	BaseID     string          `json:"base_id"`
	Capability string          `json:"capability"`
	Page       string          `json:"page"` // path relative to datasets/
	Family     string          `json:"family"`
	Question   string          `json:"question"`
	Expected   string          `json:"expected"`
	Number     *float64        `json:"number,omitempty"`
	Conditions []cropCondition `json:"conditions"`
}

func (builder *microISABuilder) buildA4Transfer() error {
	specPath := filepath.Join(builder.datasetDir, "microisa-visual.crops.json")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		builder.report.CropSpecFound = false
		return nil
	}
	builder.report.CropSpecFound = true
	var specs []cropSpec
	if err := json.Unmarshal(raw, &specs); err != nil {
		return fmt.Errorf("microisa-visual.crops.json: %w", err)
	}

	perCap := map[string]int{}
	for _, spec := range specs {
		pagePath := filepath.Join(builder.datasetDir, filepath.FromSlash(spec.Page))
		pageFile, err := os.Open(pagePath)
		if err != nil {
			return fmt.Errorf("crop base %s: %w", spec.BaseID, err)
		}
		pageImage, _, decodeErr := image.Decode(pageFile)
		pageFile.Close()
		if decodeErr != nil {
			return fmt.Errorf("crop base %s: %w", spec.BaseID, decodeErr)
		}
		perCap[spec.Capability]++
		for _, condition := range spec.Conditions {
			item := Case{
				CaseID:        fmt.Sprintf("%s-%s", spec.BaseID, condition.Name),
				Stage:         StageMicroISAVisual,
				Capabilities:  []string{spec.Capability},
				BaseID:        spec.BaseID,
				SubExperiment: "A4",
				Condition:     "crop=" + condition.Name,
				VariedDim:     "crop_extent",
				Source:        condition.Source,
				TaskFamily:    spec.Family,
				Instruction:   spec.Question,
			}
			if spec.Family == "numeric" {
				item.Expected = Expected{Number: spec.Number}
			} else {
				item.Expected = Expected{Value: spec.Expected}
			}
			cropped := cropImage(pageImage, condition.Rect)
			buffer := &bytes.Buffer{}
			_ = png.Encode(buffer, cropped)
			item.ImagePath = builder.writeImage(item.CaseID, buffer.Bytes())
			builder.emit(item, spec.Expected, "real-crop-r0")
		}
	}

	names := make([]string, 0, len(perCap))
	for name := range perCap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if perCap[name] < 8 {
			builder.report.A4Exploratory = append(builder.report.A4Exploratory,
				fmt.Sprintf("%s: %d real-PDF base stimuli (<8, transfer result is exploratory)", name, perCap[name]))
		}
	}
	return nil
}

func cropImage(source image.Image, rect []int) image.Image {
	bounds := source.Bounds()
	if len(rect) != 4 {
		return source
	}
	area := image.Rect(rect[0], rect[1], rect[0]+rect[2], rect[1]+rect[3]).Intersect(bounds)
	if area.Empty() {
		return source
	}
	out := image.NewRGBA(image.Rect(0, 0, area.Dx(), area.Dy()))
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			out.Set(x-area.Min.X, y-area.Min.Y, source.At(x, y))
		}
	}
	return out
}
