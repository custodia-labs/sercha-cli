package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ==================== SchedulerStore Tests ====================

func TestSchedulerStore_SaveAndGetTask(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create a test task
	now := time.Now().UTC().Truncate(time.Second)
	task := &domain.ScheduledTask{
		ID:          "oauth-refresh",
		Name:        "OAuth Token Refresh",
		Interval:    45 * time.Minute,
		LastRun:     now.Add(-30 * time.Minute),
		NextRun:     now.Add(15 * time.Minute),
		LastError:   "",
		LastSuccess: now.Add(-30 * time.Minute),
		Enabled:     true,
	}

	// Save task
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Get task
	retrieved, err := schedulerStore.GetTask(ctx, "oauth-refresh")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.Name, retrieved.Name)
	assert.Equal(t, task.Interval, retrieved.Interval)
	assert.Equal(t, task.Enabled, retrieved.Enabled)
	assert.WithinDuration(t, task.LastRun, retrieved.LastRun, time.Second)
	assert.WithinDuration(t, task.NextRun, retrieved.NextRun, time.Second)
	assert.WithinDuration(t, task.LastSuccess, retrieved.LastSuccess, time.Second)
}

func TestSchedulerStore_GetTask_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Get non-existent task should return nil, nil
	task, err := schedulerStore.GetTask(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, task)
}

func TestSchedulerStore_SaveTask_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create initial task
	task := &domain.ScheduledTask{
		ID:       "test-task",
		Name:     "Test Task",
		Interval: 1 * time.Hour,
		Enabled:  true,
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Update task
	task.Name = "Updated Task"
	task.Interval = 2 * time.Hour
	task.LastError = "some error"
	task.Enabled = false
	err = schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Verify update
	retrieved, err := schedulerStore.GetTask(ctx, "test-task")
	require.NoError(t, err)
	assert.Equal(t, "Updated Task", retrieved.Name)
	assert.Equal(t, 2*time.Hour, retrieved.Interval)
	assert.Equal(t, "some error", retrieved.LastError)
	assert.False(t, retrieved.Enabled)
}

