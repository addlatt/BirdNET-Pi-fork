package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/birdnet-pi/birdnet/internal/api"
	"github.com/birdnet-pi/birdnet/internal/config"
	"github.com/birdnet-pi/birdnet/internal/db"
	"github.com/birdnet-pi/birdnet/internal/mlclient"
	"github.com/birdnet-pi/birdnet/internal/monitor"
	"github.com/birdnet-pi/birdnet/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	handlers := api.NewHandlers(database, hub, memMonitor, mlClient, configMgr, scriptsDir, dataDir, birdsongsDir)

	// Public API routes
	r.Route("/api", func(r chi.Router) {
		// Health check
		r.Get("/health", handlers.Health)

		// Detections
		r.Get("/detections", handlers.ListDetections)
		r.Get("/detections/{date}/{time}/{species}", handlers.GetDetection)
		r.Delete("/detections/{date}/{time}/{species}", handlers.DeleteDetection)

		// Dates (for history page date picker)
		r.Get("/dates", handlers.ListDates)

		// Species
		r.Get("/species", handlers.ListSpecies)
		r.Get("/species/all", handlers.ListAllSpecies)
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

		// Settings
		r.Get("/settings", handlers.GetSettings)
		r.Put("/settings", handlers.UpdateSettings)
		r.Get("/settings/schema", handlers.GetSettingsSchema)

		// Services
		r.Get("/services", handlers.ListServices)
		r.Post("/services/restart-all", handlers.RestartAllServices)
		r.Post("/services/{name}/{action}", handlers.ServiceAction)
	})

	// Internal routes (Python → Go)
	r.Route("/internal", func(r chi.Router) {
		r.Post("/detection", handlers.ReceiveDetection)
	})

	// WebSocket endpoint
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	})

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

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

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
