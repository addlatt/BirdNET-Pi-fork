-- Rollback task history table
DROP INDEX IF EXISTS idx_task_history_started;
DROP INDEX IF EXISTS idx_task_history_name_started;
DROP TABLE IF EXISTS task_history;
