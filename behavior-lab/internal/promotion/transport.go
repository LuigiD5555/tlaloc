package promotion

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

type TransportVariant struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bytes     []byte `json:"-"`
}

func BuildTransportVariants(canonicalPNG []byte) ([]TransportVariant, error) {
	img, err := png.Decode(bytes.NewReader(canonicalPNG))
	if err != nil {
		return nil, fmt.Errorf("decode canonical PNG: %w", err)
	}
	b := img.Bounds()
	variants := []TransportVariant{{Name: "original", MediaType: "image/png", Width: b.Dx(), Height: b.Dy(), Bytes: append([]byte(nil), canonicalPNG...)}}
	for _, item := range []struct {
		name  string
		scale float64
	}{{"resize-75", .75}, {"resize-50", .50}} {
		w := int(float64(b.Dx()) * item.scale)
		h := int(float64(b.Dy()) * item.scale)
		resized := resizeNearest(img, w, h)
		var out bytes.Buffer
		if err := png.Encode(&out, resized); err != nil {
			return nil, err
		}
		variants = append(variants, TransportVariant{Name: item.name, MediaType: "image/png", Width: w, Height: h, Bytes: out.Bytes()})
	}
	preview := resizeNearest(img, int(float64(b.Dx())*.75), int(float64(b.Dy())*.75))
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, preview, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	variants = append(variants, TransportVariant{Name: "jpeg-preview", MediaType: "image/jpeg", Width: preview.Bounds().Dx(), Height: preview.Bounds().Dy(), Bytes: jpg.Bytes()})
	return variants, nil
}

func resizeNearest(src image.Image, width, height int) *image.NRGBA {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	sb := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sy := sb.Min.Y + (y*sb.Dy())/height
		for x := 0; x < width; x++ {
			sx := sb.Min.X + (x*sb.Dx())/width
			dst.Set(x, y, color.NRGBAModel.Convert(src.At(sx, sy)))
		}
	}
	return dst
}
