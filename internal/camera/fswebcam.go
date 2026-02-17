package camera

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/leshicodes/vigil/internal/logger"
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

	// Delay gives the camera real time to auto-expose/white-balance.
	// Critical for Logitech cameras on Pi 3B+ USB 2.0.
	delay := os.Getenv("VIGIL_FSWEBCAM_DELAY")
	if delay == "" {
		delay = "3"
	}

	args := []string{
		"--no-banner",
		"--jpeg", "85",
		"--delay", delay,
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

	logger.Debug("fswebcam", "running: fswebcam %s", strings.Join(args, " "))

	cmd := exec.Command("fswebcam", args...)
	output, err := cmd.CombinedOutput()

	// Always log output — fswebcam prints device negotiation info (resolution,
	// format, palette) that is critical for diagnosing black frame issues.
	if len(output) > 0 {
		logger.Debug("fswebcam", "output:\n%s", string(output))
	}

	if err != nil {
		logger.Error("fswebcam", "command failed: %v", err)
		return fmt.Errorf("fswebcam failed: %w\noutput: %s", err, string(output))
	}

	// Check resulting file size.
	if info, statErr := os.Stat(outputPath); statErr == nil {
		logger.Info("fswebcam", "captured %s (%d bytes)", outputPath, info.Size())
		if info.Size() < 1000 {
			logger.Warn("fswebcam", "file is suspiciously small (%d bytes) — may be a black frame", info.Size())
		}
	}

	return nil
}