func TestSchedulerStore_SaveTask_NilTask(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	err := schedulerStore.SaveTask(ctx, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestSchedulerStore_ListTasks(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create multiple tasks
	tasks := []*domain.ScheduledTask{
		{ID: "task-1", Name: "Task 1", Interval: 1 * time.Hour, Enabled: true},
		{ID: "task-2", Name: "Task 2", Interval: 2 * time.Hour, Enabled: false},
		{ID: "task-3", Name: "Task 3", Interval: 30 * time.Minute, Enabled: true},
	}

	for _, task := range tasks {
		err := schedulerStore.SaveTask(ctx, task)
		require.NoError(t, err)
	}

	// List all tasks
	retrieved, err := schedulerStore.ListTasks(ctx)
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)
}

func TestSchedulerStore_ListTasks_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// List with no tasks
	tasks, err := schedulerStore.ListTasks(ctx)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestSchedulerStore_DeleteTask(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create task
	task := &domain.ScheduledTask{
		ID:       "to-delete",
		Name:     "Delete Me",
		Interval: 1 * time.Hour,
		Enabled:  true,
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Delete task
	err = schedulerStore.DeleteTask(ctx, "to-delete")
	require.NoError(t, err)

	// Verify deletion
	retrieved, err := schedulerStore.GetTask(ctx, "to-delete")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSchedulerStore_RecordResult(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create task first
	task := &domain.ScheduledTask{
		ID:       "result-task",
		Name:     "Task with Results",
		Interval: 1 * time.Hour,
		Enabled:  true,
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Record a successful result
	now := time.Now().UTC().Truncate(time.Second)
	result := &domain.TaskResult{
		TaskID:         "result-task",
		StartedAt:      now.Add(-5 * time.Minute),
		EndedAt:        now,
		Success:        true,
		Error:          "",
		ItemsProcessed: 10,
	}
	err = schedulerStore.RecordResult(ctx, result)
	require.NoError(t, err)

	// Record a failed result
	failResult := &domain.TaskResult{
		TaskID:         "result-task",
		StartedAt:      now,
		EndedAt:        now.Add(1 * time.Minute),
		Success:        false,
		Error:          "connection timeout",
		ItemsProcessed: 0,
	}
	err = schedulerStore.RecordResult(ctx, failResult)
	require.NoError(t, err)

	// Get history
	history, err := schedulerStore.GetTaskHistory(ctx, "result-task", 10)
	require.NoError(t, err)
	assert.Len(t, history, 2)

	// Most recent first
	assert.False(t, history[0].Success)
	assert.Equal(t, "connection timeout", history[0].Error)
	assert.True(t, history[1].Success)
	assert.Equal(t, 10, history[1].ItemsProcessed)
}

func TestSchedulerStore_RecordResult_NilResult(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	err := schedulerStore.RecordResult(ctx, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestSchedulerStore_GetTaskHistory_Limit(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create task
	task := &domain.ScheduledTask{
		ID:       "history-task",
		Name:     "Task with History",
		Interval: 1 * time.Hour,
		Enabled:  true,
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Record 5 results
	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		result := &domain.TaskResult{
			TaskID:         "history-task",
			StartedAt:      now.Add(time.Duration(i) * time.Minute),
			EndedAt:        now.Add(time.Duration(i)*time.Minute + 30*time.Second),
			Success:        true,
			ItemsProcessed: i + 1,
		}
		err := schedulerStore.RecordResult(ctx, result)
		require.NoError(t, err)
	}

	// Get limited history
	history, err := schedulerStore.GetTaskHistory(ctx, "history-task", 3)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestSchedulerStore_GetTaskHistory_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create task without results
	task := &domain.ScheduledTask{
		ID:       "no-history-task",
		Name:     "Task without History",
		Interval: 1 * time.Hour,
		Enabled:  true,
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Get empty history
	history, err := schedulerStore.GetTaskHistory(ctx, "no-history-task", 10)
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestSchedulerStore_PruneHistory(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create task
	task := &domain.ScheduledTask{
		ID:       "prune-task",
		Name:     "Task for Pruning",
		Interval: 1 * time.Hour,
		Enabled:  true,
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Record 10 results
	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 10; i++ {
		result := &domain.TaskResult{
			TaskID:         "prune-task",
			StartedAt:      now.Add(time.Duration(i) * time.Minute),
			EndedAt:        now.Add(time.Duration(i)*time.Minute + 30*time.Second),
			Success:        true,
			ItemsProcessed: i + 1,
		}
		err := schedulerStore.RecordResult(ctx, result)
		require.NoError(t, err)
	}

	// Prune to keep only 3
	err = schedulerStore.PruneHistory(ctx, 3)
	require.NoError(t, err)

	// Verify only 3 remain
	history, err := schedulerStore.GetTaskHistory(ctx, "prune-task", 100)
	require.NoError(t, err)
	assert.Len(t, history, 3)

	// Most recent should be kept
	assert.Equal(t, 10, history[0].ItemsProcessed)
	assert.Equal(t, 9, history[1].ItemsProcessed)
	assert.Equal(t, 8, history[2].ItemsProcessed)
}

func TestSchedulerStore_TaskWithZeroTimes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	schedulerStore := store.SchedulerStore()

	// Create task with zero times
	task := &domain.ScheduledTask{
		ID:       "zero-times-task",
		Name:     "New Task",
		Interval: 1 * time.Hour,
		Enabled:  true,
		// LastRun, NextRun, LastSuccess all zero
	}
	err := schedulerStore.SaveTask(ctx, task)
	require.NoError(t, err)

	// Get and verify zero times are preserved
	retrieved, err := schedulerStore.GetTask(ctx, "zero-times-task")
	require.NoError(t, err)
	assert.True(t, retrieved.LastRun.IsZero())
	assert.True(t, retrieved.NextRun.IsZero())
	assert.True(t, retrieved.LastSuccess.IsZero())
}

// ==================== Helper Function Tests ====================

func TestFormatNullableTime(t *testing.T) {
	// Zero time should return nil
	result := formatNullableTime(time.Time{})
	assert.Nil(t, result)

	// Non-zero time should return RFC3339 string
	now := time.Now().UTC()
	result = formatNullableTime(now)
	assert.IsType(t, "", result)
	assert.Equal(t, now.Format(time.RFC3339), result)
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

func TestNullString(t *testing.T) {
	// Empty string should return nil
	result := nullString("")
	assert.Nil(t, result)

	// Non-empty string should return the string
	result = nullString("hello")
	assert.Equal(t, "hello", result)
}
