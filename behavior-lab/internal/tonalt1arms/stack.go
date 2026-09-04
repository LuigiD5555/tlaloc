package tonalt1arms

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

const (
	// StackPadPx is the white padding between stacked crop images (frozen).
	StackPadPx = 16
)

// StackVertical stacks PNG images vertically with padding between them.
// The output is a single PNG containing all crops stacked top-to-bottom.
// This is deterministic: identical inputs always produce identical bytes.
func StackVertical(crops [][]byte, padPx int) ([]byte, error) {
	if len(crops) == 0 {
		return nil, fmt.Errorf("cannot stack zero crops")
	}

	// Decode all images and collect dimensions
	decodedImgs := make([]image.Image, len(crops))
	var maxWidth int
	var totalHeight int

	for i, cropBytes := range crops {
		img, err := png.Decode(bytes.NewReader(cropBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to decode crop %d: %w", i, err)
		}
		decodedImgs[i] = img
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		if width > maxWidth {
			maxWidth = width
		}
		totalHeight += height
		if i > 0 {
			totalHeight += padPx // Padding between images
		}
	}

	// Create the output image with white background
	outImg := image.NewRGBA(image.Rect(0, 0, maxWidth, totalHeight))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	draw.Draw(outImg, outImg.Bounds(), image.NewUniform(white), image.Point{}, draw.Src)

	// Blit each image onto the output
	y := 0
	for i, img := range decodedImgs {
		bounds := img.Bounds()
		height := bounds.Dy()
		draw.Draw(outImg, image.Rect(0, y, maxWidth, y+height), img, bounds.Min, draw.Src)
		y += height
		if i < len(decodedImgs)-1 {
			y += padPx // Padding space stays white (already filled above)
		}
	}

	// Encode back to PNG
	var outBuf bytes.Buffer
	if err := png.Encode(&outBuf, outImg); err != nil {
		return nil, fmt.Errorf("failed to encode stacked image: %w", err)
	}

	return outBuf.Bytes(), nil
}
