package scheduler

import (
	"database/sql"
	"fmt"
	"time"
)

// HistoryStore manages task execution history in SQLite.
type HistoryStore struct {
	db *sql.DB
}

// NewHistoryStore creates a new history store with the given database connection.
func NewHistoryStore(db *sql.DB) *HistoryStore {
	return &HistoryStore{db: db}
}

// SaveExecution records a task execution in the database.
func (h *HistoryStore) SaveExecution(exec *TaskExecution) error {
	query := `
		INSERT INTO task_history (task_name, started_at, completed_at, duration_ms, status, error, trigger)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := h.db.Exec(query,
		exec.TaskName,
		exec.StartedAt.Format(time.RFC3339),
		exec.CompletedAt.Format(time.RFC3339),
		exec.DurationMs,
		exec.Status,
		exec.Error,
		exec.Trigger,
	)
	if err != nil {
		return fmt.Errorf("failed to insert task execution: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		exec.ID = id
	}
	return nil
}

// GetHistory returns the execution history for a task.
func (h *HistoryStore) GetHistory(taskName string, limit, offset int) ([]*TaskExecution, error) {
	query := `
		SELECT id, task_name, started_at, completed_at, duration_ms, status, error, trigger
		FROM task_history
		WHERE task_name = ?
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := h.db.Query(query, taskName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query task history: %w", err)
	}
	defer rows.Close()

	return h.scanExecutions(rows)
}

// GetLatestExecution returns the most recent execution for a task.
func (h *HistoryStore) GetLatestExecution(taskName string) (*TaskExecution, error) {
	query := `
		SELECT id, task_name, started_at, completed_at, duration_ms, status, error, trigger
		FROM task_history
		WHERE task_name = ?
		ORDER BY started_at DESC
		LIMIT 1
	`
	row := h.db.QueryRow(query, taskName)
	return h.scanExecution(row)
}

// GetAllHistory returns the execution history for all tasks.
func (h *HistoryStore) GetAllHistory(limit, offset int) ([]*TaskExecution, error) {
	query := `
		SELECT id, task_name, started_at, completed_at, duration_ms, status, error, trigger
		FROM task_history
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := h.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query task history: %w", err)
	}
	defer rows.Close()

	return h.scanExecutions(rows)
}

// CountHistory returns the total number of executions for a task.
func (h *HistoryStore) CountHistory(taskName string) (int, error) {
	query := `SELECT COUNT(*) FROM task_history WHERE task_name = ?`
	var count int
	if err := h.db.QueryRow(query, taskName).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count task history: %w", err)
	}
	return count, nil
}

// DeleteOldHistory removes executions older than the given duration.
func (h *HistoryStore) DeleteOldHistory(taskName string, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)
	query := `DELETE FROM task_history WHERE task_name = ? AND started_at < ?`
	result, err := h.db.Exec(query, taskName, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old history: %w", err)
	}
	return result.RowsAffected()
}

// scanExecution scans a single row into a TaskExecution.
func (h *HistoryStore) scanExecution(row *sql.Row) (*TaskExecution, error) {
	var exec TaskExecution
	var startedAt, completedAt string
	var errStr sql.NullString

	err := row.Scan(
		&exec.ID,
		&exec.TaskName,
		&startedAt,
		&completedAt,
		&exec.DurationMs,
		&exec.Status,
		&errStr,
		&exec.Trigger,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan task execution: %w", err)
	}

	exec.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	exec.CompletedAt, _ = time.Parse(time.RFC3339, completedAt)
	if errStr.Valid {
		exec.Error = errStr.String
	}

	return &exec, nil
}

// scanExecutions scans multiple rows into TaskExecution slice.
func (h *HistoryStore) scanExecutions(rows *sql.Rows) ([]*TaskExecution, error) {
	var executions []*TaskExecution

	for rows.Next() {
		var exec TaskExecution
		var startedAt, completedAt string
		var errStr sql.NullString

		err := rows.Scan(
			&exec.ID,
			&exec.TaskName,
			&startedAt,
			&completedAt,
			&exec.DurationMs,
			&exec.Status,
			&errStr,
			&exec.Trigger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task execution: %w", err)
		}

		exec.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		exec.CompletedAt, _ = time.Parse(time.RFC3339, completedAt)
		if errStr.Valid {
			exec.Error = errStr.String
		}

		executions = append(executions, &exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task history: %w", err)
	}

	return executions, nil
}

// EnsureTable creates the task_history table if it doesn't exist.
func (h *HistoryStore) EnsureTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS task_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_name TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			completed_at DATETIME NOT NULL,
			duration_ms INTEGER NOT NULL,
			status TEXT NOT NULL,
			error TEXT,
			trigger TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_task_history_name_started ON task_history(task_name, started_at DESC);
	`
	_, err := h.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create task_history table: %w", err)
	}
	return nil
}
