package camera

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestMockCameraProducesValidJPEG(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test.jpg")

	cam := &MockCamera{}
	if err := cam.Capture(outPath); err != nil {
		t.Fatalf("capture failed: %v", err)
	}

	// Verify the file exists and is a valid JPEG.
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatalf("decode jpeg: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 640 || bounds.Dy() != 480 {
		t.Fatalf("expected 640x480, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestNewCameraFactory(t *testing.T) {
	tests := []struct {
		driver  string
		wantErr bool
	}{
		{"mock", false},
		{"libcamera", false},
		{"fswebcam", false},
		{"unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			cam, err := New(tt.driver)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cam == nil {
				t.Fatal("expected non-nil camera")
			}
		})
	}
}
