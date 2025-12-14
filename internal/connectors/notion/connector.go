package notion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jomei/notionapi"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// Connector fetches pages and databases from Notion.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	client        *Client
	mu            sync.Mutex
	closed        bool
}

// New creates a new Notion connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		client:        NewClient(sourceID, tokenProvider),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "notion"
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

// Validate checks if the Notion connector is properly configured.
func (c *Connector) Validate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return domain.ErrConnectorClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate by performing a search with empty query
	_, err := c.client.Search(ctx, "", nil, "", 1)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	return nil
}

// FullSync fetches all pages and databases from Notion.
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
//
//nolint:gocognit,gocyclo,nestif // Sync logic requires iteration and type handling
func (c *Connector) runFullSync(ctx context.Context, docsChan chan<- domain.RawDocument) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	cursor := NewCursor()
	cursor.SetLastSyncTime(time.Now())

	blockExtractor := NewBlockExtractor(c.client, c.config.MaxBlockDepth, c.config.PageSize)
	commentFetcher := NewCommentFetcher(c.client, c.config.PageSize)

	// Search for all pages and databases
	var startCursor notionapi.Cursor
	seenIDs := make(map[string]bool)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := c.client.Search(ctx, "", nil, startCursor, c.config.PageSize)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		for _, obj := range resp.Results {
			if err := ctx.Err(); err != nil {
				return err
			}

			id := GetObjectID(obj)
			if seenIDs[id] {
				continue
			}
			seenIDs[id] = true

			// Track in cursor
			lastEdited := ParseLastEditedTime(obj)
			cursor.SetPageState(id, lastEdited, IsDatabase(obj))

			// Process based on type
			if IsPage(obj) && c.config.ShouldSyncPages() {
				if page, ok := obj.(*notionapi.Page); ok {
					if err := c.processPage(ctx, page, blockExtractor, commentFetcher, docsChan); err != nil {
						return err
					}
				}
			} else if IsDatabase(obj) && c.config.ShouldSyncDatabases() {
				if db, ok := obj.(*notionapi.Database); ok {
					if err := c.processDatabase(ctx, db, blockExtractor, commentFetcher, docsChan); err != nil {
						return err
					}
				}
			}
		}

		if !resp.HasMore {
			break
		}
		startCursor = resp.NextCursor
	}

	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// IncrementalSync fetches only changes since the last sync.
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
//nolint:gocognit,gocyclo,nestif,funlen // Sync logic requires complex change detection
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

	blockExtractor := NewBlockExtractor(c.client, c.config.MaxBlockDepth, c.config.PageSize)
	commentFetcher := NewCommentFetcher(c.client, c.config.PageSize)

	// Track which IDs we see in this sync
	seenIDs := make(map[string]bool)

	// Search for all pages and databases
	var startCursor notionapi.Cursor

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := c.client.Search(ctx, "", nil, startCursor, c.config.PageSize)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		for _, obj := range resp.Results {
			if err := ctx.Err(); err != nil {
				return err
			}

			id := GetObjectID(obj)
			seenIDs[id] = true

			lastEdited := ParseLastEditedTime(obj)
			isDB := IsDatabase(obj)

			// Check if this is new or updated
			prevState := cursor.GetPageState(id)
			isNew := prevState == nil
			isUpdated := !isNew && lastEdited.After(prevState.LastEditedTime)

			if isNew || isUpdated {
				// Update cursor state
				cursor.SetPageState(id, lastEdited, isDB)

				// Determine change type
				changeType := domain.ChangeUpdated
				if isNew {
					changeType = domain.ChangeCreated
				}

				// Process and emit change
				if IsPage(obj) && c.config.ShouldSyncPages() {
					if page, ok := obj.(*notionapi.Page); ok {
						err := c.processPageChange(
							ctx, page, blockExtractor, commentFetcher, changesChan, changeType,
						)
						if err != nil {
							return err
						}
					}
				} else if IsDatabase(obj) && c.config.ShouldSyncDatabases() {
					if db, ok := obj.(*notionapi.Database); ok {
						if err := c.processDatabaseChange(ctx, db, changesChan, changeType); err != nil {
							return err
						}
					}
				}
			}
		}

		if !resp.HasMore {
			break
		}
		startCursor = resp.NextCursor
	}

	// Detect deletions - pages in cursor but not seen
	for _, id := range cursor.GetAllPageIDs() {
		if seenIDs[id] {
			continue
		}
		// Page was deleted
		prevState := cursor.GetPageState(id)
		uri := fmt.Sprintf("notion://pages/%s", id)
		if prevState != nil && prevState.IsDatabase {
			uri = fmt.Sprintf("notion://databases/%s", id)
		}

		change := domain.RawDocumentChange{
			Type: domain.ChangeDeleted,
			Document: domain.RawDocument{
				SourceID: c.sourceID,
				URI:      uri,
			},
		}
		if err := c.sendChange(ctx, changesChan, &change); err != nil {
			return err
		}

		// Remove from cursor
		cursor.RemovePageState(id)
	}

	cursor.SetLastSyncTime(time.Now())
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// processPage processes a single page.
func (c *Connector) processPage(
	ctx context.Context,
	page *notionapi.Page,
	blockExtractor *BlockExtractor,
	commentFetcher *CommentFetcher,
	docsChan chan<- domain.RawDocument,
) error {
	// Extract block content (errors are non-fatal, we continue with empty content)
	content, _ := blockExtractor.ExtractContent(ctx, notionapi.BlockID(page.ID)) //nolint:errcheck

	// Fetch comments if enabled (errors are non-fatal)
	var comments []string
	if c.config.IncludeComments {
		comments, _ = commentFetcher.FetchComments(ctx, notionapi.BlockID(page.ID)) //nolint:errcheck
	}

	// Check if this is a database item
	var doc *domain.RawDocument
	if page.Parent.DatabaseID != "" {
		doc = PageWithProperties(page, content, c.sourceID, comments)
	} else {
		doc = PageToRawDocument(page, content, c.sourceID, comments)
	}

	return c.sendDocument(ctx, docsChan, doc)
}

