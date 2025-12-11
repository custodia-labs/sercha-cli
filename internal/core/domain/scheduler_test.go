package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	assert.True(t, config.Enabled)
	assert.NotNil(t, config.TaskConfigs)
	assert.Len(t, config.TaskConfigs, 2)

	// OAuth refresh config
	oauthCfg := config.TaskConfigs[TaskIDOAuthRefresh]
	assert.True(t, oauthCfg.Enabled)
	assert.Equal(t, 45*time.Minute, oauthCfg.Interval)

	// Document sync config
	docCfg := config.TaskConfigs[TaskIDDocumentSync]
	assert.True(t, docCfg.Enabled)
	assert.Equal(t, 1*time.Hour, docCfg.Interval)
}

func TestSchedulerConfig_GetTaskConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	// Existing task
	oauthCfg := config.GetTaskConfig(TaskIDOAuthRefresh)
	assert.True(t, oauthCfg.Enabled)
	assert.Equal(t, 45*time.Minute, oauthCfg.Interval)

	// Non-existent task
	unknownCfg := config.GetTaskConfig("unknown-task")
	assert.False(t, unknownCfg.Enabled)
	assert.Equal(t, time.Duration(0), unknownCfg.Interval)
}

func TestSchedulerConfig_GetTaskConfig_NilMap(t *testing.T) {
	config := SchedulerConfig{
		Enabled:     true,
		TaskConfigs: nil,
	}

	cfg := config.GetTaskConfig("any-task")
	assert.False(t, cfg.Enabled)
	assert.Equal(t, time.Duration(0), cfg.Interval)
}

func TestTaskConstants(t *testing.T) {
	assert.Equal(t, "oauth-refresh", TaskIDOAuthRefresh)
	assert.Equal(t, "document-sync", TaskIDDocumentSync)
}

func TestScheduledTask_Fields(t *testing.T) {
	now := time.Now()
	task := ScheduledTask{
		ID:          "test-task",
		Name:        "Test Task",
		Interval:    1 * time.Hour,
		LastRun:     now.Add(-30 * time.Minute),
		NextRun:     now.Add(30 * time.Minute),
		LastError:   "previous error",
		LastSuccess: now.Add(-45 * time.Minute),
		Enabled:     true,
	}

	assert.Equal(t, "test-task", task.ID)
	assert.Equal(t, "Test Task", task.Name)
	assert.Equal(t, 1*time.Hour, task.Interval)
	assert.Equal(t, "previous error", task.LastError)
	assert.True(t, task.Enabled)
}

func TestTaskResult_Fields(t *testing.T) {
	now := time.Now()
	result := TaskResult{
		TaskID:         "test-task",
		StartedAt:      now.Add(-5 * time.Minute),
		EndedAt:        now,
		Success:        true,
		Error:          "",
		ItemsProcessed: 42,
	}

	assert.Equal(t, "test-task", result.TaskID)
	assert.Equal(t, 42, result.ItemsProcessed)
	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
}

func TestTaskResult_Failed(t *testing.T) {
	now := time.Now()
	result := TaskResult{
		TaskID:         "test-task",
		StartedAt:      now.Add(-5 * time.Minute),
		EndedAt:        now,
		Success:        false,
		Error:          "connection timeout",
		ItemsProcessed: 0,
	}

	assert.False(t, result.Success)
	assert.Equal(t, "connection timeout", result.Error)
}

func TestTaskConfig_Fields(t *testing.T) {
	cfg := TaskConfig{
		Enabled:  true,
		Interval: 30 * time.Minute,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 30*time.Minute, cfg.Interval)
}
