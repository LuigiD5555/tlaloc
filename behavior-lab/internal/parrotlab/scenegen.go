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

// The instruction-cliff generator (P-1: "the dataset can be produced fully
// deterministically"). One deterministic scene generator + renderer + a
// counterbalanced 5-primitive pipeline. Depth d asks for d steps in order
// and scores only the final step's result; which primitive lands on the
// final step is rotated across four scene families by a Latin square, so a
// collapse at depth N can be separated from one weak capability.

type primitive int

const (
	primIdentify primitive = iota
	primLocate
	primCompare
	primReference
	primFormat
)

func (p primitive) name() string {
	return [...]string{"identify", "locate", "compare", "reference", "format"}[p]
}

func (p primitive) capability() string {
	return [...]string{
		"VISUAL_IDENTIFY", "VISUAL_LOCATE", "COMPARE_SIMPLE", "FOLLOW_REFERENCE", "",
	}[p]
}

func (p primitive) clause() string {
	return [...]string{
		"determine the colour of the circle",
		"determine which shape (circle or square) is on the left",
		"determine which shape (circle or square) is larger",
		"read the letter label printed under the larger shape",
		`output the letter label under the larger shape as compact JSON of the form {"label":"A"}`,
	}[p]
}

// latinSquare[family] is a cyclic left-rotation of the five primitives.
// With five families every primitive lands on every depth position exactly
// once per family cycle, so across 40 scenes each (primitive, depth) cell
// gets 8 samples — the counterbalance that separates a depth effect from a
// single weak capability (P-1 fix #4).
var latinSquare = [5][5]primitive{
	{primIdentify, primLocate, primCompare, primReference, primFormat},
	{primLocate, primCompare, primReference, primFormat, primIdentify},
	{primCompare, primReference, primFormat, primIdentify, primLocate},
	{primReference, primFormat, primIdentify, primLocate, primCompare},
	{primFormat, primIdentify, primLocate, primCompare, primReference},
}

var scenePalette = []struct {
	word string
	rgb  color.RGBA
}{
	{"red", color.RGBA{214, 46, 46, 255}},
	{"green", color.RGBA{40, 158, 66, 255}},
	{"blue", color.RGBA{52, 96, 214, 255}},
	{"orange", color.RGBA{230, 140, 22, 255}},
}

type scene struct {
	index          int
	family         int
	circleColor    int // palette index
	squareColor    int
	circleDiameter int
	squareSide     int
	circleOnLeft   bool
	circleLabel    string // "A" or "B"
	squareLabel    string
}

func (sc scene) leftShape() string {
	if sc.circleOnLeft {
		return "circle"
	}
	return "square"
}

func (sc scene) largerShape() string {
	if sc.circleDiameter > sc.squareSide {
		return "circle"
	}
	return "square"
}

func (sc scene) largerLabel() string {
	if sc.largerShape() == "circle" {
		return sc.circleLabel
	}
	return sc.squareLabel
}

func newScene(index int, source *rand.Rand) scene {
	sizes := []int{60, 75, 90, 105}
	source.Shuffle(len(sizes), func(i, j int) { sizes[i], sizes[j] = sizes[j], sizes[i] })
	colours := []int{0, 1, 2, 3}
	source.Shuffle(len(colours), func(i, j int) { colours[i], colours[j] = colours[j], colours[i] })

	circleLabel := "A"
	squareLabel := "B"
	if source.Intn(2) == 0 {
		circleLabel, squareLabel = "B", "A"
	}
	return scene{
		index:          index,
		family:         index % 5,
		circleColor:    colours[0],
		squareColor:    colours[1],
		circleDiameter: sizes[0],
		squareSide:     sizes[1],
		circleOnLeft:   source.Intn(2) == 0,
		circleLabel:    circleLabel,
		squareLabel:    squareLabel,
	}
}

// answerFor returns the canonical answer, the answer universe, and the task
// family for one primitive applied to a scene.
func (sc scene) answerFor(prim primitive) (value string, choices []string, family string) {
	switch prim {
	case primIdentify:
		value = scenePalette[sc.circleColor].word
		for _, entry := range scenePalette {
			choices = append(choices, entry.word)
		}
		return value, choices, "choice"
	case primLocate:
		return sc.leftShape(), []string{"circle", "square"}, "choice"
	case primCompare:
		return sc.largerShape(), []string{"circle", "square"}, "choice"
	case primReference:
		return sc.largerLabel(), []string{"A", "B"}, "choice"
	default: // primFormat
		return fmt.Sprintf(`{"label":"%s"}`, sc.largerLabel()), nil, "exact"
	}
}

