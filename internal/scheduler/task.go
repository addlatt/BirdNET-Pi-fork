// Package scheduler provides task scheduling functionality for BirdNET-Pi.
// It replaces cron-based shell scripts with a Go-native task scheduler
// that integrates with the API server for real-time status and control.
package scheduler

import (
	"context"
	"time"
)

// Task defines the interface that all scheduled tasks must implement.
type Task interface {
	// Name returns the unique identifier for this task.
	Name() string

	// Description returns a human-readable description of what this task does.
	Description() string

	// DefaultSchedule returns the default cron expression for this task.
	// An empty string means the task is manual-only (no automatic scheduling).
	DefaultSchedule() string

	// Timeout returns the maximum duration this task is allowed to run.
	Timeout() time.Duration

	// Run executes the task. The context is cancelled if the task is cancelled
	// or if the timeout is exceeded.
	Run(ctx context.Context) error
}

// TaskStatus represents the current execution status of a task.
type TaskStatus string

const (
	// TaskStatusIdle indicates the task is not currently running.
	TaskStatusIdle TaskStatus = "idle"

	// TaskStatusRunning indicates the task is currently executing.
	TaskStatusRunning TaskStatus = "running"

	// TaskStatusCompleted indicates the task finished successfully.
	TaskStatusCompleted TaskStatus = "completed"

	// TaskStatusFailed indicates the task finished with an error.
	TaskStatusFailed TaskStatus = "failed"

	// TaskStatusCancelled indicates the task was manually cancelled.
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TriggerType indicates how a task execution was initiated.
type TriggerType string

const (
	// TriggerScheduled indicates the task was started by the scheduler.
	TriggerScheduled TriggerType = "scheduled"

	// TriggerManual indicates the task was started via API.
	TriggerManual TriggerType = "manual"
)

// TaskInfo provides current information about a task.
type TaskInfo struct {
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Schedule        string     `json:"schedule"`         // Current schedule (may differ from default)
	Enabled         bool       `json:"enabled"`          // Whether automatic scheduling is enabled
	Status          TaskStatus `json:"status"`           // Current execution status
	LastRun         *time.Time `json:"last_run"`         // Time of last execution
	LastStatus      TaskStatus `json:"last_status"`      // Status of last execution
	LastError       string     `json:"last_error"`       // Error message from last failed execution
	LastDurationMs  int64      `json:"last_duration_ms"` // Duration of last execution in milliseconds
	NextRun         *time.Time `json:"next_run"`         // Next scheduled run time
	CurrentRunStart *time.Time `json:"current_run_start,omitempty"` // Start time of current run (if running)
}

// TaskExecution represents a single execution of a task (for history).
type TaskExecution struct {
	ID          int64       `json:"id"`
	TaskName    string      `json:"task_name"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt time.Time   `json:"completed_at"`
	DurationMs  int64       `json:"duration_ms"`
	Status      TaskStatus  `json:"status"`
	Error       string      `json:"error,omitempty"`
	Trigger     TriggerType `json:"trigger"`
}

// RunningTask holds information about a currently executing task.
type RunningTask struct {
	Task      Task
	StartedAt time.Time
	Trigger   TriggerType
	Cancel    context.CancelFunc
}
