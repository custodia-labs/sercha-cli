package calendar

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
	"github.com/custodia-labs/sercha-cli/internal/logger"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

// Connector fetches events from Microsoft Calendar via Microsoft Graph.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	rateLimiter   *microsoft.RateLimiter
	mu            sync.Mutex
	closed        bool
}

// New creates a new Microsoft Calendar connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		rateLimiter:   microsoft.NewRateLimiter(microsoft.ServiceCalendar),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "microsoft-calendar"
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

// Validate checks if the Calendar connector is properly configured.
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

	url := graphBaseURL + "/me/calendars?$top=1"
	resp, err := c.doRequest(ctx, url, token)
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

// FullSync fetches all events from Microsoft Calendar.
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

	logger.Debug("microsoft-calendar: starting full sync for source %s", c.sourceID)

	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		logger.Debug("microsoft-calendar: failed to get token: %v", err)
		return fmt.Errorf("get token: %w", err)
	}

	cursor := NewCursor()

	calendarIDs, err := c.getCalendarIDs(ctx, token)
	if err != nil {
		logger.Debug("microsoft-calendar: failed to get calendar IDs: %v", err)
		return err
	}

	logger.Debug("microsoft-calendar: found %d calendars to sync", len(calendarIDs))

	var successCount, failCount int
	for _, calID := range calendarIDs {
		logger.Debug("microsoft-calendar: syncing calendar %s", calID)
		err := c.syncCalendarEvents(ctx, token, calID, docsChan, cursor)
		if err != nil {
			logger.Warn("microsoft-calendar: failed to sync calendar %s: %v", calID, err)
			failCount++
		} else {
			logger.Debug("microsoft-calendar: successfully synced calendar %s", calID)
			successCount++
		}
	}

	logger.Debug("microsoft-calendar: sync complete - %d succeeded, %d failed", successCount, failCount)

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

	calendarIDs, err := c.getCalendarIDs(ctx, token)
	if err != nil {
		return err
	}

	for _, calID := range calendarIDs {
		c.syncCalendarWithRetry(ctx, token, calID, cursor, changesChan)
	}

	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// syncCalendarWithRetry syncs a calendar with retry on expired delta token.
func (c *Connector) syncCalendarWithRetry(
	ctx context.Context,
	token string,
	calID string,
	cursor *Cursor,
	changesChan chan<- domain.RawDocumentChange,
) {
	deltaLink := cursor.GetDeltaLink(calID)

	err := c.syncCalendarEventsIncremental(ctx, token, calID, deltaLink, changesChan, cursor)
	if err == nil {
		return
	}

	// If delta token is expired, retry with full sync for this calendar
	if deltaLink != "" && microsoft.IsDeltaTokenExpired(http.StatusGone) {
		//nolint:errcheck // Best-effort retry
		c.syncCalendarEventsIncremental(ctx, token, calID, "", changesChan, cursor)
	}
}

// getCalendarIDs returns the list of calendar IDs to sync.
func (c *Connector) getCalendarIDs(ctx context.Context, token string) ([]string, error) {
	if len(c.config.CalendarIDs) > 0 {
		return c.config.CalendarIDs, nil
	}
	return c.fetchAllCalendarIDs(ctx, token)
}

