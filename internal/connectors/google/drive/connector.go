package drive

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	"github.com/custodia-labs/sercha-cli/internal/connectors/google"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// Connector fetches documents from Google Drive.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	rateLimiter   *google.RateLimiter
	mu            sync.Mutex
	closed        bool
}

// New creates a new Google Drive connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		rateLimiter:   google.NewRateLimiter(google.ServiceDrive),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "google-drive"
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

// Validate checks if the Google Drive connector is properly configured.
func (c *Connector) Validate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return domain.ErrConnectorClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	ts := google.NewTokenSource(ctx, c.tokenProvider)
	svc, err := google.NewDriveService(ctx, ts)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	_, err = svc.About.Get().Fields("user").Context(ctx).Do()
	if err != nil {
		if google.IsUnauthorized(err) {
			return domain.ErrAuthInvalid
		}
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	return nil
}

// FullSync fetches all documents from Google Drive.
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

	ts := google.NewTokenSource(ctx, c.tokenProvider)
	svc, err := google.NewDriveService(ctx, ts)
	if err != nil {
		return fmt.Errorf("create drive service: %w", err)
	}

	startTokenResp, err := svc.Changes.GetStartPageToken().Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("get start page token: %w", google.WrapError(err))
	}

	cursor := NewCursor()
	cursor.StartPageToken = startTokenResp.StartPageToken

	if err := c.fetchAllFiles(ctx, svc, docsChan); err != nil {
		return err
	}

	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// fetchAllFiles fetches all files matching the config.
func (c *Connector) fetchAllFiles(
	ctx context.Context, svc *drive.Service, docsChan chan<- domain.RawDocument,
) error {
	var pageToken string

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		files, err := c.listFiles(ctx, svc, pageToken)
		if err != nil {
			return fmt.Errorf("list files: %w", google.WrapError(err))
		}

		if err := c.processFiles(ctx, svc, files.Files, docsChan); err != nil {
			return err
		}

		pageToken = files.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return nil
}

// listFiles creates and executes a file list request.
func (c *Connector) listFiles(ctx context.Context, svc *drive.Service, pageToken string) (*drive.FileList, error) {
	const fileFields = "nextPageToken, files(id, name, mimeType, modifiedTime, size, parents, webViewLink, trashed)"
	req := svc.Files.List().
		PageSize(c.config.MaxResults).
		Fields(googleapi.Field(fileFields))

	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	if len(c.config.FolderIDs) > 0 {
		req = req.Q("(" + buildFolderQuery(c.config.FolderIDs) + ")")
	}

	return req.Context(ctx).Do()
}

// buildFolderQuery builds a Drive query for specific folders.
func buildFolderQuery(folderIDs []string) string {
	if len(folderIDs) == 0 {
		return ""
	}
	result := fmt.Sprintf("'%s' in parents", folderIDs[0])
	for i := 1; i < len(folderIDs); i++ {
		result += fmt.Sprintf(" or '%s' in parents", folderIDs[i])
	}
	return result
}

// processFiles converts files to documents and sends them to the channel.
func (c *Connector) processFiles(
	ctx context.Context, svc *drive.Service, files []*drive.File, docsChan chan<- domain.RawDocument,
) error {
	for _, file := range files {
		if !ShouldSyncFile(file, c.config) {
			continue
		}

		rawDoc, err := FileToRawDocument(ctx, svc, file, c.sourceID)
		if err != nil || rawDoc == nil {
			continue
		}

		if err := c.sendDocument(ctx, docsChan, rawDoc); err != nil {
			return err
		}
	}
	return nil
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
		return fmt.Errorf("invalid cursor, full sync required: cursor has no page token")
	}

	ts := google.NewTokenSource(ctx, c.tokenProvider)
	svc, err := google.NewDriveService(ctx, ts)
	if err != nil {
		return fmt.Errorf("create drive service: %w", err)
	}

	newStartPageToken, err := c.processChanges(ctx, svc, cursor.StartPageToken, changesChan)
	if err != nil {
		return err
	}

	cursor.StartPageToken = newStartPageToken
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// processChanges fetches and processes all changes.
func (c *Connector) processChanges(
	ctx context.Context,
	svc *drive.Service,
	pageToken string,
	changesChan chan<- domain.RawDocumentChange,
) (string, error) {
	var newStartPageToken string

	for {
		if err := ctx.Err(); err != nil {
			return "", nil
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return "", err
		}

		changes, err := c.listChanges(ctx, svc, pageToken)
		if err != nil {
			return "", fmt.Errorf("list changes: %w", google.WrapError(err))
		}

		if err := c.processChangeList(ctx, svc, changes.Changes, changesChan); err != nil {
			return "", err
		}

		pageToken = changes.NextPageToken
		if changes.NewStartPageToken != "" {
			newStartPageToken = changes.NewStartPageToken
			break
		}
	}

	return newStartPageToken, nil
}