// GenerateInstructionCliff writes the paired 40-scene × 5-depth dataset and
// its rendered images under datasetDir. It refuses to overwrite an existing
// dataset unless force is set.
func GenerateInstructionCliff(datasetDir string, seed int64, sceneCount int, force bool) (int, error) {
	if sceneCount <= 0 {
		sceneCount = 40
	}
	datasetFile := filepath.Join(datasetDir, "instruction-cliff.jsonl")
	if _, err := os.Stat(datasetFile); err == nil && !force {
		return 0, fmt.Errorf("%s exists; pass --force to regenerate", datasetFile)
	}
	imageDir := filepath.Join(datasetDir, "instruction-cliff", "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return 0, err
	}

	var builder bytes.Buffer
	encoder := json.NewEncoder(&builder)
	written := 0

	for index := 0; index < sceneCount; index++ {
		source := rand.New(rand.NewSource(seed + int64(index)))
		sc := newScene(index, source)
		baseID := fmt.Sprintf("cliff-%03d", index)

		imageRel := filepath.ToSlash(filepath.Join("instruction-cliff", "images", baseID+".png"))
		imageBytes, err := renderScene(sc)
		if err != nil {
			return written, err
		}
		if err := os.WriteFile(filepath.Join(datasetDir, filepath.FromSlash(imageRel)), imageBytes, 0o644); err != nil {
			return written, err
		}

		order := latinSquare[sc.family]
		sentinel := index < 10 || index%9 == 0 // >=2 per family across the 5-family cycle

		for depth := 1; depth <= 5; depth++ {
			last := order[depth-1]
			value, choices, family := sc.answerFor(last)

			priorContract := false
			for step := 0; step < depth-1; step++ {
				if order[step] == primFormat && last != primFormat {
					priorContract = true
				}
			}

			capabilitySet := map[string]bool{}
			for step := 0; step < depth; step++ {
				if capability := order[step].capability(); capability != "" {
					capabilitySet[capability] = true
				}
			}
			capabilities := make([]string, 0, len(capabilitySet))
			for capability := range capabilitySet {
				capabilities = append(capabilities, capability)
			}
			sort.Strings(capabilities)

			item := Case{
				CaseID:         fmt.Sprintf("%s-op%d", baseID, depth),
				Stage:          StageInstructionCliff,
				Capabilities:   capabilities,
				Operations:     depth,
				BaseID:         baseID,
				Sentinel:       sentinel,
				TaskFamily:     family,
				Choices:        choices,
				AddedPrimitive: last.name(),
				PriorContract:  priorContract,
				Instruction:    buildInstruction(order, depth),
				ImagePath:      imageRel,
				Expected:       Expected{Value: value},
			}
			if err := encoder.Encode(item); err != nil {
				return written, err
			}
			written++
		}
	}

	if err := os.WriteFile(datasetFile, builder.Bytes(), 0o644); err != nil {
		return written, err
	}
	return written, nil
}

func buildInstruction(order [5]primitive, depth int) string {
	if depth == 1 {
		return "Using only the image, " + order[0].clause() + "."
	}
	var text strings.Builder
	text.WriteString("Using only the image, perform these steps in order:\n")
	for step := 0; step < depth; step++ {
		fmt.Fprintf(&text, "%d. %s\n", step+1, order[step].clause())
	}
	text.WriteString("Report only the result of the final step, and nothing else.")
	return text.String()
}

// --- rendering (dependency-free) ---

func renderScene(sc scene) ([]byte, error) {
	const width, height = 384, 256
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(canvas, color.RGBA{255, 255, 255, 255})

	leftCenter := image.Point{X: 104, Y: 116}
	rightCenter := image.Point{X: 280, Y: 116}
	circleCenter, squareCenter := leftCenter, rightCenter
	if !sc.circleOnLeft {
		circleCenter, squareCenter = rightCenter, leftCenter
	}

	drawDisc(canvas, circleCenter, sc.circleDiameter/2, scenePalette[sc.circleColor].rgb)
	drawSquare(canvas, squareCenter, sc.squareSide, scenePalette[sc.squareColor].rgb)

	black := color.RGBA{20, 20, 20, 255}
	drawGlyph(canvas, sc.circleLabel, image.Point{X: circleCenter.X, Y: 206}, black)
	drawGlyph(canvas, sc.squareLabel, image.Point{X: squareCenter.X, Y: 206}, black)

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func fill(canvas *image.RGBA, shade color.RGBA) {
	bounds := canvas.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			canvas.SetRGBA(x, y, shade)
		}
	}
}

func drawDisc(canvas *image.RGBA, center image.Point, radius int, shade color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				canvas.SetRGBA(center.X+dx, center.Y+dy, shade)
			}
		}
	}
}

func drawSquare(canvas *image.RGBA, center image.Point, side int, shade color.RGBA) {
	half := side / 2
	for dy := -half; dy <= half; dy++ {
		for dx := -half; dx <= half; dx++ {
			canvas.SetRGBA(center.X+dx, center.Y+dy, shade)
		}
	}
}

// 5x7 bitmap glyphs for the only two labels the dataset uses.
var glyphs = map[string][7]string{
	"A": {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	"B": {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
}

func drawGlyph(canvas *image.RGBA, label string, center image.Point, shade color.RGBA) {
	rows, ok := glyphs[label]
	if !ok {
		return
	}
	const pixel = 4
	originX := center.X - (5*pixel)/2
	originY := center.Y - (7*pixel)/2
	for rowIndex, row := range rows {
		for columnIndex, cell := range row {
			if cell != '1' {
				continue
			}
			for py := 0; py < pixel; py++ {
				for px := 0; px < pixel; px++ {
					canvas.SetRGBA(originX+columnIndex*pixel+px, originY+rowIndex*pixel+py, shade)
				}
			}
		}
	}
}
