package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leshicodes/vigil/internal/db"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	return &Server{
		DB:        d,
		DataDir:   t.TempDir(),
		StartTime: time.Now(),
	}
}

func TestStatusEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	r := srv.NewRouter()

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}

func TestSchedulesCRUD(t *testing.T) {
	srv := setupTestServer(t)
	r := srv.NewRouter()

	// Create
	body := `{"cron_expr":"*/30 * * * *","camera_id":"mock","hook_path":"","enabled":true}`
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created db.Schedule
	json.NewDecoder(w.Body).Decode(&created)
	if created.ID == 0 {
		t.Fatal("expected non-zero id")
	}

	// List
	req = httptest.NewRequest("GET", "/api/schedules", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var list []db.Schedule
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(list))
	}

	// Update
	updateBody := `{"cron_expr":"0 * * * *","camera_id":"mock","hook_path":"","enabled":false}`
	req = httptest.NewRequest("PUT", "/api/schedules/1", bytes.NewBufferString(updateBody))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", w.Code)
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/schedules/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	// Verify empty
	req = httptest.NewRequest("GET", "/api/schedules", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 0 {
		t.Fatalf("expected 0 schedules after delete, got %d", len(list))
	}
}

func TestConfigEndpoints(t *testing.T) {
	srv := setupTestServer(t)
	r := srv.NewRouter()

	// Set config
	body := `{"camera_driver":"mock","capture_interval":"30m"}`
	req := httptest.NewRequest("PUT", "/api/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("set config: expected 200, got %d", w.Code)
	}

	// Get config
	req = httptest.NewRequest("GET", "/api/config", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var cfg map[string]string
	json.NewDecoder(w.Body).Decode(&cfg)
	if cfg["camera_driver"] != "mock" {
		t.Fatalf("expected camera_driver=mock, got %v", cfg["camera_driver"])
	}
}

func TestListCapturesEmpty(t *testing.T) {
	srv := setupTestServer(t)
	r := srv.NewRouter()

	req := httptest.NewRequest("GET", "/api/captures", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var captures []db.CaptureLog
	json.NewDecoder(w.Body).Decode(&captures)
	if len(captures) != 0 {
		t.Fatalf("expected 0 captures, got %d", len(captures))
	}
}
