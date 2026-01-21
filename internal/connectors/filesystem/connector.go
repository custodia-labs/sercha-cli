package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// Connector reads documents from the local filesystem.
type Connector struct {
	sourceID string
	rootPath string
	watcher  *fsnotify.Watcher
	mu       sync.Mutex
	closed   bool
}

func New(sourceID, rootPath string) *Connector {
	fail := func(msg string) *Connector {
		fmt.Println("Error:", msg)
		fmt.Println("Please provide a valid directory path and retry.")
		return &Connector{
			sourceID: sourceID,
			rootPath: "",
		}
	}

	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return fail("filesystem connector root path is empty")
	}

	// Expand "~"
	if rootPath == "~" || strings.HasPrefix(rootPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fail("failed to resolve home directory")
		}

		if rootPath == "~" {
			rootPath = home
		} else {
			rootPath = filepath.Join(home, rootPath[2:])
		}
	}

	// Convert to absolute path and clean
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return fail(fmt.Sprintf("invalid root path %q", rootPath))
	}

	return &Connector{
		sourceID: sourceID,
		rootPath: filepath.Clean(absPath),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "filesystem"
}

// SourceID returns the source identifier.
func (c *Connector) SourceID() string {
	return c.sourceID
}

// Capabilities returns the connector's capabilities.
func (c *Connector) Capabilities() driven.ConnectorCapabilities {
	return driven.ConnectorCapabilities{
		// Core sync capabilities
		SupportsIncremental: true,
		SupportsWatch:       true,
		SupportsHierarchy:   true,
		SupportsBinary:      false,

		// Authentication - filesystem is local, no auth needed
		RequiresAuth: false,

		// Validation & health
		SupportsValidation: true,

		// Sync behaviour - filesystem uses timestamp cursors
		SupportsCursorReturn: true,
		SupportsPartialSync:  false, // We don't save incremental progress mid-walk

		// API characteristics - not applicable for filesystem
		SupportsRateLimiting: false,
		SupportsPagination:   false,
	}
}

// Validate checks if the filesystem connector is properly configured.
// For filesystem, this verifies the root path exists and is readable.
func (c *Connector) Validate(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Verify root path exists
	fmt.Print(c.rootPath)
	info, err := os.Stat(c.rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("root path does not exist: %s", c.rootPath)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing root path: %s", c.rootPath)
		}
		return fmt.Errorf("failed to access root path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", c.rootPath)
	}

	return nil
}

