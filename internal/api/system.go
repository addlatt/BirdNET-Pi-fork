package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
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
const version = "1.0.0"

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

// UpdateCheckResponse represents the update check result.
type UpdateCheckResponse struct {
	CurrentCommit   string `json:"current_commit"`
	LatestCommit    string `json:"latest_commit"`
	BehindCount     int    `json:"behind_count"`
	UpdateAvailable bool   `json:"update_available"`
}

// CheckForUpdates handles GET /api/system/update-check requests.
func (h *Handlers) CheckForUpdates(w http.ResponseWriter, r *http.Request) {
	// Fetch latest from origin
	fetchCmd := exec.CommandContext(r.Context(), "git", "fetch", "origin")
	if err := fetchCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch from origin: "+err.Error())
		return
	}

	// Get current commit
	currentCmd := exec.CommandContext(r.Context(), "git", "rev-parse", "HEAD")
	currentOutput, err := currentCmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get current commit: "+err.Error())
		return
	}
	currentCommit := strings.TrimSpace(string(currentOutput))

	// Get latest commit on origin/main
	latestCmd := exec.CommandContext(r.Context(), "git", "rev-parse", "origin/main")
	latestOutput, err := latestCmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get latest commit: "+err.Error())
		return
	}
	latestCommit := strings.TrimSpace(string(latestOutput))

	// Get count of commits behind
	behindCmd := exec.CommandContext(r.Context(), "git", "rev-list", "HEAD..origin/main", "--count")
	behindOutput, err := behindCmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get behind count: "+err.Error())
		return
	}
	behindStr := strings.TrimSpace(string(behindOutput))
	behindCount := 0
	for _, c := range behindStr {
		if c >= '0' && c <= '9' {
			behindCount = behindCount*10 + int(c-'0')
		}
	}

	response := UpdateCheckResponse{
		CurrentCommit:   currentCommit,
		LatestCommit:    latestCommit,
		BehindCount:     behindCount,
		UpdateAvailable: currentCommit != latestCommit,
	}

	writeJSON(w, http.StatusOK, response)
}

// ConfirmationRequest is used for dangerous operations that require confirmation.
type ConfirmationRequest struct {
	Confirm bool `json:"confirm"`
}

// Reboot handles POST /api/system/reboot requests.
func (h *Handlers) Reboot(w http.ResponseWriter, r *http.Request) {
	var req ConfirmationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !req.Confirm {
		writeError(w, http.StatusBadRequest, "confirmation required: set confirm=true")
		return
	}

	// Send response before rebooting
	writeJSON(w, http.StatusOK, map[string]string{"status": "rebooting"})

	// Reboot in background after response is sent
	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("sudo", "reboot").Run()
	}()
}

// Shutdown handles POST /api/system/shutdown requests.
func (h *Handlers) Shutdown(w http.ResponseWriter, r *http.Request) {
	var req ConfirmationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !req.Confirm {
		writeError(w, http.StatusBadRequest, "confirmation required: set confirm=true")
		return
	}

	// Send response before shutting down
	writeJSON(w, http.StatusOK, map[string]string{"status": "shutting_down"})

	// Shutdown in background after response is sent
	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("sudo", "shutdown", "-h", "now").Run()
	}()
}
