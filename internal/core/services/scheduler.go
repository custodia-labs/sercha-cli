package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Scheduler manages background task execution.
// It is a pure core service with no external control API.
type Scheduler struct {
	config   domain.SchedulerConfig
	store    driven.SchedulerStore
	syncOrch driving.SyncOrchestrator

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler creates a scheduler with configuration.
func NewScheduler(
	config domain.SchedulerConfig,
	store driven.SchedulerStore,
	syncOrch driving.SyncOrchestrator,
) *Scheduler {
	return &Scheduler{
		config:   config,
		store:    store,
		syncOrch: syncOrch,
	}
}

// Start begins the scheduler loop. This method blocks until Stop is called.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil // Already running
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// Initialise tasks in store
	if err := s.initialiseTasks(ctx); err != nil {
		log.Printf("scheduler: failed to initialise tasks: %v", err)
	}

	// Run the main scheduler loop
	return s.run(ctx)
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	// Wait for running tasks to complete
	s.wg.Wait()

	return nil
}

// initialiseTasks ensures all configured tasks exist in the store.
func (s *Scheduler) initialiseTasks(ctx context.Context) error {
	// Document sync task
	if taskCfg := s.config.GetTaskConfig(domain.TaskIDDocumentSync); taskCfg.Enabled {
		if err := s.ensureTask(ctx, domain.TaskIDDocumentSync, "Document Sync", taskCfg); err != nil {
			return err
		}
	}

	return nil
}

// ensureTask creates or updates a task in the store.
func (s *Scheduler) ensureTask(ctx context.Context, id, name string, cfg domain.TaskConfig) error {
	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		return err
	}

	if task == nil {
		// Create new task
		task = &domain.ScheduledTask{
			ID:       id,
			Name:     name,
			Interval: cfg.Interval,
			Enabled:  cfg.Enabled,
			NextRun:  time.Now().Add(cfg.Interval),
		}
	} else {
		// Update interval if changed
		if task.Interval != cfg.Interval {
			task.Interval = cfg.Interval
			// Recalculate next run from now
			task.NextRun = time.Now().Add(cfg.Interval)
		}
		task.Enabled = cfg.Enabled
	}

	return s.store.SaveTask(ctx, task)
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) error {
	// Check for due tasks immediately on startup
	s.checkAndRunDueTasks(ctx)

	// Use a 1-minute ticker to check for due tasks
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			s.checkAndRunDueTasks(ctx)
		}
	}
}

// checkAndRunDueTasks finds and executes tasks that are due.
func (s *Scheduler) checkAndRunDueTasks(ctx context.Context) {
	tasks, err := s.store.ListTasks(ctx)
	if err != nil {
		log.Printf("scheduler: failed to list tasks: %v", err)
		return
	}

	now := time.Now()
	for i := range tasks {
		task := &tasks[i]
		if !task.Enabled {
			continue
		}
		if task.NextRun.IsZero() || task.NextRun.Before(now) || task.NextRun.Equal(now) {
			s.runTask(ctx, task)
		}
	}
}

// runTask executes a single task.
func (s *Scheduler) runTask(ctx context.Context, task *domain.ScheduledTask) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		result := &domain.TaskResult{
			TaskID:    task.ID,
			StartedAt: time.Now(),
		}

		var err error
		switch task.ID {
		case domain.TaskIDDocumentSync:
			result.ItemsProcessed, err = s.runDocumentSync(ctx)
		default:
			log.Printf("scheduler: unknown task ID: %s", task.ID)
			return
		}

		result.EndedAt = time.Now()
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			task.LastError = err.Error()
		} else {
			result.Success = true
			task.LastError = ""
			task.LastSuccess = result.EndedAt
		}

		// Update task state
		task.LastRun = result.StartedAt
		task.NextRun = result.EndedAt.Add(task.Interval)

		if saveErr := s.store.SaveTask(ctx, task); saveErr != nil {
			log.Printf("scheduler: failed to save task %s: %v", task.ID, saveErr)
		}

		// Record result for history
		if recordErr := s.store.RecordResult(ctx, result); recordErr != nil {
			log.Printf("scheduler: failed to record result for %s: %v", task.ID, recordErr)
		}

		// Prune old history (keep last 100 results per task)
		if pruneErr := s.store.PruneHistory(ctx, 100); pruneErr != nil {
			log.Printf("scheduler: failed to prune history: %v", pruneErr)
		}
	}()
}

// runDocumentSync syncs all sources.
//
//nolint:unparam // itemsProcessed always 0 until SyncAll returns count
func (s *Scheduler) runDocumentSync(ctx context.Context) (int, error) {
	if s.syncOrch == nil {
		return 0, nil
	}

	// SyncAll syncs all configured sources
	// We don't have a direct way to count documents synced here,
	// so we return 0 for items processed
	err := s.syncOrch.SyncAll(ctx)
	return 0, err
}
