package api

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/birdnet-pi/birdnet/internal/config"
	"github.com/go-chi/chi/v5"
)

// ListServices handles GET /api/services requests.
// Returns the status of all managed BirdNET-Pi services.
func (h *Handlers) ListServices(w http.ResponseWriter, r *http.Request) {
	services := make([]config.ServiceStatus, 0, len(config.ManagedServices))

	for _, svc := range config.ManagedServices {
		status := getServiceStatus(svc.Name)
		status.DisplayName = svc.DisplayName
		services = append(services, status)
	}

	writeJSON(w, http.StatusOK, config.ServicesResponse{
		Services: services,
	})
}

// ServiceAction handles POST /api/services/{name}/{action} requests.
// Performs start/stop/restart/enable/disable on a specific service.
func (h *Handlers) ServiceAction(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "name")
	action := chi.URLParam(r, "action")

	if serviceName == "" || action == "" {
		writeError(w, http.StatusBadRequest, "Missing service name or action")
		return
	}

	// Validate service name (security: only allow managed services)
	if !isValidService(serviceName) {
		writeError(w, http.StatusBadRequest, "Invalid service name")
		return
	}

	// Validate action
	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"enable":  true,
		"disable": true,
	}
	if !validActions[action] {
		writeError(w, http.StatusBadRequest, "Invalid action. Must be one of: start, stop, restart, enable, disable")
		return
	}

	// Execute systemctl command
	var cmd *exec.Cmd
	switch action {
	case "enable", "disable":
		// Enable/disable don't need sudo in most cases if using systemd user units
		// but for system services they do
		cmd = exec.Command("sudo", "systemctl", action, serviceName)
	default:
		cmd = exec.Command("sudo", "systemctl", action, serviceName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, config.ServiceActionResponse{
			Status:  "error",
			Message: "Failed to " + action + " service: " + err.Error(),
			Output:  string(output),
		})
		return
	}

	writeJSON(w, http.StatusOK, config.ServiceActionResponse{
		Status:  "success",
		Message: "Service " + serviceName + " " + action + " successful",
		Output:  string(output),
	})
}

// RestartAllServices handles POST /api/services/restart-all requests.
// Restarts all BirdNET-Pi services using the main restart script.
func (h *Handlers) RestartAllServices(w http.ResponseWriter, r *http.Request) {
	if h.configMgr == nil {
		writeError(w, http.StatusServiceUnavailable, "Configuration manager not available")
		return
	}

	if err := h.configMgr.RestartAllServices(); err != nil {
		writeJSON(w, http.StatusInternalServerError, config.ServiceActionResponse{
			Status:  "error",
			Message: "Failed to restart all services: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, config.ServiceActionResponse{
		Status:  "success",
		Message: "All services restart initiated",
	})
}

// getServiceStatus retrieves the status of a systemd service.
func getServiceStatus(serviceName string) config.ServiceStatus {
	status := config.ServiceStatus{
		Name:   serviceName,
		Status: "unknown",
	}

	// Check if service is active
	activeCmd := exec.Command("systemctl", "is-active", serviceName)
	activeOutput, _ := activeCmd.Output()
	activeStatus := strings.TrimSpace(string(activeOutput))

	switch activeStatus {
	case "active":
		status.Status = "active"
	case "inactive":
		status.Status = "inactive"
	case "failed":
		status.Status = "failed"
	default:
		status.Status = activeStatus
	}

	// Check if service is enabled
	enabledCmd := exec.Command("systemctl", "is-enabled", serviceName)
	enabledOutput, _ := enabledCmd.Output()
	status.Enabled = strings.TrimSpace(string(enabledOutput)) == "enabled"

	// For recording service, check if there's a backlog
	if serviceName == "birdnet_recording.service" && status.Status == "active" {
		backlog := getRecordingBacklog()
		if backlog > 0 {
			status.Message = strconv.Itoa(backlog) + " files in backlog"
			if backlog > 100 {
				status.Status = "stalled"
			}
		}
	}

	return status
}

// getRecordingBacklog counts the number of files waiting to be processed.
func getRecordingBacklog() int {
	// Count files in the recording buffer directory
	// This mirrors the PHP behavior of checking for backlog
	cmd := exec.Command("sh", "-c", "ls -1 ~/BirdSongs/Extracted/*.wav 2>/dev/null | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}

	return count
}

// isValidService checks if a service name is in our managed services list.
func isValidService(name string) bool {
	for _, svc := range config.ManagedServices {
		if svc.Name == name {
			return true
		}
	}
	return false
}
