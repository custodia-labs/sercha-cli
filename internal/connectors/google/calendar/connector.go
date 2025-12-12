package calendar

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/calendar/v3"

	"github.com/custodia-labs/sercha-cli/internal/connectors/google"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// Connector fetches events from Google Calendar.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	rateLimiter   *google.RateLimiter
	mu            sync.Mutex
	closed        bool
}

// New creates a new Google Calendar connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		rateLimiter:   google.NewRateLimiter(google.ServiceCalendar),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "google-calendar"
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

	ts := google.NewTokenSource(ctx, c.tokenProvider)
	svc, err := google.NewCalendarService(ctx, ts)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	_, err = svc.CalendarList.List().MaxResults(1).Context(ctx).Do()
	if err != nil {
		if google.IsUnauthorized(err) {
			return domain.ErrAuthInvalid
		}
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	return nil
}

// FullSync fetches all events from Google Calendar.
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
	svc, err := google.NewCalendarService(ctx, ts)
	if err != nil {
		return fmt.Errorf("create calendar service: %w", err)
	}

	cursor := NewCursor()

	calendarIDs, err := c.getCalendarIDs(ctx, svc)
	if err != nil {
		return err
	}

	for _, calID := range calendarIDs {
		// Errors for individual calendars are not fatal
		//nolint:errcheck // Best-effort sync per calendar
		c.syncCalendarEvents(ctx, svc, calID, docsChan, cursor)
	}

	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// IncrementalSync fetches only changes since the last sync using syncTokens.
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
		return fmt.Errorf("invalid cursor, full sync required: cursor has no sync token")
	}

	ts := google.NewTokenSource(ctx, c.tokenProvider)
	svc, err := google.NewCalendarService(ctx, ts)
	if err != nil {
		return fmt.Errorf("create calendar service: %w", err)
	}

	calendarIDs, err := c.getCalendarIDs(ctx, svc)
	if err != nil {
		return err
	}

	for _, calID := range calendarIDs {
		c.syncCalendarWithRetry(ctx, svc, calID, cursor, changesChan)
	}

	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// syncCalendarWithRetry syncs a calendar with retry on expired sync token.
func (c *Connector) syncCalendarWithRetry(
	ctx context.Context,
	svc *calendar.Service,
	calID string,
	cursor *Cursor,
	changesChan chan<- domain.RawDocumentChange,
) {
	syncToken := cursor.GetSyncToken(calID)

	err := c.syncCalendarEventsIncremental(ctx, svc, calID, syncToken, changesChan, cursor)
	if err == nil {
		return
	}

	// If sync token is expired, retry with full sync for this calendar
	if syncToken != "" && google.IsSyncTokenExpired(err) {
		//nolint:errcheck // Best-effort retry
		c.syncCalendarEventsIncremental(ctx, svc, calID, "", changesChan, cursor)
	}
}

// Watch is not supported for Calendar (no webhooks in CLI).
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

// checkClosed returns an error if the connector is closed.
func (c *Connector) checkClosed() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return domain.ErrConnectorClosed
	}
	return nil
}

// getCalendarIDs returns the list of calendar IDs to sync.
func (c *Connector) getCalendarIDs(ctx context.Context, svc *calendar.Service) ([]string, error) {
	if len(c.config.CalendarIDs) > 0 {
		return c.config.CalendarIDs, nil
	}
	return c.fetchAllCalendarIDs(ctx, svc)
}

// fetchAllCalendarIDs retrieves all calendars the user can access.
func (c *Connector) fetchAllCalendarIDs(ctx context.Context, svc *calendar.Service) ([]string, error) {
	var calendarIDs []string
	var pageToken string

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		list, err := c.listCalendars(ctx, svc, pageToken)
		if err != nil {
			return nil, fmt.Errorf("list calendars: %w", google.WrapError(err))
		}

		for _, cal := range list.Items {
			calendarIDs = append(calendarIDs, cal.Id)
		}

		pageToken = list.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return calendarIDs, nil
}

// listCalendars creates and executes a calendar list request.
func (c *Connector) listCalendars(
	ctx context.Context, svc *calendar.Service, pageToken string,
) (*calendar.CalendarList, error) {
	req := svc.CalendarList.List()
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	return req.Context(ctx).Do()
}

