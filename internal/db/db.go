package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection and provides CRUD operations for Vigil.
type DB struct {
	conn *sql.DB
}

// Schedule represents a capture schedule entry.
type Schedule struct {
	ID       int64  `json:"id"`
	CronExpr string `json:"cron_expr"`
	CameraID string `json:"camera_id"`
	HookPath string `json:"hook_path"`
	Enabled  bool   `json:"enabled"`
}

// CaptureLog represents a single capture event record.
type CaptureLog struct {
	ID         int64     `json:"id"`
	ScheduleID int64     `json:"schedule_id"`
	Timestamp  time.Time `json:"timestamp"`
	Filepath   string    `json:"filepath"`
	HookStatus string    `json:"hook_status"` // "success", "error", "skipped"
	HookOutput string    `json:"hook_output"`
}

// Open creates or opens the SQLite database at the given path and runs
// auto-migrations to ensure all tables exist.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS schedules (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			cron_expr TEXT NOT NULL,
			camera_id TEXT NOT NULL DEFAULT 'mock',
			hook_path TEXT NOT NULL DEFAULT '',
			enabled   INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS capture_log (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			schedule_id INTEGER NOT NULL,
			timestamp   TEXT NOT NULL,
			filepath    TEXT NOT NULL,
			hook_status TEXT NOT NULL DEFAULT 'skipped',
			hook_output TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (schedule_id) REFERENCES schedules(id)
		)`,
	}
	for _, s := range stmts {
		if _, err := d.conn.Exec(s); err != nil {
			return fmt.Errorf("exec %q: %w", s[:40], err)
		}
	}
	return nil
}

// --- Config ----------------------------------------------------------------

// GetConfig returns the value for a config key, or defaultVal if not set.
func (d *DB) GetConfig(key, defaultVal string) (string, error) {
	var val string
	err := d.conn.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return defaultVal, nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetConfig upserts a config key/value pair.
func (d *DB) SetConfig(key, value string) error {
	_, err := d.conn.Exec(
		"INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

// AllConfig returns all config entries as a map.
func (d *DB) AllConfig() (map[string]string, error) {
	rows, err := d.conn.Query("SELECT key, value FROM config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// --- Schedules -------------------------------------------------------------

// ListSchedules returns all schedules.
func (d *DB) ListSchedules() ([]Schedule, error) {
	rows, err := d.conn.Query("SELECT id, cron_expr, camera_id, hook_path, enabled FROM schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Schedule
	for rows.Next() {
		var s Schedule
		var enabled int
		if err := rows.Scan(&s.ID, &s.CronExpr, &s.CameraID, &s.HookPath, &enabled); err != nil {
			return nil, err
		}
		s.Enabled = enabled == 1
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSchedule returns a single schedule by ID.
func (d *DB) GetSchedule(id int64) (*Schedule, error) {
	var s Schedule
	var enabled int
	err := d.conn.QueryRow(
		"SELECT id, cron_expr, camera_id, hook_path, enabled FROM schedules WHERE id = ?", id,
	).Scan(&s.ID, &s.CronExpr, &s.CameraID, &s.HookPath, &enabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Enabled = enabled == 1
	return &s, nil
}

// CreateSchedule inserts a new schedule and returns its ID.
func (d *DB) CreateSchedule(s Schedule) (int64, error) {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	res, err := d.conn.Exec(
		"INSERT INTO schedules (cron_expr, camera_id, hook_path, enabled) VALUES (?, ?, ?, ?)",
		s.CronExpr, s.CameraID, s.HookPath, enabled,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateSchedule updates an existing schedule.
func (d *DB) UpdateSchedule(s Schedule) error {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := d.conn.Exec(
		"UPDATE schedules SET cron_expr = ?, camera_id = ?, hook_path = ?, enabled = ? WHERE id = ?",
		s.CronExpr, s.CameraID, s.HookPath, enabled, s.ID,
	)
	return err
}

// DeleteSchedule removes a schedule by ID.
func (d *DB) DeleteSchedule(id int64) error {
	_, err := d.conn.Exec("DELETE FROM schedules WHERE id = ?", id)
	return err
}

// --- Capture Log -----------------------------------------------------------

// LogCapture records a capture event.
func (d *DB) LogCapture(log CaptureLog) (int64, error) {
	res, err := d.conn.Exec(
		"INSERT INTO capture_log (schedule_id, timestamp, filepath, hook_status, hook_output) VALUES (?, ?, ?, ?, ?)",
		log.ScheduleID, log.Timestamp.UTC().Format(time.RFC3339), log.Filepath, log.HookStatus, log.HookOutput,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListCaptures returns capture logs, optionally filtered by date (YYYY-MM-DD).
// If date is empty, returns the most recent limit entries.
func (d *DB) ListCaptures(date string, limit int) ([]CaptureLog, error) {
	var rows *sql.Rows
	var err error

	if date != "" {
		rows, err = d.conn.Query(
			"SELECT id, schedule_id, timestamp, filepath, hook_status, hook_output FROM capture_log WHERE timestamp LIKE ? ORDER BY timestamp DESC LIMIT ?",
			date+"%", limit,
		)
	} else {
		rows, err = d.conn.Query(
			"SELECT id, schedule_id, timestamp, filepath, hook_status, hook_output FROM capture_log ORDER BY timestamp DESC LIMIT ?",
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CaptureLog
	for rows.Next() {
		var cl CaptureLog
		var ts string
		if err := rows.Scan(&cl.ID, &cl.ScheduleID, &ts, &cl.Filepath, &cl.HookStatus, &cl.HookOutput); err != nil {
			return nil, err
		}
		cl.Timestamp, _ = time.Parse(time.RFC3339, ts)
		out = append(out, cl)
	}
	return out, rows.Err()
}

// GetCapture returns a single capture log entry by ID.
func (d *DB) GetCapture(id int64) (*CaptureLog, error) {
	var cl CaptureLog
	var ts string
	err := d.conn.QueryRow(
		"SELECT id, schedule_id, timestamp, filepath, hook_status, hook_output FROM capture_log WHERE id = ?", id,
	).Scan(&cl.ID, &cl.ScheduleID, &ts, &cl.Filepath, &cl.HookStatus, &cl.HookOutput)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cl.Timestamp, _ = time.Parse(time.RFC3339, ts)
	return &cl, nil
}

// DeleteCapture removes a capture log entry by ID.
func (d *DB) DeleteCapture(id int64) error {
	_, err := d.conn.Exec("DELETE FROM capture_log WHERE id = ?", id)
	return err
}

// DeleteCapturesByIDs removes multiple capture log entries by their IDs.
func (d *DB) DeleteCapturesByIDs(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	// Build placeholder list: (?, ?, ?)
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "DELETE FROM capture_log WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	res, err := d.conn.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
