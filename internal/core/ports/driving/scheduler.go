package driving

import "context"

// Scheduler manages background tasks like OAuth token refresh and document sync.
type Scheduler interface {
	// Start begins running scheduled tasks.
	// Blocks until context is cancelled or an error occurs.
	Start(ctx context.Context) error

	// Stop gracefully stops all running tasks.
	Stop() error
}
