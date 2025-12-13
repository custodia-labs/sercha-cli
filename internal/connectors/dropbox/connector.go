package dropbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/users"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// ErrCursorReset indicates the cursor has expired and a full sync is required.
var ErrCursorReset = errors.New("cursor reset, full sync required")

// Connector fetches files from Dropbox.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	rateLimiter   *RateLimiter
	mu            sync.Mutex
	closed        bool
}

// New creates a new Dropbox connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		rateLimiter:   NewRateLimiter(),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "dropbox"
}

// SourceID returns the source identifier.
func (c *Connector) SourceID() string {
	return c.sourceID
}

// Capabilities returns the connector's capabilities.
func (c *Connector) Capabilities() driven.ConnectorCapabilities {
	return driven.ConnectorCapabilities{
		SupportsIncremental:  true,
		SupportsWatch:        false,
		SupportsHierarchy:    true,
		SupportsBinary:       false,
		RequiresAuth:         true,
		SupportsValidation:   true,
		SupportsCursorReturn: true,
		SupportsPartialSync:  true,
		SupportsRateLimiting: true,
		SupportsPagination:   true,
	}
}

// Validate checks if the Dropbox connector is properly configured.
func (c *Connector) Validate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return domain.ErrConnectorClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	// Validate by getting current account using users client
	usersClient := c.createUsersClient(token)
	_, err = usersClient.GetCurrentAccount()
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	return nil
}

// FullSync fetches all files from Dropbox.
func (c *Connector) FullSync(ctx context.Context) (
	docs <-chan domain.RawDocument, errs <-chan error,
) {
	docsChan := make(chan domain.RawDocument)
	errsChan := make(chan error, 1)

	go func() {
		defer close(docsChan)
		defer close(errsChan)
		errsChan <- c.runFullSync(ctx, docsChan)
	}()

	return docsChan, errsChan
}

// runFullSync executes the full sync logic.
func (c *Connector) runFullSync(ctx context.Context, docsChan chan<- domain.RawDocument) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	client := c.createClient(token)
	cursor := NewCursor()

	// Start listing files
	arg := files.NewListFolderArg(c.config.FolderPath)
	arg.Recursive = c.config.Recursive
	arg.Limit = c.config.MaxResults
	arg.IncludeDeleted = false

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	result, err := client.ListFolder(arg)
	if err != nil {
		return fmt.Errorf("list folder: %w", err)
	}

	// Process first batch
	if err := c.processEntries(ctx, client, result.Entries, docsChan, nil); err != nil {
		return err
	}

	// Continue paginating
	for result.HasMore {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		continueArg := files.NewListFolderContinueArg(result.Cursor)
		result, err = client.ListFolderContinue(continueArg)
		if err != nil {
			return fmt.Errorf("list folder continue: %w", err)
		}

		if err := c.processEntries(ctx, client, result.Entries, docsChan, nil); err != nil {
			return err
		}
	}

	// Store the cursor for incremental sync
	cursor.SetCursor(result.Cursor)
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// IncrementalSync fetches only changes since the last sync using the cursor.
func (c *Connector) IncrementalSync(
	ctx context.Context, state domain.SyncState,
) (changes <-chan domain.RawDocumentChange, errs <-chan error) {
	changesChan := make(chan domain.RawDocumentChange)
	errsChan := make(chan error, 1)

	go func() {
		defer close(changesChan)
		defer close(errsChan)
		errsChan <- c.runIncrementalSync(ctx, state, changesChan)
	}()

	return changesChan, errsChan
}

// runIncrementalSync executes the incremental sync logic.
//
//nolint:gocognit // Sync logic requires handling pagination, rate limiting, and cursor management
func (c *Connector) runIncrementalSync(
	ctx context.Context, state domain.SyncState, changesChan chan<- domain.RawDocumentChange,
) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	cursor, err := DecodeCursor(state.Cursor)
	if err != nil {
		return fmt.Errorf("invalid cursor, full sync required: %w", err)
	}
	if cursor.IsEmpty() {
		return fmt.Errorf("invalid cursor, full sync required: cursor has no value")
	}

	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	client := c.createClient(token)

	// Continue from the stored cursor
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	continueArg := files.NewListFolderContinueArg(cursor.GetCursor())
	result, err := client.ListFolderContinue(continueArg)
	if err != nil {
		// Check for cursor reset error
		if isResetError(err) {
			return fmt.Errorf("%w: %w", ErrCursorReset, err)
		}
		return fmt.Errorf("list folder continue: %w", err)
	}

	// Process entries with change tracking
	if err := c.processEntries(ctx, client, result.Entries, nil, changesChan); err != nil {
		return err
	}

	// Continue paginating
	for result.HasMore {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		continueArg = files.NewListFolderContinueArg(result.Cursor)
		result, err = client.ListFolderContinue(continueArg)
		if err != nil {
			if isResetError(err) {
				return fmt.Errorf("%w: %w", ErrCursorReset, err)
			}
			return fmt.Errorf("list folder continue: %w", err)
		}

		if err := c.processEntries(ctx, client, result.Entries, nil, changesChan); err != nil {
			return err
		}
	}

	// Update cursor
	cursor.SetCursor(result.Cursor)
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// processEntries processes a batch of entries from a list folder response.
func (c *Connector) processEntries(
	ctx context.Context,
	client files.Client,
	entries []files.IsMetadata,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		switch e := entry.(type) {
		case *files.FileMetadata:
			if err := c.processFile(ctx, client, e, docsChan, changesChan); err != nil {
				return err
			}
		case *files.DeletedMetadata:
			if err := c.handleDeletedItem(ctx, e, changesChan); err != nil {
				return err
			}
		case *files.FolderMetadata:
			// Skip folders - we only index files
			continue
		}
	}
	return nil
}