// FullSync performs a full synchronisation of all documents.
// It walks the entire directory tree and emits RawDocuments for each file.
//
//nolint:gocognit // Sync function with goroutine and channel coordination
func (c *Connector) FullSync(ctx context.Context) (docs <-chan domain.RawDocument, errs <-chan error) {
	docsChan := make(chan domain.RawDocument)
	errsChan := make(chan error, 1)

	go func() {
		defer close(docsChan)
		defer close(errsChan)

		// Verify root path exists
		info, err := os.Stat(c.rootPath)
		if err != nil {
			if os.IsNotExist(err) {
				errsChan <- fmt.Errorf("root path does not exist: %s", c.rootPath)
			} else {
				errsChan <- fmt.Errorf("failed to stat root path: %w", err)
			}
			return
		}
		if !info.IsDir() {
			errsChan <- fmt.Errorf("root path is not a directory: %s", c.rootPath)
			return
		}

		// Walk the directory tree
		err = filepath.WalkDir(c.rootPath, func(path string, d fs.DirEntry, walkErr error) error {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if walkErr != nil {
				// Log error but continue walking
				return nil
			}

			// Skip directories
			if d.IsDir() {
				return nil
			}

			// Skip hidden files and directories
			if isHidden(path) {
				return nil
			}

			// Read file content
			rawDoc, err := c.readFile(path)
			if err != nil {
				// Skip files we can't read
				return nil
			}

			// Send document to channel
			select {
			case <-ctx.Done():
				return ctx.Err()
			case docsChan <- *rawDoc:
			}

			return nil
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			errsChan <- fmt.Errorf("walk error: %w", err)
		}
	}()

	return docsChan, errsChan
}

// readFile reads a file and creates a RawDocument.
func (c *Connector) readFile(path string) (*domain.RawDocument, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Determine parent URI (directory containing the file)
	parentPath := filepath.Dir(path)
	var parentURI *string
	if parentPath != c.rootPath {
		parentURI = &parentPath
	}

	return &domain.RawDocument{
		SourceID:  c.sourceID,
		URI:       path,
		MIMEType:  detectMIMEType(path),
		Content:   content,
		ParentURI: parentURI,
		Metadata: map[string]any{
			"filename":      filepath.Base(path),
			"extension":     strings.TrimPrefix(filepath.Ext(path), "."),
			"size":          info.Size(),
			"modified":      info.ModTime().Format(time.RFC3339),
			"modified_unix": info.ModTime().Unix(),
		},
	}, nil
}

// detectMIMEType returns the MIME type for a file based on its extension.
// Code and text file extensions are checked first because system MIME databases
// often map these to incorrect types (e.g., .ts to video/mp2t, .rs to RLS services).
//
//nolint:gocyclo // Switch statement with many cases for file extensions
func detectMIMEType(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "text/plain"
	}

	// Check code and text file types first - system MIME databases often have incorrect
	// mappings for these (e.g., .ts is video/mp2t, .rs is application/rls-services+xml)
	switch strings.ToLower(ext) {
	case ".md", ".markdown":
		return "text/markdown"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".rs":
		return "text/x-rust"
	case ".ts":
		return "text/typescript"
	case ".tsx":
		return "text/typescript-jsx"
	case ".jsx":
		return "text/javascript-jsx"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".toml":
		return "text/toml"
	case ".sh", ".bash":
		return "text/x-shellscript"
	case ".sql":
		return "text/x-sql"
	case ".xml":
		return "application/xml" // Normalised: Linux returns text/xml, macOS returns application/xml
	}

	// Use Go's mime package for standard types (images, documents, etc.)
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		// Strip charset and other parameters
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
		return mimeType
	}

	return "application/octet-stream"
}

// isHidden returns true if the path contains hidden files/directories (starting with .)
func isHidden(path string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return false
}

