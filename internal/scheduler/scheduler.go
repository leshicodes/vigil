package scheduler

import (
	"log"
	"sync"

	"github.com/leshicodes/vigil/internal/db"
	"github.com/robfig/cron/v3"
)

// CaptureFunc is the function called when a scheduled capture triggers.
type CaptureFunc func(sched db.Schedule)

// Scheduler manages cron-based capture schedules.
type Scheduler struct {
	cron      *cron.Cron
	captureFn CaptureFunc
	mu        sync.Mutex
	entryMap  map[int64]cron.EntryID // schedule ID -> cron entry ID
}

// New creates a Scheduler with the given capture function.
func New(fn CaptureFunc) *Scheduler {
	return &Scheduler{
		cron:      cron.New(),
		captureFn: fn,
		entryMap:  make(map[int64]cron.EntryID),
	}
}

// Load registers cron jobs for all enabled schedules.
// It replaces any previously loaded schedules.
func (s *Scheduler) Load(schedules []db.Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entries.
	for _, eid := range s.entryMap {
		s.cron.Remove(eid)
	}
	s.entryMap = make(map[int64]cron.EntryID)

	for _, sched := range schedules {
		if !sched.Enabled {
			continue
		}

		// Capture the schedule value for the closure.
		localSched := sched

		eid, err := s.cron.AddFunc(localSched.CronExpr, func() {
			log.Printf("[scheduler] firing schedule %d: %s", localSched.ID, localSched.CronExpr)
			s.captureFn(localSched)
		})
		if err != nil {
			log.Printf("[scheduler] failed to add schedule %d (%s): %v", localSched.ID, localSched.CronExpr, err)
			continue
		}

		s.entryMap[localSched.ID] = eid
		log.Printf("[scheduler] registered schedule %d: %s (camera=%s)", localSched.ID, localSched.CronExpr, localSched.CameraID)
	}

	return nil
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("[scheduler] started")
}

// Stop gracefully stops the scheduler and waits for running jobs.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("[scheduler] stopped")
}

// EntryCount returns the number of active cron entries.
func (s *Scheduler) EntryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entryMap)
}
