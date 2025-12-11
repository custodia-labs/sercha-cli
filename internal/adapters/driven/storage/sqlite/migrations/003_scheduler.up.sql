-- Migration 003: Scheduler tables
-- Stores scheduler state for background tasks

-- Scheduled tasks table (domain.ScheduledTask)
-- Tracks recurring background task state
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL,
    last_run TEXT,                -- ISO 8601 timestamp
    next_run TEXT,                -- ISO 8601 timestamp
    last_error TEXT,              -- Last error message if any
    last_success TEXT,            -- ISO 8601 timestamp of last success
    enabled INTEGER DEFAULT 1     -- 1 = enabled, 0 = disabled
);

-- Task results table (domain.TaskResult)
-- Logs task execution history for debugging and monitoring
CREATE TABLE IF NOT EXISTS task_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    started_at TEXT NOT NULL,     -- ISO 8601 timestamp
    ended_at TEXT NOT NULL,       -- ISO 8601 timestamp
    success INTEGER NOT NULL,     -- 1 = success, 0 = failure
    error TEXT,                   -- Error message if success = 0
    items_processed INTEGER DEFAULT 0,
    FOREIGN KEY (task_id) REFERENCES scheduled_tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_task_results_task_id ON task_results(task_id);
CREATE INDEX IF NOT EXISTS idx_task_results_started_at ON task_results(started_at);

-- Record this migration
INSERT INTO schema_migrations (version) VALUES (3);
