-- Task execution history table for the Go scheduler
-- Tracks all task executions (scheduled and manual) for monitoring and debugging

CREATE TABLE IF NOT EXISTS task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_name TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    completed_at DATETIME NOT NULL,
    duration_ms INTEGER NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('completed', 'failed', 'cancelled')),
    error TEXT,
    trigger TEXT NOT NULL CHECK(trigger IN ('scheduled', 'manual'))
);

-- Index for efficient history lookups by task name
CREATE INDEX IF NOT EXISTS idx_task_history_name_started ON task_history(task_name, started_at DESC);

-- Index for cleanup of old history entries
CREATE INDEX IF NOT EXISTS idx_task_history_started ON task_history(started_at);