// listChanges creates and executes a changes list request.
func (c *Connector) listChanges(
	ctx context.Context, svc *drive.Service, pageToken string,
) (*drive.ChangeList, error) {
	const changesFields = "nextPageToken, newStartPageToken, " +
		"changes(fileId, removed, file(id, name, mimeType, modifiedTime, size, parents, webViewLink, trashed))"

	return svc.Changes.List(pageToken).
		Fields(googleapi.Field(changesFields)).
		PageSize(c.config.MaxResults).
		Context(ctx).
		Do()
}

// processChangeList processes a batch of changes.
func (c *Connector) processChangeList(
	ctx context.Context,
	svc *drive.Service,
	changes []*drive.Change,
	changesChan chan<- domain.RawDocumentChange,
) error {
	for _, change := range changes {
		if err := c.processChange(ctx, svc, change, changesChan); err != nil {
			return err
		}
	}
	return nil
}

// processChange handles a single change.
func (c *Connector) processChange(
	ctx context.Context,
	svc *drive.Service,
	change *drive.Change,
	changesChan chan<- domain.RawDocumentChange,
) error {
	if change.Removed || change.File == nil || change.File.Trashed {
		return c.sendDeletion(ctx, change.FileId, changesChan)
	}

	if !ShouldSyncFile(change.File, c.config) {
		return nil
	}

	rawDoc, err := FileToRawDocument(ctx, svc, change.File, c.sourceID)
	if err != nil || rawDoc == nil {
		return nil
	}

	return c.sendChange(ctx, changesChan, domain.ChangeUpdated, rawDoc)
}

// sendDeletion sends a deletion change to the channel.
func (c *Connector) sendDeletion(
	ctx context.Context, fileID string, changesChan chan<- domain.RawDocumentChange,
) error {
	change := domain.RawDocumentChange{
		Type: domain.ChangeDeleted,
		Document: domain.RawDocument{
			SourceID: c.sourceID,
			URI:      fmt.Sprintf("gdrive://files/%s", fileID),
		},
	}
	return c.sendChangeRaw(ctx, changesChan, &change)
}

// sendChange sends a change to the channel.
func (c *Connector) sendChange(
	ctx context.Context,
	changesChan chan<- domain.RawDocumentChange,
	changeType domain.ChangeType,
	doc *domain.RawDocument,
) error {
	change := domain.RawDocumentChange{
		Type:     changeType,
		Document: *doc,
	}
	return c.sendChangeRaw(ctx, changesChan, &change)
}

// sendChangeRaw sends a raw change to the channel.
func (c *Connector) sendChangeRaw(
	ctx context.Context, changesChan chan<- domain.RawDocumentChange, change *domain.RawDocumentChange,
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

// Watch is not supported for Google Drive (no webhooks in CLI).
func (c *Connector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	return nil, domain.ErrNotImplemented
}

// GetAccountIdentifier fetches the Google account email for the authenticated user.
func (c *Connector) GetAccountIdentifier(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := google.GetUserInfo(ctx, accessToken)
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
