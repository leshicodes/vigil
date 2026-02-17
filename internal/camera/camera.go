package camera

import "fmt"

// Camera is the interface all camera drivers must implement.
type Camera interface {
	// Capture takes a photo and writes it to outputPath.
	// The outputPath will always end in .jpg.
	Capture(outputPath string) error
}

// New creates a Camera for the given driver name.
// Supported drivers: "mock", "libcamera", "fswebcam".
func New(driver string) (Camera, error) {
	switch driver {
	case "mock":
		return &MockCamera{}, nil
	case "libcamera":
		return &LibCamera{}, nil
	case "fswebcam":
		return &FSWebcam{}, nil
	default:
		return nil, fmt.Errorf("unknown camera driver: %q", driver)
	}
}
