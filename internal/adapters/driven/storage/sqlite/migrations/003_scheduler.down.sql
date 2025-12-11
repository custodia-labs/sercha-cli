-- Migration 003: Rollback scheduler tables

DROP INDEX IF EXISTS idx_task_results_started_at;
DROP INDEX IF EXISTS idx_task_results_task_id;
DROP TABLE IF EXISTS task_results;
DROP TABLE IF EXISTS scheduled_tasks;

DELETE FROM schema_migrations WHERE version = 3;