// syncCalendarEvents syncs all events from a calendar for full sync.
func (c *Connector) syncCalendarEvents(
	ctx context.Context,
	svc *calendar.Service,
	calendarID string,
	docsChan chan<- domain.RawDocument,
	cursor *Cursor,
) error {
	var pageToken string

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		events, err := c.listEvents(ctx, svc, calendarID, pageToken)
		if err != nil {
			return fmt.Errorf("list events: %w", google.WrapError(err))
		}

		if err := c.processEventsForFullSync(ctx, events.Items, calendarID, docsChan); err != nil {
			return err
		}

		pageToken = events.NextPageToken
		if pageToken == "" {
			if events.NextSyncToken != "" {
				cursor.SetSyncToken(calendarID, events.NextSyncToken)
			}
			break
		}
	}

	return nil
}

// listEvents creates and executes an events list request.
func (c *Connector) listEvents(
	ctx context.Context, svc *calendar.Service, calendarID, pageToken string,
) (*calendar.Events, error) {
	req := svc.Events.List(calendarID).
		MaxResults(c.config.MaxResults).
		ShowDeleted(c.config.ShowDeleted).
		SingleEvents(c.config.SingleEvents)

	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	return req.Context(ctx).Do()
}

// processEventsForFullSync processes events for a full sync.
func (c *Connector) processEventsForFullSync(
	ctx context.Context,
	events []*calendar.Event,
	calendarID string,
	docsChan chan<- domain.RawDocument,
) error {
	for _, event := range events {
		if !ShouldSyncEvent(event) || event.Status == "cancelled" {
			continue
		}

		rawDoc := EventToRawDocument(event, calendarID, c.sourceID)
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

// syncCalendarEventsIncremental syncs event changes from a calendar.
func (c *Connector) syncCalendarEventsIncremental(
	ctx context.Context,
	svc *calendar.Service,
	calendarID, syncToken string,
	changesChan chan<- domain.RawDocumentChange,
	cursor *Cursor,
) error {
	var pageToken string

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		events, err := c.listEventsIncremental(ctx, svc, calendarID, syncToken, pageToken)
		if err != nil {
			if google.IsSyncTokenExpired(err) {
				return google.ErrSyncTokenExpired
			}
			return fmt.Errorf("list events: %w", google.WrapError(err))
		}

		if err := c.processEventsForIncremental(ctx, events.Items, calendarID, changesChan); err != nil {
			return err
		}

		pageToken = events.NextPageToken
		if pageToken == "" {
			if events.NextSyncToken != "" {
				cursor.SetSyncToken(calendarID, events.NextSyncToken)
			}
			break
		}
	}

	return nil
}

// listEventsIncremental creates and executes an events list request for incremental sync.
func (c *Connector) listEventsIncremental(
	ctx context.Context,
	svc *calendar.Service,
	calendarID, syncToken, pageToken string,
) (*calendar.Events, error) {
	req := svc.Events.List(calendarID).
		MaxResults(c.config.MaxResults).
		ShowDeleted(true). // Always need deleted for incremental
		SingleEvents(c.config.SingleEvents)

	if syncToken != "" {
		req = req.SyncToken(syncToken)
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	return req.Context(ctx).Do()
}

// processEventsForIncremental processes events for an incremental sync.
func (c *Connector) processEventsForIncremental(
	ctx context.Context,
	events []*calendar.Event,
	calendarID string,
	changesChan chan<- domain.RawDocumentChange,
) error {
	for _, event := range events {
		if !ShouldSyncEvent(event) {
			continue
		}

		change := c.eventToChange(event, calendarID)
		if err := c.sendChange(ctx, changesChan, &change); err != nil {
			return err
		}
	}
	return nil
}

// eventToChange converts an event to a change.
func (c *Connector) eventToChange(event *calendar.Event, calendarID string) domain.RawDocumentChange {
	if event.Status == "cancelled" {
		return domain.RawDocumentChange{
			Type: domain.ChangeDeleted,
			Document: domain.RawDocument{
				SourceID: c.sourceID,
				URI:      fmt.Sprintf("gcal://%s/events/%s", calendarID, event.Id),
			},
		}
	}

	rawDoc := EventToRawDocument(event, calendarID, c.sourceID)
	return domain.RawDocumentChange{
		Type:     domain.ChangeUpdated,
		Document: *rawDoc,
	}
}

// sendChange sends a change to the channel.
func (c *Connector) sendChange(
	ctx context.Context, changesChan chan<- domain.RawDocumentChange, change *domain.RawDocumentChange,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changesChan <- *change:
		return nil
	}
}
