package capture

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/leshicodes/vigil/internal/camera"
	"github.com/leshicodes/vigil/internal/db"
	"github.com/leshicodes/vigil/internal/hook"
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
		log.Printf("[capture] failed to create dir %s: %v", dir, err)
		return
	}

	imgPath := filepath.Join(dir, timeFile+".jpg")

	// Get camera driver.
	cam, err := camera.New(sched.CameraID)
	if err != nil {
		log.Printf("[capture] camera error: %v", err)
		p.logCapture(sched.ID, now, imgPath, "error", fmt.Sprintf("camera init: %v", err))
		return
	}

	// Capture image.
	if err := cam.Capture(imgPath); err != nil {
		log.Printf("[capture] capture failed: %v", err)
		// Clean up any empty/broken image file.
		os.Remove(imgPath)
		p.logCapture(sched.ID, now, imgPath, "error", fmt.Sprintf("capture: %v", err))
		return
	}

	// Verify the file actually has content.
	if info, err := os.Stat(imgPath); err != nil || info.Size() < 100 {
		log.Printf("[capture] image file missing or too small: %s", imgPath)
		os.Remove(imgPath)
		p.logCapture(sched.ID, now, imgPath, "error", "captured image was empty or corrupt")
		return
	}

	log.Printf("[capture] saved %s", imgPath)

	// Run hook if configured.
	timeout := p.HookTimeout
	if timeout == 0 {
		timeout = hook.DefaultTimeout
	}
	result := hook.Run(sched.HookPath, imgPath, timeout)

	if result.Status == "success" {
		log.Printf("[capture] hook succeeded for %s", imgPath)
	} else if result.Status == "error" {
		log.Printf("[capture] hook error for %s: %s", imgPath, result.Output)
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
		log.Printf("[capture] failed to log capture: %v", err)
	}
}
