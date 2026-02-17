package scheduler

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/leshicodes/vigil/internal/db"
)

func TestSchedulerLoadAndCount(t *testing.T) {
	var called atomic.Int32
	s := New(func(sched db.Schedule) {
		called.Add(1)
	})

	schedules := []db.Schedule{
		{ID: 1, CronExpr: "* * * * *", CameraID: "mock", Enabled: true},
		{ID: 2, CronExpr: "*/5 * * * *", CameraID: "mock", Enabled: true},
		{ID: 3, CronExpr: "0 * * * *", CameraID: "mock", Enabled: false}, // disabled
	}

	if err := s.Load(schedules); err != nil {
		t.Fatal(err)
	}

	// Only 2 of 3 are enabled.
	if got := s.EntryCount(); got != 2 {
		t.Fatalf("expected 2 entries, got %d", got)
	}
}

func TestSchedulerReload(t *testing.T) {
	s := New(func(sched db.Schedule) {})

	first := []db.Schedule{
		{ID: 1, CronExpr: "* * * * *", CameraID: "mock", Enabled: true},
	}
	s.Load(first)
	if got := s.EntryCount(); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}

	// Reload with different schedules.
	second := []db.Schedule{
		{ID: 2, CronExpr: "* * * * *", CameraID: "mock", Enabled: true},
		{ID: 3, CronExpr: "*/10 * * * *", CameraID: "mock", Enabled: true},
	}
	s.Load(second)
	if got := s.EntryCount(); got != 2 {
		t.Fatalf("expected 2 entries after reload, got %d", got)
	}
}

func TestSchedulerStartStop(t *testing.T) {
	s := New(func(sched db.Schedule) {})
	s.Load([]db.Schedule{
		{ID: 1, CronExpr: "* * * * *", CameraID: "mock", Enabled: true},
	})

	s.Start()
	time.Sleep(100 * time.Millisecond)
	s.Stop()
	// Should not panic or hang.
}

func TestSchedulerInvalidCron(t *testing.T) {
	s := New(func(sched db.Schedule) {})
	err := s.Load([]db.Schedule{
		{ID: 1, CronExpr: "not-a-cron", CameraID: "mock", Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Invalid cron should be skipped, not error.
	if got := s.EntryCount(); got != 0 {
		t.Fatalf("expected 0 entries for invalid cron, got %d", got)
	}
}
