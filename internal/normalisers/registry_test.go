package normalisers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// mockNormaliser is a test double for the Normaliser interface.
type mockNormaliser struct {
	mimeTypes      []string
	connectorTypes []string
	priority       int
	normaliseFunc  func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error)
}

func (m *mockNormaliser) SupportedMIMETypes() []string {
	return m.mimeTypes
}

func (m *mockNormaliser) SupportedConnectorTypes() []string {
	return m.connectorTypes
}

func (m *mockNormaliser) Priority() int {
	return m.priority
}

func (m *mockNormaliser) Normalise(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if m.normaliseFunc != nil {
		return m.normaliseFunc(ctx, raw)
	}
	return &driven.NormaliseResult{
		Document: domain.Document{
			ID:       "mock-doc-id",
			SourceID: raw.SourceID,
			URI:      raw.URI,
			Title:    "Mock Document",
			Content:  string(raw.Content),
		},
	}, nil
}

// TestNewRegistry verifies that a new registry is created with default normalisers.
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	require.NotNil(t, registry)
	assert.NotNil(t, registry.normalisers)
	assert.NotNil(t, registry.byMIME)

	// Verify default normalisers are registered
	assert.NotEmpty(t, registry.normalisers, "registry should have default normalisers")
	assert.Equal(t, 12, len(registry.normalisers), "should have 12 default normalisers (docx, eml, html, ics, markdown, pdf, plaintext, github-issue, github-pull, notion-page, notion-database, notion-database-item)")

	// Verify MIME types are indexed
	supportedTypes := registry.SupportedMIMETypes()
	assert.NotEmpty(t, supportedTypes, "registry should support MIME types")

	// Check for expected MIME types from default normalisers
	expectedTypes := map[string]bool{
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/pdf":  true,
		"message/rfc822":   true,
		"text/calendar":    true,
		"text/html":        true,
		"text/markdown":    true,
		"text/x-markdown":  true,
		"text/plain":       true,
		"application/json": true,
	}

	for mimeType := range expectedTypes {
		assert.Contains(t, supportedTypes, mimeType, "should support %s", mimeType)
	}
}

// TestRegistryRegister verifies that normalisers can be registered.
func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	// Create a mock normaliser
	mock := &mockNormaliser{
		mimeTypes: []string{"application/test", "text/test"},
		priority:  100,
	}

	// Register the mock
	registry.Register(mock)

	// Verify it was added to the normalisers list
	assert.Contains(t, registry.normalisers, mock)

	// Verify MIME types are indexed
	supportedTypes := registry.SupportedMIMETypes()
	assert.Contains(t, supportedTypes, "application/test")
	assert.Contains(t, supportedTypes, "text/test")

	// Verify the mock is in the byMIME map
	registry.mu.RLock()
	candidates := registry.byMIME["application/test"]
	registry.mu.RUnlock()
	assert.Contains(t, candidates, mock)
}

// TestRegistryRegisterPrioritySorting verifies that normalisers are sorted by priority.
func TestRegistryRegisterPrioritySorting(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	// Register normalisers with different priorities for the same MIME type
	lowPriority := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  10,
	}
	mediumPriority := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  50,
	}
	highPriority := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  100,
	}

	// Register in non-priority order
	registry.Register(lowPriority)
	registry.Register(highPriority)
	registry.Register(mediumPriority)

	// Verify they are sorted by priority (descending)
	registry.mu.RLock()
	candidates := registry.byMIME["text/test"]
	registry.mu.RUnlock()

	require.Len(t, candidates, 3)
	assert.Equal(t, 100, candidates[0].Priority(), "highest priority should be first")
	assert.Equal(t, 50, candidates[1].Priority(), "medium priority should be second")
	assert.Equal(t, 10, candidates[2].Priority(), "lowest priority should be last")
}

// TestRegistryRegisterMultipleMIMETypes verifies registration with multiple MIME types.
func TestRegistryRegisterMultipleMIMETypes(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	mock := &mockNormaliser{
		mimeTypes: []string{"text/type1", "text/type2", "text/type3"},
		priority:  50,
	}

	registry.Register(mock)

	// Verify the normaliser is registered for all MIME types
	registry.mu.RLock()
	for _, mimeType := range mock.mimeTypes {
		candidates, exists := registry.byMIME[mimeType]
		assert.True(t, exists, "should have entry for %s", mimeType)
		assert.Contains(t, candidates, mock, "should contain mock for %s", mimeType)
	}
	registry.mu.RUnlock()
}

// TestRegistryNormalise verifies successful normalisation with a matching normaliser.
func TestRegistryNormalise(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	expectedResult := &driven.NormaliseResult{
		Document: domain.Document{
			ID:       "test-doc",
			SourceID: "test-source",
			URI:      "/test/path",
			Title:    "Test Document",
			Content:  "test content",
		},
	}

	mock := &mockNormaliser{
		mimeTypes: []string{"application/test"},
		priority:  50,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			return expectedResult, nil
		},
	}

	registry.Register(mock)

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "application/test",
		Content:  []byte("test content"),
	}

	ctx := context.Background()
	result, err := registry.Normalise(ctx, raw)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedResult.Document.ID, result.Document.ID)
	assert.Equal(t, expectedResult.Document.Title, result.Document.Title)
	assert.Equal(t, expectedResult.Document.Content, result.Document.Content)
}

// TestRegistryNormaliseUnsupportedMIMEType verifies error handling for unknown MIME types.
func TestRegistryNormaliseUnsupportedMIMEType(t *testing.T) {
	registry := NewRegistry()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "application/unsupported",
		Content:  []byte("test content"),
	}

	ctx := context.Background()
	result, err := registry.Normalise(ctx, raw)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotImplemented)
	assert.Nil(t, result)
}

