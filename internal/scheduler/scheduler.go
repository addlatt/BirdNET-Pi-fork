package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
	"github.com/birdnet-pi/birdnet/internal/ws"
	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled task execution.
type Scheduler struct {
	cron      *cron.Cron
	registry  *Registry
	history   *HistoryStore
	hub       *ws.Hub
	configMgr *config.Manager

	// running tracks currently executing tasks
	running map[string]*RunningTask
	mu      sync.RWMutex

	// cronEntries maps task names to cron entry IDs for removal
	cronEntries map[string]cron.EntryID
	cronMu      sync.Mutex

	// lastExecution caches the last execution result for each task
	lastExecution map[string]*TaskExecution
	lastMu        sync.RWMutex
}

// NewScheduler creates a new scheduler with the given dependencies.
func NewScheduler(registry *Registry, history *HistoryStore, hub *ws.Hub, configMgr *config.Manager) *Scheduler {
	return &Scheduler{
		cron:          cron.New(cron.WithLocation(time.Local)),
		registry:      registry,
		history:       history,
		hub:           hub,
		configMgr:     configMgr,
		running:       make(map[string]*RunningTask),
		cronEntries:   make(map[string]cron.EntryID),
		lastExecution: make(map[string]*TaskExecution),
	}
}

// Start initializes the scheduler and begins executing scheduled tasks.
func (s *Scheduler) Start() error {
	// Schedule all enabled tasks
	for _, task := range s.registry.List() {
		if err := s.scheduleTask(task); err != nil {
			log.Printf("Failed to schedule task %s: %v", task.Name(), err)
		}
	}

	s.cron.Start()
	log.Printf("Scheduler started with %d tasks", s.registry.Count())
	return nil
}

// Stop gracefully stops the scheduler, waiting for running tasks to complete.
func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}

// scheduleTask adds a task to the cron scheduler if enabled.
func (s *Scheduler) scheduleTask(task Task) error {
	taskName := task.Name()
	schedule := s.getTaskSchedule(taskName)
	enabled := s.isTaskEnabled(taskName)

	if !enabled || schedule == "" {
		log.Printf("Task %s is disabled or has no schedule", taskName)
		return nil
	}

	s.cronMu.Lock()
	defer s.cronMu.Unlock()

	// Remove existing entry if any
	if entryID, exists := s.cronEntries[taskName]; exists {
		s.cron.Remove(entryID)
		delete(s.cronEntries, taskName)
	}

	entryID, err := s.cron.AddFunc(schedule, func() {
		if err := s.RunTask(taskName, TriggerScheduled); err != nil {
			log.Printf("Scheduled task %s failed: %v", taskName, err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron entry: %w", err)
	}

	s.cronEntries[taskName] = entryID
	log.Printf("Scheduled task %s with schedule %q", taskName, schedule)
	return nil
}

// RunTask executes a task immediately.
// Returns an error if the task is not found or is already running.
func (s *Scheduler) RunTask(taskName string, trigger TriggerType) error {
	task := s.registry.Get(taskName)
	if task == nil {
		return fmt.Errorf("task %q not found", taskName)
	}

	// Check if already running
	s.mu.Lock()
	if _, running := s.running[taskName]; running {
		s.mu.Unlock()
		return fmt.Errorf("task %q is already running", taskName)
	}

	// Create cancellable context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout())
	startedAt := time.Now()

	s.running[taskName] = &RunningTask{
		Task:      task,
		StartedAt: startedAt,
		Trigger:   trigger,
		Cancel:    cancel,
	}
	s.mu.Unlock()

	// Broadcast task started event
	s.broadcastTaskEvent("task_started", taskName, TaskStatusRunning, "", trigger)

	// Run task in goroutine
	go func() {
		defer cancel()

		err := task.Run(ctx)
		completedAt := time.Now()
		duration := completedAt.Sub(startedAt)

		// Determine status
		var status TaskStatus
		var errMsg string
		if err != nil {
			if ctx.Err() == context.Canceled {
				status = TaskStatusCancelled
				errMsg = "task cancelled"
			} else if ctx.Err() == context.DeadlineExceeded {
				status = TaskStatusFailed
				errMsg = "task timeout exceeded"
			} else {
				status = TaskStatusFailed
				errMsg = err.Error()
			}
		} else {
			status = TaskStatusCompleted
		}

		// Remove from running
		s.mu.Lock()
		delete(s.running, taskName)
		s.mu.Unlock()

		// Save to history
		execution := &TaskExecution{
			TaskName:    taskName,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			DurationMs:  duration.Milliseconds(),
			Status:      status,
			Error:       errMsg,
			Trigger:     trigger,
		}

		if s.history != nil {
			if err := s.history.SaveExecution(execution); err != nil {
				log.Printf("Failed to save task execution history: %v", err)
			}
		}

		// Cache last execution
		s.lastMu.Lock()
		s.lastExecution[taskName] = execution
		s.lastMu.Unlock()

		// Broadcast completion event
		eventType := "task_completed"
		if status == TaskStatusFailed {
			eventType = "task_failed"
		} else if status == TaskStatusCancelled {
			eventType = "task_cancelled"
		}
		s.broadcastTaskEvent(eventType, taskName, status, errMsg, trigger)

		log.Printf("Task %s %s in %v", taskName, status, duration)
	}()

	return nil
}

// CancelTask cancels a running task.
func (s *Scheduler) CancelTask(taskName string) error {
	s.mu.RLock()
	running, exists := s.running[taskName]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %q is not running", taskName)
	}

	running.Cancel()
	return nil
}

// GetTaskInfo returns current information about a task.
func (s *Scheduler) GetTaskInfo(taskName string) (*TaskInfo, error) {
	task := s.registry.Get(taskName)
	if task == nil {
		return nil, fmt.Errorf("task %q not found", taskName)
	}

	info := &TaskInfo{
		Name:        taskName,
		Description: task.Description(),
		Schedule:    s.getTaskSchedule(taskName),
		Enabled:     s.isTaskEnabled(taskName),
		Status:      TaskStatusIdle,
	}

	// Check if running
	s.mu.RLock()
	if running, exists := s.running[taskName]; exists {
		info.Status = TaskStatusRunning
		info.CurrentRunStart = &running.StartedAt
	}
	s.mu.RUnlock()

	// Get last execution info
	s.lastMu.RLock()
	if last, exists := s.lastExecution[taskName]; exists {
		info.LastRun = &last.CompletedAt
		info.LastStatus = last.Status
		info.LastError = last.Error
		info.LastDurationMs = last.DurationMs
	}
	s.lastMu.RUnlock()

	// Get next scheduled run
	s.cronMu.Lock()
	if entryID, exists := s.cronEntries[taskName]; exists {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			info.NextRun = &entry.Next
		}
	}
	s.cronMu.Unlock()

	return info, nil
}

// ListTasks returns information about all registered tasks.
func (s *Scheduler) ListTasks() []*TaskInfo {
	tasks := s.registry.List()
	infos := make([]*TaskInfo, 0, len(tasks))

	for _, task := range tasks {
		if info, err := s.GetTaskInfo(task.Name()); err == nil {
			infos = append(infos, info)
		}
	}

	return infos
}

// IsRunning returns true if the specified task is currently running.
func (s *Scheduler) IsRunning(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.running[taskName]
	return exists
}

// GetRunningTasks returns the names of all currently running tasks.
func (s *Scheduler) GetRunningTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.running))
	for name := range s.running {
		names = append(names, name)
	}
	return names
}

