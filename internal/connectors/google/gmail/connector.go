package gmail

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/gmail/v1"

	"github.com/custodia-labs/sercha-cli/internal/connectors/google"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// Connector fetches emails from Gmail.
type Connector struct {
	sourceID      string
	config        *Config
	tokenProvider driven.TokenProvider
	rateLimiter   *google.RateLimiter
	mu            sync.Mutex
	closed        bool
}

// New creates a new Gmail connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		rateLimiter:   google.NewRateLimiter(google.ServiceGmail),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "gmail"
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

// Validate checks if the Gmail connector is properly configured.
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
	svc, err := google.NewGmailService(ctx, ts)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	_, err = svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		if google.IsUnauthorized(err) {
			return domain.ErrAuthInvalid
		}
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	return nil
}

// FullSync fetches all emails from Gmail.
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
	svc, err := google.NewGmailService(ctx, ts)
	if err != nil {
		return fmt.Errorf("create gmail service: %w", err)
	}

	profile, err := svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("get profile: %w", google.WrapError(err))
	}

	cursor := NewCursor()
	cursor.HistoryID = profile.HistoryId

	if err := c.fetchAllMessages(ctx, svc, docsChan); err != nil {
		return err
	}

	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// fetchAllMessages fetches all messages matching the config.
