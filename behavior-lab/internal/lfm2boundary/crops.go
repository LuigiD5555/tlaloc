package lfm2boundary

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

const canonicalCarrierSize = 640

type CropSpec struct {
	ID string `json:"id"`
	X0 int    `json:"x0"`
	Y0 int    `json:"y0"`
	X1 int    `json:"x1"`
	Y1 int    `json:"y1"`
	Scale int  `json:"scale"`
}

// DeclaredCrops mirror the current Origami temporal carrier renderer: BOOT/T1
// lives above y=105, T2 semantic cells occupy y=112..318, and the timeline is
// below that. SEMANTIC_FULL stops before the exact payload beginning near y=398.
func DeclaredCrops() []CropSpec {
	return []CropSpec{
		{ID:"BOOT_T1", X0:8, Y0:8, X1:632, Y1:105, Scale:3},
		{ID:"T2", X0:18, Y0:112, X1:622, Y1:318, Scale:2},
		{ID:"TIMELINE", X0:18, Y0:318, X1:622, Y1:390, Scale:3},
		{ID:"SEMANTIC_FULL", X0:8, Y0:8, X1:632, Y1:390, Scale:2},
	}
}

func CropNearestPNG(data []byte, spec CropSpec) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil { return nil, fmt.Errorf("decode carrier PNG: %w", err) }
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 { return nil, fmt.Errorf("empty carrier image") }
	sx := func(v int) int { return b.Min.X + v*b.Dx()/canonicalCarrierSize }
	sy := func(v int) int { return b.Min.Y + v*b.Dy()/canonicalCarrierSize }
	r := image.Rect(sx(spec.X0), sy(spec.Y0), sx(spec.X1), sy(spec.Y1)).Intersect(b)
	if r.Empty() { return nil, fmt.Errorf("crop %s is empty", spec.ID) }
	scale := spec.Scale; if scale <= 0 { scale = 1 }
	out := image.NewRGBA(image.Rect(0, 0, r.Dx()*scale, r.Dy()*scale))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, img.At(r.Min.X+x/scale, r.Min.Y+y/scale))
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel:png.BestSpeed}
	if err := enc.Encode(&buf, out); err != nil { return nil, err }
	return buf.Bytes(), nil
}

func WriteDeclaredCrops(carrierPath, outDir string) (map[string]string, error) {
	data, err := os.ReadFile(carrierPath); if err != nil { return nil, err }
	if err := os.MkdirAll(outDir, 0o700); err != nil { return nil, err }
	paths := map[string]string{}
	for _, spec := range DeclaredCrops() {
		crop, err := CropNearestPNG(data, spec); if err != nil { return nil, err }
		path := filepath.Join(outDir, spec.ID+".png")
		if err := os.WriteFile(path, crop, 0o600); err != nil { return nil, err }
		paths[spec.ID] = path
	}
	return paths, nil
}

// Keep image/draw linked in the standard-library-only package so future crop
// overlays can remain deterministic without adding an imaging dependency.
var _ = draw.Src
