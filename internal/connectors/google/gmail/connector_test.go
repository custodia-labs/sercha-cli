package gmail

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// mockTokenProvider implements driven.TokenProvider for testing.
type mockTokenProvider struct {
	token    string
	authID   string
	method   domain.AuthMethod
	isAuthed bool
}

func (m *mockTokenProvider) GetToken(_ context.Context) (string, error) {
	return m.token, nil
}

func (m *mockTokenProvider) AuthorizationID() string {
	return m.authID
}

func (m *mockTokenProvider) AuthMethod() domain.AuthMethod {
	return m.method
}

func (m *mockTokenProvider) IsAuthenticated() bool {
	return m.isAuthed
}

func TestNew(t *testing.T) {
	tp := &mockTokenProvider{token: "test-token", isAuthed: true}
	cfg := DefaultConfig()

	conn := New("source-123", cfg, tp)

	require.NotNil(t, conn)
	assert.Equal(t, "source-123", conn.sourceID)
	assert.Equal(t, cfg, conn.config)
	assert.Equal(t, tp, conn.tokenProvider)
	assert.NotNil(t, conn.rateLimiter)
	assert.False(t, conn.closed)
}

func TestConnector_Type(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)
	assert.Equal(t, "gmail", conn.Type())
}

func TestConnector_SourceID(t *testing.T) {
	conn := New("my-source-id", DefaultConfig(), nil)
	assert.Equal(t, "my-source-id", conn.SourceID())
}

func TestConnector_Capabilities(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)
	caps := conn.Capabilities()

	assert.True(t, caps.SupportsIncremental, "Gmail supports incremental sync via History API")
	assert.False(t, caps.SupportsWatch, "Gmail does not support watch in CLI")
	assert.True(t, caps.SupportsHierarchy, "Gmail has thread hierarchy")
	assert.False(t, caps.SupportsBinary, "Gmail returns RFC 2822 text")
	assert.True(t, caps.RequiresAuth, "Gmail requires OAuth")
	assert.True(t, caps.SupportsValidation, "Gmail supports validation")
	assert.True(t, caps.SupportsCursorReturn, "Gmail returns cursor on sync complete")
	assert.True(t, caps.SupportsPartialSync, "Gmail supports partial sync")
	assert.True(t, caps.SupportsRateLimiting, "Gmail connector handles rate limiting")
	assert.True(t, caps.SupportsPagination, "Gmail uses pagination")
}

func TestConnector_Close(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)

	assert.False(t, conn.closed)

	err := conn.Close()
	require.NoError(t, err)
	assert.True(t, conn.closed)
}

func TestConnector_checkClosed(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)

	// Not closed - should return nil
	err := conn.checkClosed()
	assert.NoError(t, err)

	// Close the connector
	require.NoError(t, conn.Close())

	// Now should return error
	err = conn.checkClosed()
	assert.ErrorIs(t, err, domain.ErrConnectorClosed)
}

func TestConnector_Watch(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)

	changes, err := conn.Watch(context.Background())

	assert.Nil(t, changes)
	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestConnector_FullSync_WhenClosed(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)
	require.NoError(t, conn.Close())

	docs, errs := conn.FullSync(context.Background())

	// Should receive ErrConnectorClosed on error channel
	var receivedErr error
	for err := range errs {
		receivedErr = err
	}
	// Drain docs channel
	for range docs {
	}

	assert.ErrorIs(t, receivedErr, domain.ErrConnectorClosed)
}

func TestConnector_IncrementalSync_WhenClosed(t *testing.T) {
	conn := New("source-123", DefaultConfig(), nil)
	require.NoError(t, conn.Close())

	cursor := NewCursor()
	cursor.HistoryID = 12345
	state := domain.SyncState{Cursor: cursor.Encode()}

	changes, errs := conn.IncrementalSync(context.Background(), state)

	// Should receive ErrConnectorClosed on error channel
	var receivedErr error
	for err := range errs {
		receivedErr = err
	}
	// Drain changes channel
	for range changes {
	}

	assert.ErrorIs(t, receivedErr, domain.ErrConnectorClosed)
}

func TestConnector_IncrementalSync_InvalidCursor(t *testing.T) {
	tp := &mockTokenProvider{token: "test-token", isAuthed: true}
	conn := New("source-123", DefaultConfig(), tp)

	// Empty cursor should fail
	state := domain.SyncState{Cursor: ""}

	changes, errs := conn.IncrementalSync(context.Background(), state)

	var receivedErr error
	for err := range errs {
		receivedErr = err
	}
	for range changes {
	}

	assert.Error(t, receivedErr)
	assert.Contains(t, receivedErr.Error(), "invalid cursor")
}
