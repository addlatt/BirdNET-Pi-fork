package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/birdnet-pi/birdnet/internal/scheduler"
	"github.com/go-chi/chi/v5"
)

// ListTasks handles GET /api/tasks - returns all tasks with their current status.
func (h *Handlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	if h.scheduler == nil {
		http.Error(w, "Scheduler not available", http.StatusServiceUnavailable)
		return
	}

	tasks := h.scheduler.ListTasks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
	})
}

// GetTask handles GET /api/tasks/{name} - returns detailed info about a specific task.
func (h *Handlers) GetTask(w http.ResponseWriter, r *http.Request) {
	if h.scheduler == nil {
		http.Error(w, "Scheduler not available", http.StatusServiceUnavailable)
		return
	}

	taskName := chi.URLParam(r, "name")
	if taskName == "" {
		http.Error(w, "Task name required", http.StatusBadRequest)
		return
	}

	info, err := h.scheduler.GetTaskInfo(taskName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// RunTask handles POST /api/tasks/{name}/run - manually triggers a task.
func (h *Handlers) RunTask(w http.ResponseWriter, r *http.Request) {
	if h.scheduler == nil {
		http.Error(w, "Scheduler not available", http.StatusServiceUnavailable)
		return
	}

	taskName := chi.URLParam(r, "name")
	if taskName == "" {
		http.Error(w, "Task name required", http.StatusBadRequest)
		return
	}

	err := h.scheduler.RunTask(taskName, scheduler.TriggerManual)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "started",
		"message": "Task started successfully",
	})
}

// CancelTask handles POST /api/tasks/{name}/cancel - cancels a running task.
func (h *Handlers) CancelTask(w http.ResponseWriter, r *http.Request) {
	if h.scheduler == nil {
		http.Error(w, "Scheduler not available", http.StatusServiceUnavailable)
		return
	}

	taskName := chi.URLParam(r, "name")
	if taskName == "" {
		http.Error(w, "Task name required", http.StatusBadRequest)
		return
	}

	err := h.scheduler.CancelTask(taskName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "cancelled",
		"message": "Task cancellation requested",
	})
}

// GetTaskHistory handles GET /api/tasks/{name}/history - returns execution history.
func (h *Handlers) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
	if h.scheduler == nil || h.taskHistory == nil {
		http.Error(w, "Task history not available", http.StatusServiceUnavailable)
		return
	}

	taskName := chi.URLParam(r, "name")
	if taskName == "" {
		http.Error(w, "Task name required", http.StatusBadRequest)
		return
	}

	// Parse pagination params
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	history, err := h.taskHistory.GetHistory(taskName, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get task history", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	total, err := h.taskHistory.CountHistory(taskName)
	if err != nil {
		total = len(history)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetAllTaskHistory handles GET /api/tasks/history - returns execution history for all tasks.
func (h *Handlers) GetAllTaskHistory(w http.ResponseWriter, r *http.Request) {
	if h.taskHistory == nil {
		http.Error(w, "Task history not available", http.StatusServiceUnavailable)
		return
	}

	// Parse pagination params
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	history, err := h.taskHistory.GetAllHistory(limit, offset)
	if err != nil {
		http.Error(w, "Failed to get task history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"limit":   limit,
		"offset":  offset,
	})
}