// TestRegistryNormaliseSelectsHighestPriority verifies that the highest priority normaliser is selected.
func TestRegistryNormaliseSelectsHighestPriority(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	var calledPriority int

	lowPriority := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  10,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			calledPriority = 10
			return &driven.NormaliseResult{}, nil
		},
	}

	highPriority := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  100,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			calledPriority = 100
			return &driven.NormaliseResult{}, nil
		},
	}

	registry.Register(lowPriority)
	registry.Register(highPriority)

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "text/test",
		Content:  []byte("test content"),
	}

	ctx := context.Background()
	_, err := registry.Normalise(ctx, raw)

	require.NoError(t, err)
	assert.Equal(t, 100, calledPriority, "should call the highest priority normaliser")
}

// TestRegistryNormaliseContextPropagation verifies that context is properly passed.
func TestRegistryNormaliseContextPropagation(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	type contextKey string
	const testKey contextKey = "test"
	expectedValue := "test-value"

	mock := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  50,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			value := ctx.Value(testKey)
			assert.Equal(t, expectedValue, value, "context value should be propagated")
			return &driven.NormaliseResult{}, nil
		},
	}

	registry.Register(mock)

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "text/test",
		Content:  []byte("test content"),
	}

	ctx := context.WithValue(context.Background(), testKey, expectedValue)
	_, err := registry.Normalise(ctx, raw)

	require.NoError(t, err)
}

// TestRegistryNormaliseErrorPropagation verifies that errors from normalisers are propagated.
func TestRegistryNormaliseErrorPropagation(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	expectedError := domain.ErrInvalidInput

	mock := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  50,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			return nil, expectedError
		},
	}

	registry.Register(mock)

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "text/test",
		Content:  []byte("test content"),
	}

	ctx := context.Background()
	result, err := registry.Normalise(ctx, raw)

	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedError)
	assert.Nil(t, result)
}

// TestRegistrySupportedMIMETypes verifies the SupportedMIMETypes method.
func TestRegistrySupportedMIMETypes(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	mock1 := &mockNormaliser{
		mimeTypes: []string{"text/type1", "text/type2"},
		priority:  50,
	}

	mock2 := &mockNormaliser{
		mimeTypes: []string{"text/type3", "text/type1"}, // type1 is shared
		priority:  60,
	}

	registry.Register(mock1)
	registry.Register(mock2)

	supportedTypes := registry.SupportedMIMETypes()

	// Should contain all unique MIME types
	assert.Len(t, supportedTypes, 3, "should have 3 unique MIME types")
	assert.Contains(t, supportedTypes, "text/type1")
	assert.Contains(t, supportedTypes, "text/type2")
	assert.Contains(t, supportedTypes, "text/type3")
}

// TestRegistrySupportedMIMETypesEmpty verifies behavior with no normalisers.
func TestRegistrySupportedMIMETypesEmpty(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	supportedTypes := registry.SupportedMIMETypes()

	assert.Empty(t, supportedTypes, "should have no supported MIME types")
}

// TestRegistryConcurrentAccess verifies thread-safe concurrent operations.
func TestRegistryConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	// Add some test normalisers
	mock := &mockNormaliser{
		mimeTypes: []string{"text/concurrent"},
		priority:  50,
	}
	registry.Register(mock)

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "text/markdown",
		Content:  []byte("test content"),
	}

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 100)

	for i := 0; i < 50; i++ {
		go func() {
			_, _ = registry.Normalise(ctx, raw)
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		go func() {
			_ = registry.SupportedMIMETypes()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// If we reach here without deadlock or race conditions, the test passes
	assert.True(t, true, "concurrent access should not cause issues")
}

// TestRegistryNormaliseWithMetadata verifies normalisation with document metadata.
func TestRegistryNormaliseWithMetadata(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	var receivedMetadata map[string]any

	mock := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  50,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			receivedMetadata = raw.Metadata
			return &driven.NormaliseResult{}, nil
		},
	}

	registry.Register(mock)

	expectedMetadata := map[string]any{
		"author": "test-author",
		"tags":   []string{"tag1", "tag2"},
		"count":  42,
	}

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/path",
		MIMEType: "text/test",
		Content:  []byte("test content"),
		Metadata: expectedMetadata,
	}

	ctx := context.Background()
	_, err := registry.Normalise(ctx, raw)

	require.NoError(t, err)
	assert.Equal(t, expectedMetadata, receivedMetadata, "metadata should be passed to normaliser")
}

// TestRegistryNormaliseWithParentURI verifies normalisation with hierarchical documents.
func TestRegistryNormaliseWithParentURI(t *testing.T) {
	registry := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}

	var receivedParentURI *string

	mock := &mockNormaliser{
		mimeTypes: []string{"text/test"},
		priority:  50,
		normaliseFunc: func(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
			receivedParentURI = raw.ParentURI
			return &driven.NormaliseResult{}, nil
		},
	}

	registry.Register(mock)

	parentURI := "/parent/path"
	raw := &domain.RawDocument{
		SourceID:  "test-source",
		URI:       "/test/path",
		MIMEType:  "text/test",
		Content:   []byte("test content"),
		ParentURI: &parentURI,
	}

	ctx := context.Background()
	_, err := registry.Normalise(ctx, raw)

	require.NoError(t, err)
	require.NotNil(t, receivedParentURI)
	assert.Equal(t, parentURI, *receivedParentURI, "parent URI should be passed to normaliser")
}

// TestRegistryInterfaceCompliance verifies Registry implements NormaliserRegistry.
func TestRegistryInterfaceCompliance(t *testing.T) {
	var _ driven.NormaliserRegistry = (*Registry)(nil)
	// This test will fail at compile time if Registry doesn't implement the interface
}