// fetchAllCalendarIDs retrieves all calendars the user can access.
func (c *Connector) fetchAllCalendarIDs(ctx context.Context, token string) ([]string, error) {
	var calendarIDs []string
	url := graphBaseURL + "/me/calendars"

	logger.Debug("microsoft-calendar: fetching calendars from Microsoft Graph")

	for url != "" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := c.doRequest(ctx, url, token)
		if err != nil {
			logger.Debug("microsoft-calendar: request error: %v", err)
			return nil, fmt.Errorf("list calendars: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		logger.Debug("microsoft-calendar: calendars response status %d, body length %d", resp.StatusCode, len(body))

		if resp.StatusCode != http.StatusOK {
			logger.Debug("microsoft-calendar: list calendars failed with body: %s", string(body))
			return nil, fmt.Errorf("list calendars failed: status %d", resp.StatusCode)
		}

		var listResp struct {
			Value    []struct{ ID string } `json:"value"`
			NextLink string                `json:"@odata.nextLink"`
		}
		if err := json.Unmarshal(body, &listResp); err != nil {
			logger.Debug("microsoft-calendar: failed to decode calendars response: %v", err)
			return nil, fmt.Errorf("decode calendars: %w", err)
		}

		logger.Debug("microsoft-calendar: found %d calendars in this page", len(listResp.Value))
		for _, cal := range listResp.Value {
			logger.Debug("microsoft-calendar: found calendar ID: %s", cal.ID)
			calendarIDs = append(calendarIDs, cal.ID)
		}

		url = listResp.NextLink
	}

	return calendarIDs, nil
}

// syncCalendarEvents syncs all events from a calendar for full sync.
func (c *Connector) syncCalendarEvents(
	ctx context.Context,
	token string,
	calendarID string,
	docsChan chan<- domain.RawDocument,
	cursor *Cursor,
) error {
	deltaURL := c.buildDeltaURL(calendarID)

	newDeltaLink, err := c.processDeltaPages(ctx, token, calendarID, deltaURL, docsChan, nil)
	if err != nil {
		return err
	}

	cursor.SetDeltaLink(calendarID, newDeltaLink)
	return nil
}

// syncCalendarEventsIncremental syncs event changes from a calendar.
func (c *Connector) syncCalendarEventsIncremental(
	ctx context.Context,
	token string,
	calendarID, deltaLink string,
	changesChan chan<- domain.RawDocumentChange,
	cursor *Cursor,
) error {
	var url string
	if deltaLink != "" {
		url = deltaLink
	} else {
		url = c.buildDeltaURL(calendarID)
	}

	newDeltaLink, err := c.processDeltaPages(ctx, token, calendarID, url, nil, changesChan)
	if err != nil {
		return err
	}

	cursor.SetDeltaLink(calendarID, newDeltaLink)
	return nil
}

// buildDeltaURL builds the initial delta query URL for a calendar.
// We use /events/delta to efficiently track changes (returns minimal fields: id, type, start, end).
// Then we fetch full event details via GET /events/{id} for each changed event.
func (c *Connector) buildDeltaURL(calendarID string) string {
	return fmt.Sprintf("%s/me/calendars/%s/events/delta", graphBaseURL, calendarID)
}

// deltaPageResult holds the result of fetching a single delta page.
type deltaPageResult struct {
	events    []json.RawMessage
	nextLink  string
	deltaLink string
}

// processDeltaPages processes all pages of a delta query.
func (c *Connector) processDeltaPages(
	ctx context.Context,
	token string,
	calendarID string,
	initialURL string,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) (string, error) {
	currentURL := initialURL
	var finalDeltaLink string
	var totalEvents int

	logger.Debug("microsoft-calendar: starting delta sync for calendar %s", calendarID)

	for currentURL != "" {
		if err := ctx.Err(); err != nil {
			return "", nil
		}

		pageResult, err := c.fetchDeltaPage(ctx, token, currentURL)
		if err != nil {
			logger.Debug("microsoft-calendar: delta page fetch error: %v", err)
			return "", err
		}

		logger.Debug("microsoft-calendar: fetched page with %d events", len(pageResult.events))
		totalEvents += len(pageResult.events)

		if err := c.processEvents(ctx, token, calendarID, pageResult.events, docsChan, changesChan); err != nil {
			logger.Debug("microsoft-calendar: process events error: %v", err)
			return "", err
		}

		currentURL = pageResult.nextLink
		if currentURL == "" {
			finalDeltaLink = pageResult.deltaLink
		}
	}

	logger.Debug("microsoft-calendar: delta sync complete for calendar %s, total events: %d", calendarID, totalEvents)

	return finalDeltaLink, nil
}

// fetchDeltaPage fetches a single page of delta results.
func (c *Connector) fetchDeltaPage(
	ctx context.Context, token, url string,
) (*deltaPageResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	logger.Debug("microsoft-calendar: fetching delta page: %s", url)

	resp, err := c.doRequest(ctx, url, token)
	if err != nil {
		return nil, fmt.Errorf("delta request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	logger.Debug("microsoft-calendar: delta response status %d, body length %d", resp.StatusCode, len(body))

	if resp.StatusCode == http.StatusGone {
		logger.Debug("microsoft-calendar: delta token expired (410 Gone)")
		return nil, microsoft.ErrDeltaTokenExpired
	}
	if resp.StatusCode != http.StatusOK {
		logger.Debug("microsoft-calendar: delta request failed with body: %s", string(body))
		return nil, fmt.Errorf("delta request failed: status %d: %w",
			resp.StatusCode, microsoft.WrapError(resp.StatusCode))
	}

	var deltaResp struct {
		Value     []json.RawMessage `json:"value"`
		NextLink  string            `json:"@odata.nextLink"`
		DeltaLink string            `json:"@odata.deltaLink"`
	}
	if err := json.Unmarshal(body, &deltaResp); err != nil {
		logger.Debug("microsoft-calendar: failed to decode delta response: %v", err)
		return nil, fmt.Errorf("decode delta response: %w", err)
	}

	logger.Debug("microsoft-calendar: delta response: %d events, hasNextLink=%v, hasDeltaLink=%v",
		len(deltaResp.Value), deltaResp.NextLink != "", deltaResp.DeltaLink != "")

	return &deltaPageResult{
		events:    deltaResp.Value,
		nextLink:  deltaResp.NextLink,
		deltaLink: deltaResp.DeltaLink,
	}, nil
}

// processEvents processes a batch of events from a delta response.
func (c *Connector) processEvents(
	ctx context.Context,
	token string,
	calendarID string,
	events []json.RawMessage,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	var processedCount, skippedCount int
	for i, raw := range events {
		// Log the first raw event to see what fields Microsoft returns
		if i == 0 {
			logger.Debug("microsoft-calendar: raw delta event JSON sample: %s", string(raw))
		}

		var eventWithRemoved EventWithRemoved
		if err := json.Unmarshal(raw, &eventWithRemoved); err != nil {
			logger.Debug("microsoft-calendar: failed to unmarshal event: %v", err)
			skippedCount++
			continue
		}

		if err := c.processSingleEvent(ctx, token, calendarID, &eventWithRemoved, docsChan, changesChan); err != nil {
			return err
		}
		processedCount++
	}
	logger.Debug("microsoft-calendar: processed %d events, skipped %d", processedCount, skippedCount)
	return nil
}

// processSingleEvent processes a single event from the delta response.
// The delta API only returns minimal fields (id, type, start, end), so we fetch
// full event details via GET /events/{id} for non-deleted events.
func (c *Connector) processSingleEvent(
	ctx context.Context,
	token string,
	calendarID string,
	eventWithRemoved *EventWithRemoved,
	docsChan chan<- domain.RawDocument,
	changesChan chan<- domain.RawDocumentChange,
) error {
	logger.Debug("microsoft-calendar: processing event %s", eventWithRemoved.ID)

	if IsEventRemoved(eventWithRemoved) {
		logger.Debug("microsoft-calendar: event %s is removed, handling deletion", eventWithRemoved.ID)
		return c.handleDeletedEvent(ctx, calendarID, eventWithRemoved.ID, changesChan)
	}

	if !ShouldSyncEvent(&eventWithRemoved.Event) {
		logger.Debug("microsoft-calendar: event %s filtered by ShouldSyncEvent", eventWithRemoved.ID)
		return nil
	}

	// Fetch full event details since delta only returns minimal fields
	fullEvent, err := c.fetchFullEvent(ctx, token, calendarID, eventWithRemoved.ID)
	if err != nil {
		logger.Debug("microsoft-calendar: failed to fetch full event %s: %v", eventWithRemoved.ID, err)
		return nil // Skip this event but continue with others
	}

	// Skip cancelled events in full sync
	if docsChan != nil && fullEvent.IsCancelled && !c.config.ShowCancelled {
		logger.Debug("microsoft-calendar: event %s skipped (cancelled)", fullEvent.ID)
		return nil
	}

	logger.Debug("microsoft-calendar: emitting event %s (subject: %s)", fullEvent.ID, fullEvent.Subject)
	doc := EventToRawDocument(fullEvent, calendarID, c.sourceID)
	return c.emitDocument(ctx, doc, docsChan, changesChan)
}

// fetchFullEvent fetches complete event details from the Graph API.
func (c *Connector) fetchFullEvent(ctx context.Context, token, calendarID, eventID string) (*Event, error) {
	url := fmt.Sprintf("%s/me/calendars/%s/events/%s", graphBaseURL, calendarID, eventID)

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, url, token)
	if err != nil {
		return nil, fmt.Errorf("fetch event: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch event failed: status %d", resp.StatusCode)
	}

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}

	return &event, nil
}

// handleDeletedEvent sends a deletion change for a removed event.
func (c *Connector) handleDeletedEvent(
	ctx context.Context, calendarID, eventID string, changesChan chan<- domain.RawDocumentChange,
) error {
	if changesChan == nil {
		return nil
	}

	change := domain.RawDocumentChange{
		Type: domain.ChangeDeleted,
		Document: domain.RawDocument{
			SourceID: c.sourceID,
			URI:      fmt.Sprintf("mscal://%s/events/%s", calendarID, eventID),
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
			Type:     domain.ChangeUpdated,
			Document: *doc,
		}
		if err := c.sendChange(ctx, changesChan, &change); err != nil {
			return err
		}
	}

	return nil
}

// doRequest performs an HTTP GET request with authentication.
func (c *Connector) doRequest(
	ctx context.Context, url, token string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	// Combine Prefer directives: timezone and page size (odata.maxpagesize for delta queries)
	req.Header.Set("Prefer", fmt.Sprintf("outlook.timezone=\"UTC\", odata.maxpagesize=%d", c.config.MaxResults))

	client := &http.Client{Timeout: 60 * time.Second}
	return client.Do(req)
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

// Watch is not supported for Calendar (no webhooks in CLI).
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
