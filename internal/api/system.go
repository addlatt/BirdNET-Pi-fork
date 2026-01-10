package api

import (
	"context"
	"net/http"
	"runtime"
	"time"
)

// SystemStatusResponse represents the system status.
type SystemStatusResponse struct {
	Status          string                 `json:"status"`
	Version         string                 `json:"version"`
	GoVersion       string                 `json:"go_version"`
	Uptime          int64                  `json:"uptime_seconds"`
	MLServiceStatus string                 `json:"ml_service_status"`
	WebSocketClients int                   `json:"websocket_clients"`
	Database        DatabaseStatusResponse `json:"database"`
}

// DatabaseStatusResponse represents database status.
type DatabaseStatusResponse struct {
	Connected bool   `json:"connected"`
	Path      string `json:"path,omitempty"`
}

// SystemMemoryResponse represents memory usage information.
type SystemMemoryResponse struct {
	Go     GoMemoryResponse     `json:"go"`
	System SystemMemResponse    `json:"system"`
	ML     MLMemoryResponse     `json:"ml,omitempty"`
}

// GoMemoryResponse represents Go runtime memory.
type GoMemoryResponse struct {
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapInUse    uint64 `json:"heap_in_use"`
	StackInUse   uint64 `json:"stack_in_use"`
	NumGoroutine int    `json:"num_goroutine"`
}

// SystemMemResponse represents system memory.
type SystemMemResponse struct {
	Total     uint64 `json:"total"`
	Free      uint64 `json:"free"`
	Available uint64 `json:"available"`
	Used      uint64 `json:"used"`
}

// MLMemoryResponse represents ML service memory.
type MLMemoryResponse struct {
	BirdNET uint64 `json:"birdnet"`
	VAD     uint64 `json:"vad"`
	LLM     uint64 `json:"llm"`
	Total   uint64 `json:"total"`
}

var startTime = time.Now()
const version = "0.1.0"

// SystemStatus handles GET /api/system/status requests.
func (h *Handlers) SystemStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check ML service status
	mlStatus := "unknown"
	if h.mlClient != nil {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if h.mlClient.IsHealthy(ctx) {
			mlStatus = "healthy"
		} else {
			mlStatus = "unhealthy"
		}
	}

	// Check database connection
	dbConnected := true
	if h.db != nil {
		if err := h.db.Conn().Ping(); err != nil {
			dbConnected = false
		}
	}

	response := SystemStatusResponse{
		Status:           "running",
		Version:          version,
		GoVersion:        runtime.Version(),
		Uptime:           int64(time.Since(startTime).Seconds()),
		MLServiceStatus:  mlStatus,
		WebSocketClients: h.hub.GetClientCount(),
		Database: DatabaseStatusResponse{
			Connected: dbConnected,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// SystemMemory handles GET /api/system/memory requests.
func (h *Handlers) SystemMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get memory stats from monitor
	stats := h.monitor.GetStats()

	response := SystemMemoryResponse{
		Go: GoMemoryResponse{
			HeapAlloc:    stats.GoHeapAlloc,
			HeapSys:      stats.GoHeapSys,
			HeapInUse:    stats.GoHeapInUse,
			StackInUse:   stats.GoStackInUse,
			NumGoroutine: stats.GoNumGoroutine,
		},
		System: SystemMemResponse{
			Total:     stats.SystemTotal,
			Free:      stats.SystemFree,
			Available: stats.SystemAvailable,
			Used:      stats.SystemUsed,
		},
	}

	// Try to get ML service memory
	if h.mlClient != nil {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if mlStats, err := h.mlClient.GetMemoryUsage(ctx); err == nil {
			response.ML = MLMemoryResponse{
				BirdNET: mlStats.BirdNET,
				VAD:     mlStats.VAD,
				LLM:     mlStats.LLM,
				Total:   mlStats.Total,
			}
		}
	}

	writeJSON(w, http.StatusOK, response)
}
