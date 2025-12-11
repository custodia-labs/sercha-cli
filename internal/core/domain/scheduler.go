package domain

import "time"

// ScheduledTask represents a recurring background task.
type ScheduledTask struct {
	// ID is the unique identifier for the task.
	ID string

	// Name is a human-readable name for the task.
	Name string

	// Interval defines how often the task should run.
	Interval time.Duration

	// LastRun is when the task last ran.
	LastRun time.Time

	// NextRun is when the task should run next.
	NextRun time.Time

	// LastError contains the last error message, if any.
	LastError string

	// LastSuccess is when the task last completed successfully.
	LastSuccess time.Time

	// Enabled indicates whether the task is active.
	Enabled bool
}

// TaskResult represents the outcome of a task execution.
type TaskResult struct {
	// TaskID identifies which task was run.
	TaskID string

	// StartedAt is when the task started.
	StartedAt time.Time

	// EndedAt is when the task completed.
	EndedAt time.Time

	// Success indicates whether the task completed without error.
	Success bool

	// Error contains the error message if Success is false.
	Error string

	// ItemsProcessed is a count of items handled (e.g., documents synced).
	ItemsProcessed int
}

// SchedulerConfig holds scheduler configuration.
type SchedulerConfig struct {
	// Enabled is the master switch for the scheduler.
	Enabled bool

	// TaskConfigs holds per-task configuration.
	TaskConfigs map[string]TaskConfig
}

// TaskConfig holds configuration for a single task.
type TaskConfig struct {
	// Enabled indicates whether this task should run.
	Enabled bool

	// Interval defines how often the task should run.
	Interval time.Duration
}

// GetTaskConfig returns the configuration for a specific task.
// Returns a zero TaskConfig if the task is not configured.
func (c *SchedulerConfig) GetTaskConfig(taskID string) TaskConfig {
	if c.TaskConfigs == nil {
		return TaskConfig{}
	}
	return c.TaskConfigs[taskID]
}

// DefaultSchedulerConfig returns sensible defaults for the scheduler.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Enabled: true,
		TaskConfigs: map[string]TaskConfig{
			"oauth-refresh": {
				Enabled:  true,
				Interval: 45 * time.Minute,
			},
			"document-sync": {
				Enabled:  true,
				Interval: 1 * time.Hour,
			},
		},
	}
}

// Task IDs for built-in tasks.
const (
	TaskIDOAuthRefresh = "oauth-refresh"
	TaskIDDocumentSync = "document-sync"
)