// processDatabase processes a database and its items.
func (c *Connector) processDatabase(
	ctx context.Context,
	db *notionapi.Database,
	blockExtractor *BlockExtractor,
	commentFetcher *CommentFetcher,
	docsChan chan<- domain.RawDocument,
) error {
	// Emit the database itself
	doc := DatabaseToRawDocument(db, c.sourceID)
	if err := c.sendDocument(ctx, docsChan, doc); err != nil {
		return err
	}

	// Query and emit all database items
	var startCursor notionapi.Cursor

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := c.client.QueryDatabase(ctx, notionapi.DatabaseID(db.ID), startCursor, c.config.PageSize)
		if err != nil {
			// If we can't query, just emit the database without items
			return nil
		}

		for _, page := range resp.Results {
			if err := c.processPage(ctx, &page, blockExtractor, commentFetcher, docsChan); err != nil {
				return err
			}
		}

		if !resp.HasMore {
			break
		}
		startCursor = resp.NextCursor
	}

	return nil
}

// processPageChange processes a page change.
func (c *Connector) processPageChange(
	ctx context.Context,
	page *notionapi.Page,
	blockExtractor *BlockExtractor,
	commentFetcher *CommentFetcher,
	changesChan chan<- domain.RawDocumentChange,
	changeType domain.ChangeType,
) error {
	// Extract block content (errors are non-fatal, we continue with empty content)
	content, _ := blockExtractor.ExtractContent(ctx, notionapi.BlockID(page.ID)) //nolint:errcheck

	// Fetch comments if enabled (errors are non-fatal)
	var comments []string
	if c.config.IncludeComments {
		comments, _ = commentFetcher.FetchComments(ctx, notionapi.BlockID(page.ID)) //nolint:errcheck
	}

	// Check if this is a database item
	var doc *domain.RawDocument
	if page.Parent.DatabaseID != "" {
		doc = PageWithProperties(page, content, c.sourceID, comments)
	} else {
		doc = PageToRawDocument(page, content, c.sourceID, comments)
	}

	change := domain.RawDocumentChange{
		Type:     changeType,
		Document: *doc,
	}
	return c.sendChange(ctx, changesChan, &change)
}

// processDatabaseChange processes a database change.
func (c *Connector) processDatabaseChange(
	ctx context.Context,
	db *notionapi.Database,
	changesChan chan<- domain.RawDocumentChange,
	changeType domain.ChangeType,
) error {
	doc := DatabaseToRawDocument(db, c.sourceID)

	change := domain.RawDocumentChange{
		Type:     changeType,
		Document: *doc,
	}
	return c.sendChange(ctx, changesChan, &change)
}

// sendDocument sends a document to the channel.
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

// Watch is not supported for Notion (no webhooks in CLI).
func (c *Connector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	return nil, domain.ErrNotImplemented
}

// GetAccountIdentifier fetches the Notion workspace name for the authenticated user.
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
