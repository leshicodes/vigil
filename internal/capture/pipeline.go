package capture

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/leshicodes/vigil/internal/camera"
	"github.com/leshicodes/vigil/internal/db"
	"github.com/leshicodes/vigil/internal/hook"
)

// Pipeline orchestrates a single capture event: take a photo, run hooks,
// and log the result.
type Pipeline struct {
	DataDir     string
	DB          *db.DB
	HookTimeout time.Duration
}

// Execute runs one capture cycle for the given schedule.
func (p *Pipeline) Execute(sched db.Schedule) {
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
		p.logCapture(sched.ID, now, imgPath, "error", fmt.Sprintf("capture: %v", err))
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
