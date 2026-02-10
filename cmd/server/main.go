package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/birdnet-pi/birdnet/internal/api"
	"github.com/birdnet-pi/birdnet/internal/config"
	"github.com/birdnet-pi/birdnet/internal/db"
	"github.com/birdnet-pi/birdnet/internal/mlclient"
	"github.com/birdnet-pi/birdnet/internal/monitor"
	"github.com/birdnet-pi/birdnet/internal/scheduler"
	"github.com/birdnet-pi/birdnet/internal/tasks"
	"github.com/birdnet-pi/birdnet/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for scheduler history
)

func main() {
	// Configuration
	port := getEnv("PORT", "8080")
	dbPath := getEnv("DB_PATH", "data/db/birds.db")
	mlServiceURL := getEnv("ML_SERVICE_URL", "http://127.0.0.1:8001")
	scriptsDir := getEnv("SCRIPTS_DIR", "scripts")
	dataDir := getEnv("DATA_DIR", "data")
	birdsongsDir := getEnv("BIRDSONGS_DIR", expandHome("~/BirdSongs"))
	configPath := getEnv("CONFIG_PATH", config.DefaultConfigPath)
	homeDir := getEnv("HOME", expandHome("~"))

	// Initialize configuration manager
	configMgr := config.NewManager(configPath, homeDir)
	if err := configMgr.Load(); err != nil {
		log.Printf("Warning: Failed to load configuration: %v (settings API may not work)", err)
	}

	// Initialize database (read-only)
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Initialize memory monitor
	memMonitor := monitor.NewMemoryMonitor()

	// Initialize ML client
	mlClient := mlclient.New(mlServiceURL)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS middleware
	r.Use(corsMiddleware)

	// Initialize API handlers
	handlers := api.NewHandlers(database, hub, memMonitor, mlClient, configMgr, scriptsDir, dataDir, birdsongsDir, homeDir)

	// Initialize task scheduler
	taskScheduler, taskHistory := initScheduler(database, hub, configMgr, homeDir, birdsongsDir, scriptsDir, dataDir)
	handlers.SetScheduler(taskScheduler, taskHistory)

	// Public API routes
	r.Route("/api", func(r chi.Router) {
		// Health check
		r.Get("/health", handlers.Health)

		// Detections
		r.Get("/detections", handlers.ListDetections)
		r.Get("/detections/{date}/{time}/{species}", handlers.GetDetection)
		r.Delete("/detections/{date}/{time}/{species}", handlers.DeleteDetection)
		r.Post("/detections/reclassify", handlers.ReclassifyDetection)

		// Dates (for history page date picker)
		r.Get("/dates", handlers.ListDates)

		// Species
		r.Get("/species", handlers.ListSpecies)
		r.Get("/species/all", handlers.ListAllSpecies)
		r.Get("/species/ranking", handlers.GetSpeciesRanking)
		r.Get("/species/{name}", handlers.GetSpeciesDetail)
		r.Get("/species/{name}/history", handlers.GetSpeciesHistory)
		r.Get("/species/{name}/count", handlers.GetSpeciesCount)
		r.Delete("/species/{name}/all", handlers.DeleteAllSpeciesDetections)

		// Species Lists (confirmed, excluded, whitelisted, include)
		r.Get("/species-lists", handlers.GetSpeciesLists)
		r.Put("/species-lists/{listType}", handlers.UpdateSpeciesList)
		r.Post("/species-lists/{listType}/add", handlers.AddToSpeciesList)
		r.Post("/species-lists/{listType}/remove", handlers.RemoveFromSpeciesList)

		// Labels (all available species from labels.txt)
		r.Get("/labels", handlers.GetLabels)

		// Stats
		r.Get("/stats", handlers.GetStats)

		// Heatmap
		r.Get("/heatmap/today", handlers.GetHeatmapToday)

		// Spectrogram
		r.Get("/spectrogram/info", handlers.GetSpectrogramInfo)
		r.Get("/spectrogram/image", handlers.GetSpectrogramImage)
		r.Get("/spectrogram/detections", handlers.GetRecentDetections)
		r.Get("/stream", handlers.ProxyLivestream)

		// System
		r.Get("/system/status", handlers.SystemStatus)
		r.Get("/system/memory", handlers.SystemMemory)
		r.Get("/system/update-check", handlers.CheckForUpdates)
		r.Post("/system/reboot", handlers.Reboot)
		r.Post("/system/shutdown", handlers.Shutdown)

		// Reports
		r.Get("/reports/weekly", handlers.WeeklyReport)
		r.Get("/reports/weekly/export", handlers.ExportWeeklyReport)

		// Diagnostics (replaces shell scripts)
		r.Get("/diagnostics/disk", handlers.DiskUsage)
		r.Get("/diagnostics/most-recent", handlers.MostRecent)
		r.Get("/diagnostics/pi", handlers.PiDiagnostics)
		r.Get("/diagnostics/system", handlers.SystemDiagnostics)
		r.Get("/diagnostics/species-count", handlers.SpeciesCount)
		r.Get("/diagnostics/logs", handlers.DumpLogs)

		// Logs API (recent logs without streaming)
		r.Get("/logs/recent", handlers.GetRecentLogs)

		// Settings
		r.Get("/settings", handlers.GetSettings)
		r.Put("/settings", handlers.UpdateSettings)
		r.Get("/settings/schema", handlers.GetSettingsSchema)
		r.Post("/settings/caddy/regenerate", handlers.RegenerateCaddyfile)

		// Labels (available species for reclassification)
		r.Get("/labels/model", handlers.GetModelLabels)

		// Services
		r.Get("/services", handlers.ListServices)
		r.Post("/services/restart-all", handlers.RestartAllServices)
		r.Post("/services/{name}/{action}", handlers.ServiceAction)

		// Recordings (Play/Audio page)
		r.Get("/recordings/dates", handlers.ListRecordingDates)
		r.Get("/recordings/species", handlers.ListRecordingSpecies)
		r.Get("/recordings/by-date/{date}", handlers.ListRecordingsByDate)
		r.Get("/recordings/by-species/{name}", handlers.ListRecordingsBySpecies)
		r.Post("/recordings/{date}/{species}/{filename}/delete", handlers.DeleteRecording)
		r.Post("/recordings/{date}/{species}/{filename}/change", handlers.ChangeRecordingIdentification)
		r.Post("/recordings/{date}/{species}/{filename}/lock", handlers.ToggleRecordingLock)
		r.Post("/recordings/{date}/{species}/{filename}/shift", handlers.ToggleRecordingShift)
		r.Get("/recordings/exclusions", handlers.GetExclusionList)

		// Task scheduler
		r.Get("/tasks", handlers.ListTasks)
		r.Get("/tasks/history", handlers.GetAllTaskHistory)
		r.Get("/tasks/{name}", handlers.GetTask)
		r.Post("/tasks/{name}/run", handlers.RunTask)
		r.Post("/tasks/{name}/cancel", handlers.CancelTask)
		r.Get("/tasks/{name}/history", handlers.GetTaskHistory)

		// Backup/Restore
		r.Post("/backup/create", handlers.CreateBackup)
		r.Post("/backup/restore", handlers.RestoreBackup)
		r.Get("/backup/status", handlers.GetRestoreStatus)
	})

	// Internal routes (Python → Go)
	r.Route("/internal", func(r chi.Router) {
		r.Post("/detection", handlers.ReceiveDetection)
	})

	// WebSocket endpoints
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	})
	r.Get("/ws/logs", handlers.StreamLogs)
	r.Get("/ws/logs/detections", handlers.StreamDetectionLogs)

	// Static file server for Preact app (in production)
	staticDir := getEnv("STATIC_DIR", "web/dist")
	if _, err := os.Stat(staticDir); err == nil {
		fileServer := http.FileServer(http.Dir(staticDir))
		r.Handle("/app/*", http.StripPrefix("/app", fileServer))
	}

	// Static file server for bird recordings and spectrograms
	extractedDir := birdsongsDir + "/Extracted"
	if _, err := os.Stat(extractedDir); err == nil {
		birdFileServer := http.FileServer(http.Dir(extractedDir))
		r.Handle("/By_Date/*", http.StripPrefix("", birdFileServer))
		r.Handle("/Charts/*", http.StripPrefix("", birdFileServer))
	}

	// Create server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting BirdNET-Pi server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Start the scheduler
	if taskScheduler != nil {
		if err := taskScheduler.Start(); err != nil {
			log.Printf("Warning: Failed to start scheduler: %v", err)
		}
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Stop scheduler first
	if taskScheduler != nil {
		log.Println("Stopping scheduler...")
		schedulerCtx := taskScheduler.Stop()
		<-schedulerCtx.Done()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// initScheduler creates and configures the task scheduler with all tasks.
func initScheduler(database *db.DB, hub *ws.Hub, configMgr *config.Manager, homeDir, birdsongsDir, scriptsDir, dataDir string) (*scheduler.Scheduler, *scheduler.HistoryStore) {
	// Open a separate database connection for the scheduler (read-write for history)
	historyDBPath := filepath.Join(dataDir, "db", "birds.db")
	historyDB, err := sql.Open("sqlite3", historyDBPath)
	if err != nil {
		log.Printf("Warning: Failed to open history database: %v (task history will not be persisted)", err)
		return nil, nil
	}

	// Create history store and ensure table exists
	historyStore := scheduler.NewHistoryStore(historyDB)
	if err := historyStore.EnsureTable(); err != nil {
		log.Printf("Warning: Failed to create task_history table: %v", err)
	}

	// Create task registry and register all tasks
	registry := scheduler.NewRegistry()

	// Paths for tasks
	extractedDir := filepath.Join(birdsongsDir, "Extracted")
	processedDir := filepath.Join(birdsongsDir, "Processed")

	// Register disk cleanup task
	diskCleanup := tasks.NewDiskCleanupTask(configMgr, extractedDir, processedDir, dataDir)
	registry.MustRegister(diskCleanup)

	// Register weekly report task
	weeklyReport := tasks.NewWeeklyReportTask(scriptsDir)
	registry.MustRegister(weeklyReport)

	// Register species cleanup task (needs DB access for species list)
	speciesCleanup := tasks.NewSpeciesCleanupTask(configMgr, historyDB, extractedDir, dataDir)
	registry.MustRegister(speciesCleanup)

	// Register backup task
	backup := tasks.NewBackupTask(homeDir, birdsongsDir)
	registry.MustRegister(backup)

	// Register data reset task (manual only, no automatic schedule)
	dataReset := tasks.NewDataResetTask(configMgr, historyDB, homeDir, birdsongsDir, dataDir)
	registry.MustRegister(dataReset)

	// Register species list update task
	speciesListUpdate := tasks.NewSpeciesListUpdateTask(configMgr, historyDB, homeDir)
	registry.MustRegister(speciesListUpdate)

	// Register notification task
	notification := tasks.NewNotificationTask(configMgr, historyDB, homeDir)
	registry.MustRegister(notification)

	// Register service control tasks (manual only)
	restartServices := tasks.NewRestartServicesTask()
	registry.MustRegister(restartServices)

	stopCoreServices := tasks.NewStopCoreServicesTask()
	registry.MustRegister(stopCoreServices)

	// Create scheduler
	sched := scheduler.NewScheduler(registry, historyStore, hub, configMgr)

	log.Printf("Task scheduler initialized with %d tasks", registry.Count())
	return sched, historyStore
}
