package camera

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LibCamera captures images using the `libcamera-still` CLI tool,
// typically available on Raspberry Pi OS.
type LibCamera struct {
	// ExtraArgs allows passing additional flags to libcamera-still.
	ExtraArgs []string
}

func (l *LibCamera) Capture(outputPath string) error {
	args := []string{
		"-o", outputPath,
		"--nopreview",
		"--immediate",
		"--width", "1920",
		"--height", "1080",
	}
	if extra := os.Getenv("VIGIL_LIBCAMERA_ARGS"); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	args = append(args, l.ExtraArgs...)

	// Raspberry Pi OS Debian 12 (Bookworm) and newer use rpicam-still
	exe := "libcamera-still"
	if _, err := exec.LookPath("rpicam-still"); err == nil {
		exe = "rpicam-still"
	}

	cmd := exec.Command(exe, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w\noutput: %s", exe, err, string(output))
	}
	return nil
}
