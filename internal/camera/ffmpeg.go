package camera

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FFmpegCamera captures images using FFmpeg, which can interface with
// DirectShow devices on Windows and V4L2 devices on Linux.
// This is useful for USB webcams on development machines.
type FFmpegCamera struct {
	// Device is the input device identifier.
	// Windows: "Logi C310 HD WebCam" (DirectShow device name)
	// Linux: "/dev/video0" (V4L2 device path)
	// If empty, auto-detects the first available video device.
	Device string
	// ExtraArgs allows passing additional flags.
	ExtraArgs []string
}

func (f *FFmpegCamera) Capture(outputPath string) error {
	device := f.Device
	if device == "" {
		device = os.Getenv("VIGIL_FFMPEG_DEVICE")
	}
	if device == "" {
		// Auto-detect the first video device.
		detected, err := detectFirstVideoDevice()
		if err != nil {
			return fmt.Errorf("auto-detect device: %w", err)
		}
		device = detected
	}

	inputArgs := os.Getenv("VIGIL_FFMPEG_INPUT_ARGS")
	var args []string
	if isWindows() {
		// DirectShow on Windows
		args = []string{"-y"}
		if inputArgs != "" {
			args = append(args, strings.Fields(inputArgs)...)
		} else {
			args = append(args, "-f", "dshow", "-rtbufsize", "100M")
		}
		args = append(args, "-i", "video="+device)
	} else {
		// V4L2 on Linux
		args = []string{"-y"}
		if inputArgs != "" {
			args = append(args, strings.Fields(inputArgs)...)
		} else {
			// For USB cams on Linux, mjpeg often prevents bandwidth/black frame issues.
			// No hardcoded -video_size to let the camera negotiate.
			args = append(args, "-f", "v4l2", "-input_format", "mjpeg")
		}
		args = append(args, "-i", device)
	}

	// Warm up the camera. For live streams, -t 1 tells ffmpeg to consume 1s of stream
	// before seeking with -ss. 2.0s is usually enough for auto-exposure.
	args = append(args, "-t", "3", "-ss", "2.0", "-frames:v", "1", "-q:v", "2")
	args = append(args, f.ExtraArgs...)
	args = append(args, outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\noutput: %s", err, string(output))
	}
	return nil
}

// detectFirstVideoDevice finds the first available video device.
func detectFirstVideoDevice() (string, error) {
	if isWindows() {
		return detectFirstDShowVideoDevice()
	}
	// On Linux, default to /dev/video0
	return "/dev/video0", nil
}

// detectFirstDShowVideoDevice lists DirectShow devices and returns the first video device.
func detectFirstDShowVideoDevice() (string, error) {
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	output, _ := cmd.CombinedOutput() // ffmpeg exits non-zero for -list_devices, that's normal

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Lines look like: [dshow @ ...] "Device Name" (video)
		if strings.Contains(line, "(video)") && !strings.Contains(line, "Alternative name") {
			// Extract the device name between quotes.
			start := strings.Index(line, `"`)
			if start == -1 {
				continue
			}
			end := strings.Index(line[start+1:], `"`)
			if end == -1 {
				continue
			}
			name := line[start+1 : start+1+end]
			return name, nil
		}
	}

	return "", fmt.Errorf("no video devices found; install a webcam or specify device name")
}

func isWindows() bool {
	_, err := exec.LookPath("cmd.exe")
	return err == nil
}
