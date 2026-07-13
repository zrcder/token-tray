package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

type DotColor struct {
	R, G, B, A uint8
}

var (
	colGreen  = DotColor{0x34, 0xC7, 0x59, 0xFF}
	colYellow = DotColor{0xFF, 0xCC, 0x00, 0xFF}
	colRed    = DotColor{0xFF, 0x3B, 0x30, 0xFF}
	colGray   = DotColor{0x8E, 0x8E, 0x93, 0xFF}
)

func colorForFraction(f *float64) DotColor {
	if f == nil {
		return colGray
	}
	v := *f
	switch {
	case v >= 0.9:
		return colRed
	case v >= 0.7:
		return colYellow
	default:
		return colGreen
	}
}

// generateSegmentedIcon draws a horizontal bar split into N independently-colored segments. Each segment is a solid rectangle.
func generateSegmentedIcon(segments []DotColor) []byte {
	const (
		canvasW = 48
		canvasH = 14
		gap     = 2
		padY    = 1
	)

	n := len(segments)
	if n == 0 {
		n = 1
		segments = []DotColor{colGray}
	}

	segW := (canvasW - gap*(n-1)) / n
	img := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))

	for i, col := range segments {
		x0 := i * (segW + gap)
		x1 := x0 + segW
		for y := padY; y < canvasH-padY; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, color.RGBA{col.R, col.G, col.B, col.A})
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

var iconLoading = generateSegmentedIcon([]DotColor{colGray})