// processFile processes a single file entry.
func (c *Connector) processFile(
	ctx context.Context,
	client files.Client,
	file *files.FileMetadata,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	if !ShouldSyncFile(file, c.config) {
		return nil
	}

	// Download content if appropriate
	var content []byte
	mimeType := getMIMEType(file.Name)
	if shouldDownloadContent(mimeType) && file.Size <= MaxContentSize {
		var err error
		content, err = c.downloadFileContent(ctx, client, file.PathLower)
		if err != nil {
			// Continue without content on error
			content = nil
		}
	}

	doc := FileToRawDocument(file, content, c.sourceID)
	return c.emitDocument(ctx, doc, docsChan, changesChan)
}

// downloadFileContent downloads the content of a file.
func (c *Connector) downloadFileContent(
	ctx context.Context,
	client files.Client,
	path string,
) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	arg := files.NewDownloadArg(path)
	_, reader, err := client.Download(arg)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer reader.Close()

	// Read with size limit
	limitedReader := io.LimitReader(reader, MaxContentSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	return data, nil
}

// handleDeletedItem sends a deletion change for a removed item.
func (c *Connector) handleDeletedItem(
	ctx context.Context,
	deleted *files.DeletedMetadata,
	changesChan chan<- domain.RawDocumentChange,
) error {
	if changesChan == nil {
		return nil
	}

	change := domain.RawDocumentChange{
		Type: domain.ChangeDeleted,
		Document: domain.RawDocument{
			SourceID: c.sourceID,
			URI:      fmt.Sprintf("dropbox://files%s", deleted.PathLower),
		},
	}
	return c.sendChange(ctx, changesChan, &change)
}

// emitDocument sends a document to the appropriate channel.
func (c *Connector) emitDocument(
	ctx context.Context,
	doc *domain.RawDocument,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	if docsChan != nil {
		if err := c.sendDocument(ctx, docsChan, doc); err != nil {
			return err
		}
	}

	if changesChan != nil {
		change := domain.RawDocumentChange{
			Type:     domain.ChangeCreated,
			Document: *doc,
		}
		if err := c.sendChange(ctx, changesChan, &change); err != nil {
			return err
		}
	}

	return nil
}

// createClient creates a Dropbox files client with the given access token.
func (c *Connector) createClient(accessToken string) files.Client {
	config := dropbox.Config{
		Token: accessToken,
	}
	return files.New(config)
}

// createUsersClient creates a Dropbox users client with the given access token.
func (c *Connector) createUsersClient(accessToken string) users.Client {
	config := dropbox.Config{
		Token: accessToken,
	}
	return users.New(config)
}

// sendDocument sends a document to the channel or returns on context cancellation.
func (c *Connector) sendDocument(
	ctx context.Context, docsChan chan<- domain.RawDocument, doc *domain.RawDocument,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case docsChan <- *doc:
		return nil
	}
}

// sendChange sends a change to the channel.
func (c *Connector) sendChange(
	ctx context.Context,
	changesChan chan<- domain.RawDocumentChange,
	change *domain.RawDocumentChange,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changesChan <- *change:
		return nil
	}
}

// checkClosed returns an error if the connector is closed.
func (c *Connector) checkClosed() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return domain.ErrConnectorClosed
	}
	return nil
}

// Watch is not supported for Dropbox (no webhooks in CLI).
func (c *Connector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	return nil, domain.ErrNotImplemented
}

// GetAccountIdentifier fetches the Dropbox account email for the authenticated user.
func (c *Connector) GetAccountIdentifier(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

// Close releases resources.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// isResetError checks if the error indicates the cursor has been reset.
func isResetError(err error) bool {
	if err == nil {
		return false
	}
	// Dropbox SDK returns specific error types for cursor reset
	// Check the error message for reset indicators
	errStr := err.Error()
	return strings.Contains(errStr, "reset") || strings.Contains(errStr, "expired")
}