// IncrementalSync syncs changes since the last sync state.
// The cursor is a Unix timestamp in nanoseconds representing the last sync time.
// Only files modified after this time are included.
//
//nolint:gocognit,gocyclo // Sync function with goroutine and channel coordination
func (c *Connector) IncrementalSync(
	ctx context.Context, state domain.SyncState,
) (changes <-chan domain.RawDocumentChange, errs <-chan error) {
	changesChan := make(chan domain.RawDocumentChange)
	errsChan := make(chan error, 1)

	go func() {
		defer close(changesChan)
		defer close(errsChan)

		// Parse cursor as Unix timestamp (nanoseconds)
		var sinceTime time.Time
		if state.Cursor != "" {
			nanos, err := strconv.ParseInt(state.Cursor, 10, 64)
			if err != nil {
				errsChan <- fmt.Errorf("invalid cursor format: %w", err)
				return
			}
			sinceTime = time.Unix(0, nanos)
		} else {
			// No cursor means we treat it like a full sync
			sinceTime = time.Time{}
		}

		// Verify root path exists
		info, err := os.Stat(c.rootPath)
		if err != nil {
			if os.IsNotExist(err) {
				errsChan <- fmt.Errorf("root path does not exist: %s", c.rootPath)
			} else {
				errsChan <- fmt.Errorf("failed to stat root path: %w", err)
			}
			return
		}
		if !info.IsDir() {
			errsChan <- fmt.Errorf("root path is not a directory: %s", c.rootPath)
			return
		}

		// Track files we've seen (for detecting deletions)
		currentFiles := make(map[string]struct{})

		// Walk the directory tree
		err = filepath.WalkDir(c.rootPath, func(path string, d fs.DirEntry, walkErr error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if walkErr != nil {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if isHidden(path) {
				return nil
			}

			// Track this file
			currentFiles[path] = struct{}{}

			// Get file info
			fileInfo, err := d.Info()
			if err != nil {
				return nil
			}

			// Skip files not modified since last sync
			if !sinceTime.IsZero() && fileInfo.ModTime().Before(sinceTime) {
				return nil
			}

			// Read file content
			rawDoc, err := c.readFile(path)
			if err != nil {
				return nil
			}

			// Determine change type
			changeType := domain.ChangeUpdated
			if sinceTime.IsZero() || fileInfo.ModTime().After(sinceTime.Add(-time.Second)) {
				// Could be new or updated - we can't tell for certain without tracking state
				changeType = domain.ChangeUpdated
			}

			// Send change to channel
			select {
			case <-ctx.Done():
				return ctx.Err()
			case changesChan <- domain.RawDocumentChange{
				Type:     changeType,
				Document: *rawDoc,
			}:
			}

			return nil
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			errsChan <- fmt.Errorf("walk error: %w", err)
			return
		}

		// Send SyncComplete with new cursor (current time in nanoseconds)
		errsChan <- &driven.SyncComplete{
			NewCursor: strconv.FormatInt(time.Now().UnixNano(), 10),
		}
	}()

	return changesChan, errsChan
}

// Watch monitors for real-time document changes using fsnotify.
// Returns a channel that receives changes as they occur.
//
//nolint:gocognit,gocyclo // Watch function with goroutine and event coordination
func (c *Connector) Watch(ctx context.Context) (<-chan domain.RawDocumentChange, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, fmt.Errorf("connector is closed")
	}

	// Verify root path exists
	info, err := os.Stat(c.rootPath)
	if err != nil {
		return nil, fmt.Errorf("root path error: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", c.rootPath)
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	c.watcher = watcher

	// Add all directories recursively
	err = filepath.WalkDir(c.rootPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if isHidden(path) {
				return filepath.SkipDir
			}
			if err := watcher.Add(path); err != nil {
				return nil // Continue even if we can't watch a directory
			}
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to add directories to watcher: %w", err)
	}

	changesChan := make(chan domain.RawDocumentChange)

	go func() {
		defer close(changesChan)
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Handle the event
				change := c.handleFsEvent(event)
				if change != nil {
					select {
					case <-ctx.Done():
						return
					case changesChan <- *change:
					}
				}

				// If a new directory was created, add it to the watcher
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() && !isHidden(event.Name) {
						_ = watcher.Add(event.Name) //nolint:errcheck // best-effort directory watching
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// Log error but continue watching
				_ = err
			}
		}
	}()

	return changesChan, nil
}

// handleFsEvent processes a filesystem event and returns a RawDocumentChange if applicable.
func (c *Connector) handleFsEvent(event fsnotify.Event) *domain.RawDocumentChange {
	path := event.Name

	// Skip hidden files
	if isHidden(path) {
		return nil
	}

	// Skip directories
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil
	}

	switch {
	case event.Op&fsnotify.Remove != 0 || event.Op&fsnotify.Rename != 0:
		// File was deleted or renamed (treat rename as delete + create)
		return &domain.RawDocumentChange{
			Type: domain.ChangeDeleted,
			Document: domain.RawDocument{
				SourceID: c.sourceID,
				URI:      path,
			},
		}

	case event.Op&fsnotify.Create != 0:
		// New file created
		rawDoc, err := c.readFile(path)
		if err != nil {
			return nil
		}
		return &domain.RawDocumentChange{
			Type:     domain.ChangeCreated,
			Document: *rawDoc,
		}

	case event.Op&fsnotify.Write != 0:
		// File was modified
		rawDoc, err := c.readFile(path)
		if err != nil {
			return nil
		}
		return &domain.RawDocumentChange{
			Type:     domain.ChangeUpdated,
			Document: *rawDoc,
		}
	}

	return nil
}

// GetAccountIdentifier returns an empty string for filesystem connector.
// Filesystem is a local, no-auth connector so there is no account to identify.
func (c *Connector) GetAccountIdentifier(_ context.Context, _ string) (string, error) {
	return "", nil
}

// Close releases resources.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.watcher != nil {
		err := c.watcher.Close()
		c.watcher = nil
		return err
	}
	return nil
}
