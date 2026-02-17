package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/leshicodes/vigil/internal/api"
	"github.com/leshicodes/vigil/internal/capture"
	"github.com/leshicodes/vigil/internal/db"
	"github.com/leshicodes/vigil/internal/logger"
	"github.com/leshicodes/vigil/internal/scheduler"
)

func main() {
	var (
		dataDir   = flag.String("data-dir", envOrDefault("VIGIL_DATA_DIR", "./data"), "path to data directory")
		camera    = flag.String("camera", envOrDefault("VIGIL_CAMERA", "mock"), "camera driver: mock, libcamera, fswebcam, ffmpeg")
		port      = flag.String("port", envOrDefault("VIGIL_PORT", "8080"), "HTTP server port")
		staticDir = flag.String("static-dir", envOrDefault("VIGIL_STATIC_DIR", "./web/dist"), "path to frontend static files")
		apiKey    = flag.String("api-key", envOrDefault("VIGIL_API_KEY", ""), "API key for auth (empty = auth disabled)")
	)
	flag.Parse()

	// Initialize structured logging.
	logger.SetLevel(os.Getenv("VIGIL_LOG_LEVEL"))
	logger.Info("vigil", "starting — data=%s camera=%s port=%s log_level=%s",
		*dataDir, *camera, *port, logger.GetLevel())

	// Ensure data directory exists.
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v (check permissions)", err)
	}

	// Open database.
	dbPath := filepath.Join(*dataDir, "vigil.db")
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v - this often means the data directory isn't writable by the vigil user", err)
	}
	defer database.Close()
	logger.Info("vigil", "database opened: %s", dbPath)

	// Seed a default schedule if none exist.
	schedules, err := database.ListSchedules()
	if err != nil {
		log.Fatalf("list schedules: %v", err)
	}
	if len(schedules) == 0 {
		id, err := database.CreateSchedule(db.Schedule{
			CronExpr: "*/30 * * * *", // Every 30 minutes
			CameraID: *camera,
			Enabled:  true,
		})
		if err != nil {
			log.Fatalf("seed schedule: %v", err)
		}
		logger.Info("vigil", "seeded default schedule (id=%d): every 30 minutes using %s", id, *camera)

		// Refresh the list.
		schedules, _ = database.ListSchedules()
	}

	// Sync schedule camera_id with the configured camera driver.
	// This prevents the "stale DB" trap where VIGIL_CAMERA=fswebcam is set
	// but existing schedules still reference the old driver (e.g. "mock").
	for _, s := range schedules {
		if s.CameraID != *camera {
			logger.Warn("vigil", "schedule %d has camera_id=%q but VIGIL_CAMERA=%q — updating to match",
				s.ID, s.CameraID, *camera)
			s.CameraID = *camera
			if err := database.UpdateSchedule(s); err != nil {
				logger.Error("vigil", "failed to update schedule %d camera_id: %v", s.ID, err)
			}
		}
	}

	// Refresh schedules after potential updates.
	schedules, _ = database.ListSchedules()

	// Build the capture pipeline.
	pipeline := &capture.Pipeline{
		DataDir: *dataDir,
		DB:      database,
	}

	// Start the scheduler.
	sched := scheduler.New(func(s db.Schedule) {
		logger.Info("scheduler", "triggering capture for schedule %d (driver: %s)", s.ID, s.CameraID)
		pipeline.Execute(s)
	})
	if err := sched.Load(schedules); err != nil {
		log.Fatalf("load schedules: %v", err)
	}
	sched.Start()
	defer sched.Stop()

	// Start the HTTP server.
	srv := &api.Server{
		DB:            database,
		Scheduler:     sched,
		Pipeline:      pipeline,
		DataDir:       *dataDir,
		StaticDir:     *staticDir,
		APIKey:        *apiKey,
		DefaultCamera: *camera,
		StartTime:     time.Now(),
	}

	httpServer := &http.Server{
		Addr:    ":" + *port,
		Handler: srv.NewRouter(),
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("vigil", "HTTP server listening on :%s", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-done
	logger.Info("vigil", "shutting down...")
	httpServer.Close()
	logger.Info("vigil", "goodbye")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func init() {
	// Print banner.
	fmt.Println(`
 ╦  ╦╦╔═╗╦╦  
 ╚╗╔╝║║ ╦║║  
  ╚╝ ╩╚═╝╩╩═╝
  Time-Lapse Monitor`)
}
