package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SchedulerStore persists scheduler state for crash recovery.
// It stores task state and execution history.
type SchedulerStore interface {
	// GetTask retrieves a scheduled task by ID.
	// Returns nil and no error if the task does not exist.
	GetTask(ctx context.Context, taskID string) (*domain.ScheduledTask, error)

	// ListTasks returns all scheduled tasks.
	ListTasks(ctx context.Context) ([]domain.ScheduledTask, error)

	// SaveTask persists a task's state.
	// Creates or updates the task based on ID.
	SaveTask(ctx context.Context, task *domain.ScheduledTask) error

	// DeleteTask removes a task from storage.
	DeleteTask(ctx context.Context, taskID string) error

	// RecordResult logs a task execution result.
	RecordResult(ctx context.Context, result *domain.TaskResult) error

	// GetTaskHistory returns recent results for a task.
	// Results are ordered by start time descending (most recent first).
	GetTaskHistory(ctx context.Context, taskID string, limit int) ([]domain.TaskResult, error)

	// PruneHistory removes old task results beyond the retention limit.
	// Keeps the most recent 'keep' results per task.
	PruneHistory(ctx context.Context, keep int) error
}
