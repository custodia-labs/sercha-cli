package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jomei/notionapi"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Client wraps the notionapi client with rate limiting and token refresh.
type Client struct {
	client        *notionapi.Client
	sourceID      string
	tokenProvider driven.TokenProvider
	rateLimiter   *RateLimiter
}

// NewClient creates a new Notion API client.
func NewClient(sourceID string, tokenProvider driven.TokenProvider) *Client {
	return &Client{
		sourceID:      sourceID,
		tokenProvider: tokenProvider,
		rateLimiter:   NewRateLimiter(),
	}
}

// init initialises the client with a fresh token.
func (c *Client) init(ctx context.Context) error {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	c.client = notionapi.NewClient(notionapi.Token(token))
	return nil
}

// ensureClient ensures the client is initialised.
func (c *Client) ensureClient(ctx context.Context) error {
	if c.client == nil {
		return c.init(ctx)
	}
	return nil
}

// searchRequest is a custom struct for Notion search API that properly
// handles filter serialisation with omitempty on all fields.
type searchRequest struct {
	Query       string                `json:"query,omitempty"`
	Sort        *notionapi.SortObject `json:"sort,omitempty"`
	Filter      *searchFilter         `json:"filter,omitempty"` // Pointer to allow nil/omit
	StartCursor notionapi.Cursor      `json:"start_cursor,omitempty"`
	PageSize    int                   `json:"page_size,omitempty"`
}

// searchFilter is a custom struct with omitempty to avoid serialising empty values.
type searchFilter struct {
	Value    string `json:"value,omitempty"`
	Property string `json:"property,omitempty"`
}

// Search searches for pages and databases in the workspace.
// Uses a custom HTTP implementation to avoid the notionapi library's
// SearchFilter serialisation issue where empty filter fields cause API errors.
func (c *Client) Search(
	ctx context.Context,
	query string,
	filter *notionapi.SearchFilter,
	startCursor notionapi.Cursor,
	pageSize int,
) (*notionapi.SearchResponse, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Get fresh token for the request
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// Build request with proper omitempty handling
	req := &searchRequest{
		Query:    query,
		PageSize: pageSize,
	}
	if startCursor != "" {
		req.StartCursor = startCursor
	}
	// Only set filter if it has valid content (use pointer for proper omitempty)
	if filter != nil && filter.Property != "" {
		req.Filter = &searchFilter{
			Value:    filter.Value,
			Property: filter.Property,
		}
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal search request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.notion.com/v1/search", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Notion-Version", notionAPIVersion)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp notionapi.SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	return &searchResp, nil
}

// GetPage retrieves a page by ID.
func (c *Client) GetPage(ctx context.Context, pageID notionapi.PageID) (*notionapi.Page, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	page, err := c.client.Page.Get(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("get page %s: %w", pageID, err)
	}

	return page, nil
}

// GetDatabase retrieves a database by ID.
func (c *Client) GetDatabase(ctx context.Context, databaseID notionapi.DatabaseID) (*notionapi.Database, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	db, err := c.client.Database.Get(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("get database %s: %w", databaseID, err)
	}

	return db, nil
}

// QueryDatabase queries a database for its items (pages).
func (c *Client) QueryDatabase(
	ctx context.Context,
	databaseID notionapi.DatabaseID,
	startCursor notionapi.Cursor,
	pageSize int,
) (*notionapi.DatabaseQueryResponse, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	req := &notionapi.DatabaseQueryRequest{
		PageSize: pageSize,
	}
	if startCursor != "" {
		req.StartCursor = startCursor
	}

	resp, err := c.client.Database.Query(ctx, databaseID, req)
	if err != nil {
		return nil, fmt.Errorf("query database %s: %w", databaseID, err)
	}

	return resp, nil
}

// GetBlockChildren retrieves child blocks of a block.
func (c *Client) GetBlockChildren(
	ctx context.Context,
	blockID notionapi.BlockID,
	startCursor notionapi.Cursor,
	pageSize int,
) (*notionapi.GetChildrenResponse, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	pagination := &notionapi.Pagination{
		PageSize: pageSize,
	}
	if startCursor != "" {
		pagination.StartCursor = startCursor
	}

	resp, err := c.client.Block.GetChildren(ctx, blockID, pagination)
	if err != nil {
		return nil, fmt.Errorf("get block children %s: %w", blockID, err)
	}

	return resp, nil
}

// GetComments retrieves comments for a block or page.
func (c *Client) GetComments(
	ctx context.Context,
	blockID notionapi.BlockID,
	startCursor notionapi.Cursor,
	pageSize int,
) (*notionapi.CommentQueryResponse, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	pagination := &notionapi.Pagination{
		PageSize: pageSize,
	}
	if startCursor != "" {
		pagination.StartCursor = startCursor
	}

	resp, err := c.client.Comment.Get(ctx, blockID, pagination)
	if err != nil {
		return nil, fmt.Errorf("get comments %s: %w", blockID, err)
	}

	return resp, nil
}

// ListAllUsers lists all users in the workspace.
func (c *Client) ListAllUsers(ctx context.Context) ([]*notionapi.User, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	var users []*notionapi.User
	var cursor notionapi.Cursor

	for {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		pagination := &notionapi.Pagination{
			PageSize: 100,
		}
		if cursor != "" {
			pagination.StartCursor = cursor
		}

		resp, err := c.client.User.List(ctx, pagination)
		if err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}

		for i := range resp.Results {
			users = append(users, &resp.Results[i])
		}

		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}

	return users, nil
}

// ParseLastEditedTime extracts the last edited time from a page or database.
func ParseLastEditedTime(obj notionapi.Object) time.Time {
	switch v := obj.(type) {
	case *notionapi.Page:
		return v.LastEditedTime
	case *notionapi.Database:
		return v.LastEditedTime
	default:
		return time.Time{}
	}
}

// GetObjectID extracts the ID from a page or database.
func GetObjectID(obj notionapi.Object) string {
	switch v := obj.(type) {
	case *notionapi.Page:
		return string(v.ID)
	case *notionapi.Database:
		return string(v.ID)
	default:
		return ""
	}
}

// IsDatabase returns true if the object is a database.
func IsDatabase(obj notionapi.Object) bool {
	_, ok := obj.(*notionapi.Database)
	return ok
}

// IsPage returns true if the object is a page.
func IsPage(obj notionapi.Object) bool {
	_, ok := obj.(*notionapi.Page)
	return ok
}
