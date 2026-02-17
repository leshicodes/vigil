package camera

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	device := f.Device
	if device == "" {
		device = os.Getenv("VIGIL_FSWEBCAM_DEVICE")
	}

	args := []string{
		"--no-banner",
		"--jpeg", "85",
		"--skip", "20",
	}
	if device != "" {
		args = append(args, "-d", device)
	}

	// Resolution: allow override via env.
	if res := os.Getenv("VIGIL_FSWEBCAM_RES"); res != "" {
		args = append(args, "-r", res)
	}

	if extra := os.Getenv("VIGIL_FSWEBCAM_ARGS"); extra != "" {
		args = append(args, strings.Fields(extra)...)
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
