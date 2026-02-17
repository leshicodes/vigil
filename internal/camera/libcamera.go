package camera

import (
	"fmt"
	"os/exec"
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
	args = append(args, l.ExtraArgs...)

	cmd := exec.Command("libcamera-still", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libcamera-still failed: %w\noutput: %s", err, string(output))
	}
	return nil
}
