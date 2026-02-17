package db

import (
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestConfigGetSetDefault(t *testing.T) {
	d := openTestDB(t)

	val, err := d.GetConfig("missing_key", "default123")
	if err != nil {
		t.Fatal(err)
	}
	if val != "default123" {
		t.Fatalf("expected default123, got %q", val)
	}
}

func TestConfigUpsert(t *testing.T) {
	d := openTestDB(t)

	if err := d.SetConfig("camera_driver", "mock"); err != nil {
		t.Fatal(err)
	}
	val, err := d.GetConfig("camera_driver", "")
	if err != nil {
		t.Fatal(err)
	}
	if val != "mock" {
		t.Fatalf("expected mock, got %q", val)
	}

	// Upsert
	if err := d.SetConfig("camera_driver", "libcamera"); err != nil {
		t.Fatal(err)
	}
	val, err = d.GetConfig("camera_driver", "")
	if err != nil {
		t.Fatal(err)
	}
	if val != "libcamera" {
		t.Fatalf("expected libcamera, got %q", val)
	}
}

func TestAllConfig(t *testing.T) {
	d := openTestDB(t)

	d.SetConfig("a", "1")
	d.SetConfig("b", "2")

	m, err := d.AllConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 || m["a"] != "1" || m["b"] != "2" {
		t.Fatalf("unexpected config map: %v", m)
	}
}

func TestScheduleCRUD(t *testing.T) {
	d := openTestDB(t)

	// Create
	id, err := d.CreateSchedule(Schedule{
		CronExpr: "*/30 * * * *",
		CameraID: "mock",
		HookPath: "/hooks/analyze.sh",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	// Read single
	s, err := d.GetSchedule(id)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected schedule, got nil")
	}
	if s.CronExpr != "*/30 * * * *" || s.CameraID != "mock" || !s.Enabled {
		t.Fatalf("unexpected schedule: %+v", s)
	}

	// List
	all, err := d.ListSchedules()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(all))
	}

	// Update
	s.CronExpr = "0 * * * *"
	s.Enabled = false
	if err := d.UpdateSchedule(*s); err != nil {
		t.Fatal(err)
	}
	s2, _ := d.GetSchedule(id)
	if s2.CronExpr != "0 * * * *" || s2.Enabled {
		t.Fatalf("update failed: %+v", s2)
	}

	// Delete
	if err := d.DeleteSchedule(id); err != nil {
		t.Fatal(err)
	}
	s3, _ := d.GetSchedule(id)
	if s3 != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestGetScheduleNotFound(t *testing.T) {
	d := openTestDB(t)
	s, err := d.GetSchedule(999)
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatal("expected nil for missing schedule")
	}
}

func TestLogCaptureAndList(t *testing.T) {
	d := openTestDB(t)

	// Need a schedule first
	sid, _ := d.CreateSchedule(Schedule{CronExpr: "* * * * *", CameraID: "mock", Enabled: true})

	now := time.Date(2024, 10, 27, 9, 0, 0, 0, time.UTC)
	id, err := d.LogCapture(CaptureLog{
		ScheduleID: sid,
		Timestamp:  now,
		Filepath:   "/data/captures/2024-10-27/09-00-00.jpg",
		HookStatus: "success",
		HookOutput: `{"rise":45}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero capture log id")
	}

	// List all
	logs, err := d.ListCaptures("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].HookStatus != "success" {
		t.Fatalf("unexpected status: %s", logs[0].HookStatus)
	}

	// List by date
	logs2, err := d.ListCaptures("2024-10-27", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs2) != 1 {
		t.Fatalf("expected 1 log for date filter, got %d", len(logs2))
	}

	// Different date
	logs3, err := d.ListCaptures("2024-10-28", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs3) != 0 {
		t.Fatalf("expected 0 logs for wrong date, got %d", len(logs3))
	}
}
