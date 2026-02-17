package camera

import (
	"fmt"
	"os/exec"
)

// FSWebcam captures images using the `fswebcam` CLI tool,
// commonly used for USB webcams on Linux.
type FSWebcam struct {
	// Device is the video device path, e.g. "/dev/video0".
	// If empty, fswebcam uses its default.
	Device string
	// ExtraArgs allows passing additional flags.
	ExtraArgs []string
}

func (f *FSWebcam) Capture(outputPath string) error {
	args := []string{
		"-r", "1920x1080",
		"--no-banner",
		"--jpeg", "85",
	}
	if f.Device != "" {
		args = append(args, "-d", f.Device)
	}
	args = append(args, f.ExtraArgs...)
	args = append(args, outputPath)

	cmd := exec.Command("fswebcam", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fswebcam failed: %w\noutput: %s", err, string(output))
	}
	return nil
}
