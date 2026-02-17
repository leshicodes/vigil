package camera

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"time"
)

// MockCamera generates a synthetic test image with a timestamp.
// Used for development and testing when no physical camera is available.
type MockCamera struct{}

func (m *MockCamera) Capture(outputPath string) error {
	const width, height = 640, 480

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a dark gradient background.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8(30 + (y*20)/height)
			g := uint8(30 + (x*20)/width)
			b := uint8(60 + ((x+y)*30)/(width+height))
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Draw a simple timestamp indicator - a bright rectangle block whose
	// position encodes the current second (0-59) as a visual marker.
	ts := time.Now()
	sec := ts.Second()
	blockX := (sec * (width - 40)) / 59
	for dy := 0; dy < 20; dy++ {
		for dx := 0; dx < 40; dx++ {
			img.Set(blockX+dx, height-30+dy, color.RGBA{R: 0, G: 255, B: 180, A: 255})
		}
	}

	// Draw a border to make the image recognizable.
	borderColor := color.RGBA{R: 100, G: 200, B: 255, A: 255}
	for x := 0; x < width; x++ {
		img.Set(x, 0, borderColor)
		img.Set(x, 1, borderColor)
		img.Set(x, height-1, borderColor)
		img.Set(x, height-2, borderColor)
	}
	for y := 0; y < height; y++ {
		img.Set(0, y, borderColor)
		img.Set(1, y, borderColor)
		img.Set(width-1, y, borderColor)
		img.Set(width-2, y, borderColor)
	}

	// Encode as JPEG.
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("encode jpeg: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
