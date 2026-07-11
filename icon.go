package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

const (
	iconSize   = 32
	iconCenter = float64(iconSize) / 2
	iconRadius = 12.0
)

func generateCircleIcon(col color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))

	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			dx := float64(x) - iconCenter + 0.5
			dy := float64(y) - iconCenter + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist <= iconRadius {
				alpha := 1.0
				if dist > iconRadius-1.0 {
					alpha = iconRadius - dist
				}
				a := uint8(float64(col.A) * alpha)
				img.SetRGBA(x, y, color.RGBA{col.R, col.G, col.B, a})
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

var (
	iconGreen  = generateCircleIcon(color.RGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF})
	iconYellow = generateCircleIcon(color.RGBA{R: 0xFF, G: 0xCC, B: 0x00, A: 0xFF})
	iconRed    = generateCircleIcon(color.RGBA{R: 0xFF, G: 0x3B, B: 0x30, A: 0xFF})
	iconGray   = generateCircleIcon(color.RGBA{R: 0x8E, G: 0x8E, B: 0x93, A: 0xFF})
)

func iconForStatus(s ProviderStatus) []byte {
	switch s {
	case StatusOK:
		return iconGreen
	case StatusWarning:
		return iconYellow
	case StatusCritical:
		return iconRed
	case StatusError:
		return iconGray
	default:
		return iconGray
	}
}
