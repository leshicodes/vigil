package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/leshicodes/vigil/internal/capture"
	"github.com/leshicodes/vigil/internal/db"
	"github.com/leshicodes/vigil/internal/scheduler"
)

// Server holds dependencies for the API handlers.
type Server struct {
	DB            *db.DB
	Scheduler     *scheduler.Scheduler
	Pipeline      *capture.Pipeline
	DataDir       string
	StaticDir     string
	APIKey        string
	DefaultCamera string
	StartTime     time.Time
}

// NewRouter creates the chi router with all routes mounted.
func (s *Server) NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(corsMiddleware)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Public routes (no auth required).
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/check", s.handleAuthCheck)
		r.Get("/status", s.handleStatus)

		// Protected routes.
		r.Group(func(r chi.Router) {
			r.Use(apiKeyAuth(s.APIKey))

			r.Get("/schedules", s.handleListSchedules)
			r.Post("/schedules", s.handleCreateSchedule)
			r.Put("/schedules/{id}", s.handleUpdateSchedule)
			r.Delete("/schedules/{id}", s.handleDeleteSchedule)

			r.Get("/config", s.handleGetConfig)
			r.Put("/config", s.handleSetConfig)

			r.Get("/captures", s.handleListCaptures)
			r.Get("/captures/{date}/{file}", s.handleServeCapture)
			r.Post("/captures/trigger", s.handleTriggerCapture)
			r.Post("/captures/export", s.handleExportCaptures)
			r.Delete("/captures", s.handleDeleteCaptures)
		})
	})

	// Static files - serve the frontend SPA.
	if s.StaticDir != "" {
		if info, err := os.Stat(s.StaticDir); err == nil && info.IsDir() {
			fileServer := http.FileServer(http.Dir(s.StaticDir))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				w.Header().Del("Content-Type") // Let file server determine it.
				// Try to serve the file; fallback to index.html for SPA routing.
				path := filepath.Join(s.StaticDir, req.URL.Path)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					http.ServeFile(w, req, filepath.Join(s.StaticDir, "index.html"))
					return
				}
				fileServer.ServeHTTP(w, req)
			})
		}
	}

	return r
}

// --- Handlers ---------------------------------------------------------------

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"uptime": time.Since(s.StartTime).String(),
	})
}

// Schedules

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.DB.ListSchedules()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if schedules == nil {
		schedules = []db.Schedule{}
	}
	json.NewEncoder(w).Encode(schedules)
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var sched db.Schedule
	if err := json.NewDecoder(r.Body).Decode(&sched); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	id, err := s.DB.CreateSchedule(sched)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	sched.ID = id

	// Reload the scheduler to pick up the new schedule.
	s.reloadScheduler()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sched)
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var sched db.Schedule
	if err := json.NewDecoder(r.Body).Decode(&sched); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	sched.ID = id

	if err := s.DB.UpdateSchedule(sched); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	s.reloadScheduler()
	json.NewEncoder(w).Encode(sched)
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := s.DB.DeleteSchedule(id); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	s.reloadScheduler()
	w.WriteHeader(http.StatusNoContent)
}

// Config

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.DB.AllConfig()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	for k, v := range body {
		if err := s.DB.SetConfig(k, v); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(body)
}

// Captures

func (s *Server) handleListCaptures(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	captures, err := s.DB.ListCaptures(date, limit)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if captures == nil {
		captures = []db.CaptureLog{}
	}
	json.NewEncoder(w).Encode(captures)
}

func (s *Server) handleServeCapture(w http.ResponseWriter, r *http.Request) {
	date := chi.URLParam(r, "date")
	file := chi.URLParam(r, "file")
	path := filepath.Join(s.DataDir, "captures", date, file)

	// Validate the file exists within captures dir.
	absCaptures, _ := filepath.Abs(filepath.Join(s.DataDir, "captures"))
	absPath, _ := filepath.Abs(path)
	if len(absPath) < len(absCaptures) || absPath[:len(absCaptures)] != absCaptures {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	w.Header().Del("Content-Type") // Let ServeFile determine it.
	http.ServeFile(w, r, path)
}

func (s *Server) handleTriggerCapture(w http.ResponseWriter, r *http.Request) {
	if s.Pipeline == nil {
		http.Error(w, `{"error":"pipeline not configured"}`, http.StatusInternalServerError)
		return
	}

	// Reject if a capture is already in progress.
	if s.Pipeline.Busy() {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "capture already in progress",
		})
		return
	}

	// Use the first enabled schedule to determine camera/hook config,
	// or fall back to a basic mock capture.
	schedules, _ := s.DB.ListSchedules()
	var sched db.Schedule
	found := false
	for _, sc := range schedules {
		if sc.Enabled {
			sched = sc
			found = true
			break
		}
	}
	if !found {
		// Fallback: use global default camera with no hook.
		sched = db.Schedule{CameraID: s.DefaultCamera, Enabled: true}
	}

	// Execute the capture in a goroutine so we don't block the response.
	go s.Pipeline.Execute(sched)

	json.NewEncoder(w).Encode(map[string]string{
		"status":    "triggered",
		"camera_id": sched.CameraID,
	})
}

func (s *Server) handleDeleteCaptures(w http.ResponseWriter, r *http.Request) {
	// Accept IDs as query param (?ids=1,2,3) or JSON body {"ids":[1,2,3]}
	var ids []int64

	idsParam := r.URL.Query().Get("ids")
	if idsParam != "" {
		for _, s := range strings.Split(idsParam, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if err == nil {
				ids = append(ids, id)
			}
		}
	} else {
		var body struct {
			IDs []int64 `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid json - expected {\"ids\":[1,2,3]}"}`, http.StatusBadRequest)
			return
		}
		ids = body.IDs
	}

	if len(ids) == 0 {
		http.Error(w, `{"error":"no ids provided"}`, http.StatusBadRequest)
		return
	}

	// Delete image files from disk before removing DB records.
	for _, id := range ids {
		cap, err := s.DB.GetCapture(id)
		if err != nil || cap == nil {
			continue
		}
		os.Remove(cap.Filepath)
	}

	count, err := s.DB.DeleteCapturesByIDs(ids)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": count,
	})
}

// reloadScheduler fetches all schedules from the DB and reloads the cron scheduler.
func (s *Server) reloadScheduler() {
	if s.Scheduler == nil {
		return
	}
	schedules, err := s.DB.ListSchedules()
	if err != nil {
		return
	}
	s.Scheduler.Load(schedules)
}

// corsMiddleware allows cross-origin requests from the Vite dev server.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