// ReloadTaskSchedule reloads the schedule for a task from config.
func (s *Scheduler) ReloadTaskSchedule(taskName string) error {
	task := s.registry.Get(taskName)
	if task == nil {
		return fmt.Errorf("task %q not found", taskName)
	}
	return s.scheduleTask(task)
}

// getTaskSchedule returns the configured schedule for a task.
func (s *Scheduler) getTaskSchedule(taskName string) string {
	if s.configMgr == nil {
		task := s.registry.Get(taskName)
		if task != nil {
			return task.DefaultSchedule()
		}
		return ""
	}

	cfg := s.configMgr.Get()

	switch taskName {
	case "disk_cleanup":
		if cfg.TaskDiskCleanupSchedule != "" {
			return cfg.TaskDiskCleanupSchedule
		}
	case "weekly_report":
		if cfg.TaskWeeklyReportSchedule != "" {
			return cfg.TaskWeeklyReportSchedule
		}
	case "species_cleanup":
		if cfg.TaskSpeciesCleanupSchedule != "" {
			return cfg.TaskSpeciesCleanupSchedule
		}
	case "backup":
		if cfg.TaskBackupSchedule != "" {
			return cfg.TaskBackupSchedule
		}
	}

	// Fall back to task default
	task := s.registry.Get(taskName)
	if task != nil {
		return task.DefaultSchedule()
	}
	return ""
}

// isTaskEnabled returns whether automatic scheduling is enabled for a task.
func (s *Scheduler) isTaskEnabled(taskName string) bool {
	if s.configMgr == nil {
		return true
	}

	cfg := s.configMgr.Get()

	switch taskName {
	case "disk_cleanup":
		return cfg.TaskDiskCleanupEnabled == 1
	case "weekly_report":
		return cfg.TaskWeeklyReportEnabled == 1
	case "species_cleanup":
		return cfg.TaskSpeciesCleanupEnabled == 1
	case "backup":
		return cfg.TaskBackupEnabled == 1
	default:
		return true
	}
}

// broadcastTaskEvent sends a task event to WebSocket clients.
func (s *Scheduler) broadcastTaskEvent(eventType, taskName string, status TaskStatus, errMsg string, trigger TriggerType) {
	if s.hub == nil {
		return
	}

	payload := map[string]interface{}{
		"task_name": taskName,
		"status":    status,
		"trigger":   trigger,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}

	if err := s.hub.Broadcast(ws.ChannelTasks, eventType, payload); err != nil {
		log.Printf("Failed to broadcast task event: %v", err)
	}
}
