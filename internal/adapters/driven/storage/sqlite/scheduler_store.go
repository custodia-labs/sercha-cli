package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// schedulerStore implements driven.SchedulerStore.
type schedulerStore struct {
	store *Store
}

var _ driven.SchedulerStore = (*schedulerStore)(nil)

// GetTask retrieves a scheduled task by ID.
// Returns nil and no error if the task does not exist.
func (s *schedulerStore) GetTask(ctx context.Context, taskID string) (*domain.ScheduledTask, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, name, interval_seconds, last_run, next_run, last_error, last_success, enabled
		FROM scheduled_tasks WHERE id = ?
	`, taskID)

	task, err := scanScheduledTask(row)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil // Per interface: return nil and no error if not found
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

// ListTasks returns all scheduled tasks.
func (s *schedulerStore) ListTasks(ctx context.Context) ([]domain.ScheduledTask, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, name, interval_seconds, last_run, next_run, last_error, last_success, enabled
		FROM scheduled_tasks
	`)
	if err != nil {
		return nil, fmt.Errorf("querying scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.ScheduledTask //nolint:prealloc // size unknown from query
	for rows.Next() {
		task, err := scanScheduledTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating scheduled tasks: %w", err)
	}

	return tasks, nil
}

// SaveTask persists a task's state.
// Creates or updates the task based on ID.
func (s *schedulerStore) SaveTask(ctx context.Context, task *domain.ScheduledTask) error {
	if task == nil {
		return domain.ErrInvalidInput
	}

	_, err := s.store.db.ExecContext(ctx, `
		INSERT INTO scheduled_tasks (id, name, interval_seconds, last_run, next_run, last_error, last_success, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			interval_seconds = excluded.interval_seconds,
			last_run = excluded.last_run,
			next_run = excluded.next_run,
			last_error = excluded.last_error,
			last_success = excluded.last_success,
			enabled = excluded.enabled
	`, task.ID, task.Name, int64(task.Interval.Seconds()),
		formatNullableTime(task.LastRun), formatNullableTime(task.NextRun),
		nullString(task.LastError), formatNullableTime(task.LastSuccess),
		boolToInt(task.Enabled))

	if err != nil {
		return fmt.Errorf("saving scheduled task: %w", err)
	}
	return nil
}

// DeleteTask removes a task from storage.
func (s *schedulerStore) DeleteTask(ctx context.Context, taskID string) error {
	_, err := s.store.db.ExecContext(ctx, "DELETE FROM scheduled_tasks WHERE id = ?", taskID)
	if err != nil {
		return fmt.Errorf("deleting scheduled task: %w", err)
	}
	return nil
}

// RecordResult logs a task execution result.
func (s *schedulerStore) RecordResult(ctx context.Context, result *domain.TaskResult) error {
	if result == nil {
		return domain.ErrInvalidInput
	}

	_, err := s.store.db.ExecContext(ctx, `
		INSERT INTO task_results (task_id, started_at, ended_at, success, error, items_processed)
		VALUES (?, ?, ?, ?, ?, ?)
	`, result.TaskID,
		result.StartedAt.Format(time.RFC3339),
		result.EndedAt.Format(time.RFC3339),
		boolToInt(result.Success),
		nullString(result.Error),
		result.ItemsProcessed)

	if err != nil {
		return fmt.Errorf("recording task result: %w", err)
	}
	return nil
}

// GetTaskHistory returns recent results for a task.
// Results are ordered by start time descending (most recent first).
func (s *schedulerStore) GetTaskHistory(ctx context.Context, taskID string, limit int) ([]domain.TaskResult, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT task_id, started_at, ended_at, success, error, items_processed
		FROM task_results
		WHERE task_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, taskID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying task history: %w", err)
	}
	defer rows.Close()

	var results []domain.TaskResult //nolint:prealloc // size unknown from query
	for rows.Next() {
		result, err := scanTaskResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task history: %w", err)
	}

	return results, nil
}

// PruneHistory removes old task results beyond the retention limit.
// Keeps the most recent 'keep' results per task.
func (s *schedulerStore) PruneHistory(ctx context.Context, keep int) error {
	// Delete all results except the most recent 'keep' per task
	_, err := s.store.db.ExecContext(ctx, `
		DELETE FROM task_results
		WHERE id NOT IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY started_at DESC) as rn
				FROM task_results
			) WHERE rn <= ?
		)
	`, keep)
	if err != nil {
		return fmt.Errorf("pruning task history: %w", err)
	}
	return nil
}

// ==================== Helper Functions ====================

// scanScheduledTask scans a single scheduled task row.
func scanScheduledTask(row *sql.Row) (*domain.ScheduledTask, error) {
	var task domain.ScheduledTask
	var intervalSeconds int64
	var lastRun, nextRun, lastError, lastSuccess sql.NullString
	var enabled int

	if err := row.Scan(&task.ID, &task.Name, &intervalSeconds,
		&lastRun, &nextRun, &lastError, &lastSuccess, &enabled); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning scheduled task: %w", err)
	}

	task.Interval = time.Duration(intervalSeconds) * time.Second
	task.LastRun = parseNullableTime(lastRun)
	task.NextRun = parseNullableTime(nextRun)
	if lastError.Valid {
		task.LastError = lastError.String
	}
	task.LastSuccess = parseNullableTime(lastSuccess)
	task.Enabled = enabled == 1

	return &task, nil
}

// scanScheduledTaskRows scans a scheduled task from *sql.Rows.
func scanScheduledTaskRows(rows *sql.Rows) (*domain.ScheduledTask, error) {
	var task domain.ScheduledTask
	var intervalSeconds int64
	var lastRun, nextRun, lastError, lastSuccess sql.NullString
	var enabled int

	if err := rows.Scan(&task.ID, &task.Name, &intervalSeconds,
		&lastRun, &nextRun, &lastError, &lastSuccess, &enabled); err != nil {
		return nil, fmt.Errorf("scanning scheduled task: %w", err)
	}

	task.Interval = time.Duration(intervalSeconds) * time.Second
	task.LastRun = parseNullableTime(lastRun)
	task.NextRun = parseNullableTime(nextRun)
	if lastError.Valid {
		task.LastError = lastError.String
	}
	task.LastSuccess = parseNullableTime(lastSuccess)
	task.Enabled = enabled == 1

	return &task, nil
}

// scanTaskResult scans a task result from *sql.Rows.
func scanTaskResult(rows *sql.Rows) (*domain.TaskResult, error) {
	var result domain.TaskResult
	var startedAt, endedAt string
	var success int
	var errMsg sql.NullString

	if err := rows.Scan(&result.TaskID, &startedAt, &endedAt,
		&success, &errMsg, &result.ItemsProcessed); err != nil {
		return nil, fmt.Errorf("scanning task result: %w", err)
	}

	if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
		result.StartedAt = t
	}
	if t, err := time.Parse(time.RFC3339, endedAt); err == nil {
		result.EndedAt = t
	}
	result.Success = success == 1
	if errMsg.Valid {
		result.Error = errMsg.String
	}

	return &result, nil
}

// formatNullableTime formats a time to RFC3339 string, or returns nil for zero time.
func formatNullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

// parseNullableTime parses a nullable RFC3339 string to time.Time.
// Returns zero time if the string is empty or invalid.
func parseNullableTime(s sql.NullString) time.Time {
	if !s.Valid || s.String == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return time.Time{} // Return zero time on parse error
	}
	return t
}

// nullString returns nil for empty strings, otherwise the string.
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// boolToInt converts a bool to 1 (true) or 0 (false).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
