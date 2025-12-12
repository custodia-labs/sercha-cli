package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/connectors/microsoft"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

// MaxContentSize is the maximum file size to download (5MB).
const MaxContentSize = 5 * 1024 * 1024

// Connector fetches files from OneDrive via Microsoft Graph.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	rateLimiter   *microsoft.RateLimiter
	mu            sync.Mutex
	closed        bool
}

// New creates a new OneDrive connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		rateLimiter:   microsoft.NewRateLimiter(microsoft.ServiceOneDrive),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "onedrive"
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

// Validate checks if the OneDrive connector is properly configured.
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

	url := graphBaseURL + "/me/drive"
	resp, err := c.doRequest(ctx, http.MethodGet, url, token)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return domain.ErrAuthInvalid
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", domain.ErrAuthRequired, resp.StatusCode)
	}

	return nil
}

// FullSync fetches all files from OneDrive.
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

	cursor := NewCursor()

	// Build initial delta URL
	deltaURL := c.buildDeltaURL()

	// Process all pages
	newDeltaLink, err := c.processDeltaPages(ctx, token, deltaURL, docsChan, nil)
	if err != nil {
		return err
	}

	cursor.SetDeltaLink(newDeltaLink)
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// IncrementalSync fetches only changes since the last sync using delta queries.
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
		return fmt.Errorf("invalid cursor, full sync required: cursor has no delta link")
	}

	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	// Use the stored delta link
	newDeltaLink, err := c.processDeltaPages(ctx, token, cursor.GetDeltaLink(), nil, changesChan)
	if err != nil {
		if microsoft.IsDeltaTokenExpired(http.StatusGone) {
			return fmt.Errorf("%w: full sync required", microsoft.ErrDeltaTokenExpired)
		}
		return err
	}

	cursor.SetDeltaLink(newDeltaLink)
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// buildDeltaURL builds the initial delta query URL.
func (c *Connector) buildDeltaURL() string {
	// If folder IDs specified, use first folder; otherwise use root
	if len(c.config.FolderIDs) > 0 {
		return fmt.Sprintf("%s/me/drive/items/%s/delta?$top=%d",
			graphBaseURL, c.config.FolderIDs[0], c.config.MaxResults)
	}
	return fmt.Sprintf("%s/me/drive/root/delta?$top=%d",
		graphBaseURL, c.config.MaxResults)
}

// deltaPageResult holds the result of fetching a single delta page.
type deltaPageResult struct {
	items     []json.RawMessage
	nextLink  string
	deltaLink string
}

// processDeltaPages processes all pages of a delta query.
func (c *Connector) processDeltaPages(
	ctx context.Context,
	token string,
	initialURL string,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) (string, error) {
	currentURL := initialURL
	var finalDeltaLink string

	for currentURL != "" {
		if err := ctx.Err(); err != nil {
			return "", nil
		}

		pageResult, err := c.fetchDeltaPage(ctx, token, currentURL)
		if err != nil {
			return "", err
		}

		if err := c.processItems(ctx, token, pageResult.items, docsChan, changesChan); err != nil {
			return "", err
		}

		currentURL = pageResult.nextLink
		if currentURL == "" {
			finalDeltaLink = pageResult.deltaLink
		}
	}

	return finalDeltaLink, nil
}

// fetchDeltaPage fetches a single page of delta results.
func (c *Connector) fetchDeltaPage(
	ctx context.Context, token, url string,
) (*deltaPageResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodGet, url, token)
	if err != nil {
		return nil, fmt.Errorf("delta request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusGone {
		return nil, microsoft.ErrDeltaTokenExpired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("delta request failed: status %d: %w",
			resp.StatusCode, microsoft.WrapError(resp.StatusCode))
	}

	var deltaResp struct {
		Value     []json.RawMessage `json:"value"`
		NextLink  string            `json:"@odata.nextLink"`
		DeltaLink string            `json:"@odata.deltaLink"`
	}
	if err := json.Unmarshal(body, &deltaResp); err != nil {
		return nil, fmt.Errorf("decode delta response: %w", err)
	}

	return &deltaPageResult{
		items:     deltaResp.Value,
		nextLink:  deltaResp.NextLink,
		deltaLink: deltaResp.DeltaLink,
	}, nil
}

// processItems processes a batch of items from a delta response.
func (c *Connector) processItems(
	ctx context.Context,
	token string,
	items []json.RawMessage,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	for _, raw := range items {
		var itemWithRemoved DriveItemWithRemoved
		if err := json.Unmarshal(raw, &itemWithRemoved); err != nil {
			continue
		}

		if err := c.processSingleItem(ctx, token, &itemWithRemoved, docsChan, changesChan); err != nil {
			return err
		}
	}
	return nil
}

// processSingleItem processes a single item from the delta response.
func (c *Connector) processSingleItem(
	ctx context.Context,
	token string,
	itemWithRemoved *DriveItemWithRemoved,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	if IsItemRemoved(itemWithRemoved) {
		return c.handleDeletedItem(ctx, itemWithRemoved.ID, changesChan)
	}

	if !ShouldSyncFile(&itemWithRemoved.DriveItem, c.config) {
		return nil
	}

	// Fetch file content if it's a type we can normalise and small enough
	var content []byte
	if shouldDownloadContent(itemWithRemoved.GetMIMEType()) && itemWithRemoved.Size <= MaxContentSize {
		var err error
		content, err = c.downloadFileContent(ctx, token, itemWithRemoved.ID)
		if err != nil {
			// Continue without content on error
			content = nil
		}
	}

	doc := FileToRawDocument(&itemWithRemoved.DriveItem, content, c.sourceID)
	return c.emitDocument(ctx, doc, docsChan, changesChan)
}

// handleDeletedItem sends a deletion change for a removed item.
func (c *Connector) handleDeletedItem(
	ctx context.Context, itemID string, changesChan chan<- domain.RawDocumentChange,
) error {
	if changesChan == nil {
		return nil
	}

	change := domain.RawDocumentChange{
		Type: domain.ChangeDeleted,
		Document: domain.RawDocument{
			SourceID: c.sourceID,
			URI:      fmt.Sprintf("onedrive://files/%s", itemID),
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

// downloadFileContent downloads the content of a file.
func (c *Connector) downloadFileContent(ctx context.Context, token, itemID string) ([]byte, error) {
	url := fmt.Sprintf("%s/me/drive/items/%s/content", graphBaseURL, itemID)

	resp, err := c.doRequest(ctx, http.MethodGet, url, token)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, MaxContentSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	return data, nil
}

// doRequest performs an HTTP request with authentication.
func (c *Connector) doRequest(
	ctx context.Context, method, url, token string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	return client.Do(req)
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

// Watch is not supported for OneDrive (no webhooks in CLI).
func (c *Connector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	return nil, domain.ErrNotImplemented
}

// GetAccountIdentifier fetches the Microsoft account email for the authenticated user.
func (c *Connector) GetAccountIdentifier(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := microsoft.GetUserInfo(ctx, accessToken)
	if err != nil {
		return "", err
	}
	return userInfo.GetUserEmail(), nil
}

// Close releases resources.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}