func (c *Connector) fetchAllMessages(
	ctx context.Context, svc *gmail.Service, docsChan chan<- domain.RawDocument,
) error {
	var pageToken string

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		resp, err := c.listMessages(ctx, svc, pageToken)
		if err != nil {
			return fmt.Errorf("list messages: %w", google.WrapError(err))
		}

		if err := c.processMessageRefs(ctx, svc, resp.Messages, docsChan); err != nil {
			return err
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return nil
}

// listMessages creates and executes a message list request.
func (c *Connector) listMessages(
	ctx context.Context, svc *gmail.Service, pageToken string,
) (*gmail.ListMessagesResponse, error) {
	req := svc.Users.Messages.List("me").
		MaxResults(c.config.MaxResults).
		IncludeSpamTrash(c.config.IncludeSpamTrash)

	if len(c.config.LabelIDs) > 0 {
		req = req.LabelIds(c.config.LabelIDs...)
	}
	if c.config.Query != "" {
		req = req.Q(c.config.Query)
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	return req.Context(ctx).Do()
}

// processMessageRefs fetches full messages and sends them to the channel.
func (c *Connector) processMessageRefs(
	ctx context.Context,
	svc *gmail.Service,
	refs []*gmail.Message,
	docsChan chan<- domain.RawDocument,
) error {
	for _, msgRef := range refs {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		msg, err := c.fetchMessage(ctx, svc, msgRef.Id)
		if err != nil {
			continue
		}

		if !ShouldSyncMessage(msg, c.config) {
			continue
		}

		if err := c.sendDocument(ctx, docsChan, MessageToRawDocument(msg, c.sourceID)); err != nil {
			return err
		}
	}
	return nil
}

// fetchMessage retrieves a full message by ID in raw RFC 2822 format.
func (c *Connector) fetchMessage(ctx context.Context, svc *gmail.Service, id string) (*gmail.Message, error) {
	return svc.Users.Messages.Get("me", id).Format("raw").Context(ctx).Do()
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

// IncrementalSync fetches only changes since the last sync using History API.
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
		return fmt.Errorf("invalid cursor, full sync required: cursor has no history ID")
	}

	ts := google.NewTokenSource(ctx, c.tokenProvider)
	svc, err := google.NewGmailService(ctx, ts)
	if err != nil {
		return fmt.Errorf("create gmail service: %w", err)
	}

	latestHistoryID, err := c.processHistory(ctx, svc, cursor.HistoryID, changesChan)
	if err != nil {
		return err
	}

	cursor.HistoryID = latestHistoryID
	return &driven.SyncComplete{NewCursor: cursor.Encode()}
}

// processHistory fetches and processes all history records.
func (c *Connector) processHistory(
	ctx context.Context,
	svc *gmail.Service,
	startHistoryID uint64,
	changesChan chan<- domain.RawDocumentChange,
) (uint64, error) {
	var pageToken string
	latestHistoryID := startHistoryID

	for {
		if err := ctx.Err(); err != nil {
			return 0, nil
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return 0, err
		}

		history, err := c.listHistory(ctx, svc, startHistoryID, pageToken)
		if err != nil {
			if google.IsHistoryIDExpired(err) {
				return 0, fmt.Errorf("%w: full sync required", google.ErrHistoryIDExpired)
			}
			return 0, fmt.Errorf("list history: %w", google.WrapError(err))
		}

		if history.HistoryId > latestHistoryID {
			latestHistoryID = history.HistoryId
		}

		if err := c.processHistoryRecords(ctx, svc, history.History, changesChan); err != nil {
			return 0, err
		}

		pageToken = history.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return latestHistoryID, nil
}

// listHistory creates and executes a history list request.
func (c *Connector) listHistory(
	ctx context.Context, svc *gmail.Service, startHistoryID uint64, pageToken string,
) (*gmail.ListHistoryResponse, error) {
	req := svc.Users.History.List("me").
		StartHistoryId(startHistoryID).
		MaxResults(c.config.MaxResults)

	if len(c.config.LabelIDs) > 0 {
		req = req.LabelId(c.config.LabelIDs[0])
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	return req.Context(ctx).Do()
}

// processHistoryRecords processes a batch of history records.
func (c *Connector) processHistoryRecords(
	ctx context.Context,
	svc *gmail.Service,
	records []*gmail.History,
	changesChan chan<- domain.RawDocumentChange,
) error {
	for _, h := range records {
		if err := c.processAddedMessages(ctx, svc, h.MessagesAdded, changesChan); err != nil {
			return err
		}
		if err := c.processDeletedMessages(ctx, h.MessagesDeleted, changesChan); err != nil {
			return err
		}
		if err := c.processLabelChanges(ctx, svc, h.LabelsAdded, h.LabelsRemoved, changesChan); err != nil {
			return err
		}
	}
	return nil
}

// processAddedMessages handles newly added messages.
func (c *Connector) processAddedMessages(
	ctx context.Context,
	svc *gmail.Service,
	added []*gmail.HistoryMessageAdded,
	changesChan chan<- domain.RawDocumentChange,
) error {
	for _, a := range added {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return err
		}

		msg, err := c.fetchMessage(ctx, svc, a.Message.Id)
		if err != nil {
			continue
		}

		if !ShouldSyncMessage(msg, c.config) {
			continue
		}

		doc := MessageToRawDocument(msg, c.sourceID)
		if err := c.sendChange(ctx, changesChan, domain.ChangeCreated, doc); err != nil {
			return err
		}
	}
	return nil
}

// processDeletedMessages handles deleted messages.
func (c *Connector) processDeletedMessages(
	ctx context.Context, deleted []*gmail.HistoryMessageDeleted, changesChan chan<- domain.RawDocumentChange,
) error {
	for _, d := range deleted {
		change := domain.RawDocumentChange{
			Type: domain.ChangeDeleted,
			Document: domain.RawDocument{
				SourceID: c.sourceID,
				URI:      fmt.Sprintf("gmail://messages/%s", d.Message.Id),
			},
		}
		if err := c.sendChangeRaw(ctx, changesChan, &change); err != nil {
			return err
		}
	}
	return nil
}

// processLabelChanges handles label additions and removals as updates.
func (c *Connector) processLabelChanges(
	ctx context.Context,
	svc *gmail.Service,
	added []*gmail.HistoryLabelAdded,
	removed []*gmail.HistoryLabelRemoved,
	changesChan chan<- domain.RawDocumentChange,
) error {
	// Process label additions
	for _, lblAdd := range added {
		if err := c.sendLabelChangeUpdate(ctx, svc, lblAdd.Message.Id, changesChan); err != nil {
			return err
		}
	}

	// Process label removals
	for _, lblRemove := range removed {
		if err := c.sendLabelChangeUpdate(ctx, svc, lblRemove.Message.Id, changesChan); err != nil {
			return err
		}
	}

	return nil
}

// sendLabelChangeUpdate fetches a message and sends it as an update.
func (c *Connector) sendLabelChangeUpdate(
	ctx context.Context, svc *gmail.Service, messageID string, changesChan chan<- domain.RawDocumentChange,
) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	msg, err := c.fetchMessage(ctx, svc, messageID)
	if err != nil {
		return nil // Skip individual message errors
	}

	return c.sendChange(ctx, changesChan, domain.ChangeUpdated, MessageToRawDocument(msg, c.sourceID))
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

// Watch is not supported for Gmail (no webhooks in CLI).
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
