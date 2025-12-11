package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// --- Mock implementations for scheduler testing ---

// mockSchedulerStore implements driven.SchedulerStore for testing.
type mockSchedulerStore struct {
	mu       sync.RWMutex
	tasks    map[string]*domain.ScheduledTask
	results  map[string][]domain.TaskResult
	saveErr  error
	listErr  error
	getErr   error
	pruneErr error
}

func newMockSchedulerStore() *mockSchedulerStore {
	return &mockSchedulerStore{
		tasks:   make(map[string]*domain.ScheduledTask),
		results: make(map[string][]domain.TaskResult),
	}
}

func (m *mockSchedulerStore) GetTask(_ context.Context, taskID string) (*domain.ScheduledTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, nil
	}
	// Return a copy
	taskCopy := *task
	return &taskCopy, nil
}

func (m *mockSchedulerStore) ListTasks(_ context.Context) ([]domain.ScheduledTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	tasks := make([]domain.ScheduledTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (m *mockSchedulerStore) SaveTask(_ context.Context, task *domain.ScheduledTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	if task == nil {
		return domain.ErrInvalidInput
	}
	taskCopy := *task
	m.tasks[task.ID] = &taskCopy
	return nil
}

func (m *mockSchedulerStore) DeleteTask(_ context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, taskID)
	return nil
}

func (m *mockSchedulerStore) RecordResult(_ context.Context, result *domain.TaskResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if result == nil {
		return domain.ErrInvalidInput
	}
	m.results[result.TaskID] = append(m.results[result.TaskID], *result)
	return nil
}

func (m *mockSchedulerStore) GetTaskHistory(_ context.Context, taskID string, limit int) ([]domain.TaskResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	results := m.results[taskID]
	if len(results) > limit {
		results = results[len(results)-limit:]
	}
	return results, nil
}

func (m *mockSchedulerStore) PruneHistory(_ context.Context, _ int) error {
	return m.pruneErr
}

// mockSyncOrchestrator implements driving.SyncOrchestrator for testing.
type mockSyncOrchestrator struct {
	syncAllCalled bool
	syncAllErr    error
}

func (m *mockSyncOrchestrator) Sync(_ context.Context, _ string) error {
	return nil
}

func (m *mockSyncOrchestrator) SyncAll(_ context.Context) error {
	m.syncAllCalled = true
	return m.syncAllErr
}

func (m *mockSyncOrchestrator) Status(_ context.Context, _ string) (*driving.SyncStatus, error) {
	return &driving.SyncStatus{}, nil
}

// Ensure mocks implement interfaces
var _ driven.SchedulerStore = (*mockSchedulerStore)(nil)
var _ driving.SyncOrchestrator = (*mockSyncOrchestrator)(nil)

// ==================== Scheduler Tests ====================

func TestNewScheduler(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)

	require.NotNil(t, scheduler)
	assert.Equal(t, config.Enabled, scheduler.config.Enabled)
}

func TestScheduler_StartStop(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)

	ctx, cancel := context.WithCancel(context.Background())

	// Start scheduler in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = scheduler.Start(ctx)
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop scheduler
	cancel()
	err := scheduler.Stop()
	require.NoError(t, err)

	wg.Wait()
}

func TestScheduler_StopWithoutStart(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)

	// Stop without starting should be safe
	err := scheduler.Stop()
	require.NoError(t, err)
}

func TestScheduler_DoubleStart(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First start
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = scheduler.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Second start should return immediately (already running)
	ctx2 := context.Background()
	err := scheduler.Start(ctx2)
	assert.NoError(t, err) // Should not error

	cancel()
	scheduler.Stop() //nolint:errcheck
	wg.Wait()
}

func TestScheduler_InitialiseTasks(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)

	ctx := context.Background()
	err := scheduler.initialiseTasks(ctx)
	require.NoError(t, err)

	// Check document sync task was created
	docTask, err := store.GetTask(ctx, domain.TaskIDDocumentSync)
	require.NoError(t, err)
	require.NotNil(t, docTask)
	assert.Equal(t, "Document Sync", docTask.Name)
	assert.True(t, docTask.Enabled)
}

func TestScheduler_EnsureTask_UpdateInterval(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)
	ctx := context.Background()

	// Create initial task
	taskCfg := domain.TaskConfig{
		Enabled:  true,
		Interval: 1 * time.Hour,
	}
	err := scheduler.ensureTask(ctx, "test-task", "Test Task", taskCfg)
	require.NoError(t, err)

	// Update with new interval
	taskCfg.Interval = 2 * time.Hour
	err = scheduler.ensureTask(ctx, "test-task", "Test Task", taskCfg)
	require.NoError(t, err)

	// Verify interval was updated
	task, err := store.GetTask(ctx, "test-task")
	require.NoError(t, err)
	assert.Equal(t, 2*time.Hour, task.Interval)
}

func TestScheduler_RunDocumentSync(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)
	ctx := context.Background()

	_, err := scheduler.runDocumentSync(ctx)
	require.NoError(t, err)
	assert.True(t, syncOrch.syncAllCalled)
}

func TestScheduler_RunDocumentSync_NilOrchestrator(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()

	scheduler := NewScheduler(config, store, nil)
	ctx := context.Background()

	_, err := scheduler.runDocumentSync(ctx)
	require.NoError(t, err)
}

func TestScheduler_CheckAndRunDueTasks(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()
	syncOrch := &mockSyncOrchestrator{}

	scheduler := NewScheduler(config, store, syncOrch)
	ctx := context.Background()

	// Create a task that is due
	now := time.Now()
	dueTask := &domain.ScheduledTask{
		ID:       domain.TaskIDDocumentSync,
		Name:     "Document Sync",
		Interval: 1 * time.Hour,
		NextRun:  now.Add(-1 * time.Minute), // Already past due
		Enabled:  true,
	}
	err := store.SaveTask(ctx, dueTask)
	require.NoError(t, err)

	// Check and run due tasks
	scheduler.checkAndRunDueTasks(ctx)

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	// Verify sync was called
	assert.True(t, syncOrch.syncAllCalled)
}

func TestScheduler_RunTask_UnknownTaskID(t *testing.T) {
	config := domain.DefaultSchedulerConfig()
	store := newMockSchedulerStore()

	scheduler := NewScheduler(config, store, nil)
	ctx := context.Background()

	// Create unknown task
	task := &domain.ScheduledTask{
		ID:      "unknown-task",
		Name:    "Unknown",
		Enabled: true,
	}

	// This should just log and return, not panic
	scheduler.runTask(ctx, task)
	scheduler.wg.Wait()
}
