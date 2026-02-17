package capture

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/leshicodes/vigil/internal/camera"
	"github.com/leshicodes/vigil/internal/db"
	"github.com/leshicodes/vigil/internal/hook"
	"github.com/leshicodes/vigil/internal/logger"
)

// Pipeline orchestrates a single capture event: take a photo, run hooks,
// and log the result. It serializes access to the camera to prevent
// concurrent DirectShow/V4L2 conflicts.
type Pipeline struct {
	DataDir     string
	DB          *db.DB
	HookTimeout time.Duration

	mu sync.Mutex // protects camera access
}

// Busy returns true if a capture is currently in progress.
func (p *Pipeline) Busy() bool {
	if p.mu.TryLock() {
		p.mu.Unlock()
		return false
	}
	return true
}

// Execute runs one capture cycle for the given schedule.
// It acquires a mutex so only one capture runs at a time.
func (p *Pipeline) Execute(sched db.Schedule) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Build output path: /data/captures/YYYY-MM-DD/HH-MM-SS.jpg
	dateDir := now.Format("2006-01-02")
	timeFile := now.Format("15-04-05")
	dir := filepath.Join(p.DataDir, "captures", dateDir)

	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("capture", "failed to create dir %s: %v", dir, err)
		return
	}

	imgPath := filepath.Join(dir, timeFile+".jpg")

	// Get camera driver.
	logger.Info("capture", "starting capture - driver=%s output=%s", sched.CameraID, imgPath)
	logger.Debug("capture", "schedule details - id=%d cron=%q hook=%q enabled=%v",
		sched.ID, sched.CronExpr, sched.HookPath, sched.Enabled)

	cam, err := camera.New(sched.CameraID)
	if err != nil {
		logger.Error("capture", "camera init failed for driver %q: %v", sched.CameraID, err)
		return
	}

	// Retry logic: the Pi 3B+ USB subsystem can hold /dev/video0 as "busy"
	// for several seconds after a previous capture. Retry with a delay.
	maxRetries := 3
	retryDelay := 5 * time.Second
	if v := os.Getenv("VIGIL_CAPTURE_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxRetries = n
		}
	}

	var captureErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		captureErr = cam.Capture(imgPath)
		if captureErr == nil {
			break
		}
		logger.Warn("capture", "attempt %d/%d failed: %v", attempt, maxRetries, captureErr)
		os.Remove(imgPath) // clean up any partial file
		if attempt < maxRetries {
			logger.Info("capture", "retrying in %s...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	if captureErr != nil {
		logger.Error("capture", "capture failed after %d attempts - skipping DB entry", maxRetries)
		os.Remove(imgPath)
		return
	}

	// Verify the file actually has content.
	// A valid JPEG from a webcam is always > 1KB. Anything smaller is garbage.
	if info, err := os.Stat(imgPath); err != nil || info.Size() < 1000 {
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		logger.Warn("capture", "image file missing or too small (%d bytes): %s - skipping DB entry", size, imgPath)
		os.Remove(imgPath)
		return
	}

	info, _ := os.Stat(imgPath)
	logger.Info("capture", "saved %s (%d bytes)", imgPath, info.Size())

	// Run hook if configured.
	timeout := p.HookTimeout
	if timeout == 0 {
		timeout = hook.DefaultTimeout
	}
	result := hook.Run(sched.HookPath, imgPath, timeout)

	if result.Status == "success" {
		logger.Info("capture", "hook succeeded for %s", imgPath)
	} else if result.Status == "error" {
		logger.Error("capture", "hook error for %s: %s", imgPath, result.Output)
	}

	p.logCapture(sched.ID, now, imgPath, result.Status, result.Output)
}

func (p *Pipeline) logCapture(schedID int64, ts time.Time, path, status, output string) {
	if p.DB == nil {
		return
	}
	_, err := p.DB.LogCapture(db.CaptureLog{
		ScheduleID: schedID,
		Timestamp:  ts,
		Filepath:   path,
		HookStatus: status,
		HookOutput: output,
	})
	if err != nil {
		logger.Error("capture", "failed to log capture to DB: %v", err)
	}
}
