package services

import (
	"context"
	"errors"
	stdsync "sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/memory"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// --- Mock implementations for sync testing ---
// Note: These are prefixed with "sync" to avoid conflicts with search_test.go mocks

// syncMockConnector implements driven.Connector for testing.
type syncMockConnector struct {
	sourceID     string
	connType     string
	capabilities driven.ConnectorCapabilities
	fullSyncDocs []domain.RawDocument
	fullSyncErr  error
	incSyncDocs  []domain.RawDocumentChange
	incSyncErr   error
	closed       bool
}

func (m *syncMockConnector) Type() string     { return m.connType }
func (m *syncMockConnector) SourceID() string { return m.sourceID }
func (m *syncMockConnector) Capabilities() driven.ConnectorCapabilities {
	return m.capabilities
}

func (m *syncMockConnector) FullSync(ctx context.Context) (<-chan domain.RawDocument, <-chan error) {
	docs := make(chan domain.RawDocument)
	errs := make(chan error, 1)

	go func() {
		defer close(docs)
		defer close(errs)

		if m.fullSyncErr != nil {
			errs <- m.fullSyncErr
			return
		}

		for _, doc := range m.fullSyncDocs {
			select {
			case <-ctx.Done():
				return
			case docs <- doc:
			}
		}
	}()

	return docs, errs
}

func (m *syncMockConnector) IncrementalSync(ctx context.Context, _ domain.SyncState) (<-chan domain.RawDocumentChange, <-chan error) {
	changes := make(chan domain.RawDocumentChange)
	errs := make(chan error, 1)

	go func() {
		defer close(changes)
		defer close(errs)

		if m.incSyncErr != nil {
			errs <- m.incSyncErr
			return
		}

		for _, change := range m.incSyncDocs {
			select {
			case <-ctx.Done():
				return
			case changes <- change:
			}
		}
	}()

	return changes, errs
}

func (m *syncMockConnector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	return nil, errors.New("watch not implemented")
}

func (m *syncMockConnector) Validate(_ context.Context) error {
	return nil
}

func (m *syncMockConnector) Close() error {
	m.closed = true
	return nil
}

func (m *syncMockConnector) GetAccountIdentifier(_ context.Context, _ string) (string, error) {
	return "", nil
}

// syncMockConnectorFactory implements driven.ConnectorFactory.
type syncMockConnectorFactory struct {
	connectors map[string]*syncMockConnector
	createErr  error
}

func newSyncMockConnectorFactory() *syncMockConnectorFactory {
	return &syncMockConnectorFactory{
		connectors: make(map[string]*syncMockConnector),
	}
}

func (f *syncMockConnectorFactory) Create(_ context.Context, source domain.Source) (driven.Connector, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if conn, ok := f.connectors[source.ID]; ok {
		return conn, nil
	}
	return nil, errors.New("no connector configured for source")
}

func (f *syncMockConnectorFactory) Register(_ string, _ driven.ConnectorBuilder) {}

func (f *syncMockConnectorFactory) SupportedTypes() []string {
	return []string{"mock"}
}

func (f *syncMockConnectorFactory) GetDefaultOAuthConfig(_ string) *driven.OAuthDefaults {
	return nil
}

func (f *syncMockConnectorFactory) GetSetupHint(_ string) string {
	return ""
}

func (f *syncMockConnectorFactory) SupportsOAuth(_ string) bool {
	return false
}

func (f *syncMockConnectorFactory) BuildAuthURL(_ string, _ *domain.AuthProvider, _, _, _ string) (string, error) {
	return "", nil
}

func (f *syncMockConnectorFactory) ExchangeCode(_ context.Context, _ string, _ *domain.AuthProvider, _, _, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

func (f *syncMockConnectorFactory) RefreshToken(_ context.Context, _ string, _ *domain.AuthProvider, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

func (f *syncMockConnectorFactory) GetUserInfo(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// syncMockNormaliserRegistry implements driven.NormaliserRegistry.
type syncMockNormaliserRegistry struct {
	normaliseResult *driven.NormaliseResult
	normaliseErr    error
}

func (r *syncMockNormaliserRegistry) Register(_ driven.Normaliser) {}

func (r *syncMockNormaliserRegistry) SupportedMIMETypes() []string {
	return []string{"text/plain"}
}

func (r *syncMockNormaliserRegistry) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if r.normaliseErr != nil {
		return nil, r.normaliseErr
	}
	if r.normaliseResult != nil {
		return r.normaliseResult, nil
	}

	// Default: create a simple document from the raw document
	// Use SourceID + URI to create unique IDs across sources
	docID := raw.SourceID + "-doc-" + raw.URI
	doc := domain.Document{
		ID:        docID,
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     raw.URI, // Use URI as title
		Content:   string(raw.Content),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// syncMockPostProcessorPipeline implements driven.PostProcessorPipeline.
type syncMockPostProcessorPipeline struct{}

func (p *syncMockPostProcessorPipeline) Process(_ context.Context, doc *domain.Document) ([]domain.Chunk, error) {
	// Create a single chunk from the document content
	chunk := domain.Chunk{
		ID:         doc.SourceID + "-chunk-" + doc.URI,
		DocumentID: doc.ID,
		Content:    doc.Content,
		Position:   0,
	}
	return []domain.Chunk{chunk}, nil
}

// syncMockSearchEngine implements driven.SearchEngine with state tracking.
type syncMockSearchEngine struct {
	indexed map[string]domain.Chunk
	mu      stdsync.Mutex
}

func newSyncMockSearchEngine() *syncMockSearchEngine {
	return &syncMockSearchEngine{
		indexed: make(map[string]domain.Chunk),
	}
}

func (e *syncMockSearchEngine) Index(_ context.Context, chunk domain.Chunk) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.indexed[chunk.ID] = chunk
	return nil
}

func (e *syncMockSearchEngine) Search(_ context.Context, _ string, _ int) ([]driven.SearchHit, error) {
	return nil, nil
}

func (e *syncMockSearchEngine) Delete(_ context.Context, chunkID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.indexed, chunkID)
	return nil
}

func (e *syncMockSearchEngine) Close() error { return nil }

// syncMockVectorIndex implements driven.VectorIndex with state tracking.
type syncMockVectorIndex struct {
	vectors map[string][]float32
	mu      stdsync.Mutex
}

func newSyncMockVectorIndex() *syncMockVectorIndex {
	return &syncMockVectorIndex{
		vectors: make(map[string][]float32),
	}
}

func (v *syncMockVectorIndex) Add(_ context.Context, id string, embedding []float32) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.vectors[id] = embedding
	return nil
}

func (v *syncMockVectorIndex) Search(_ context.Context, _ []float32, _ int) ([]driven.VectorHit, error) {
	return nil, nil
}

func (v *syncMockVectorIndex) Delete(_ context.Context, id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.vectors, id)
	return nil
}

func (v *syncMockVectorIndex) Close() error { return nil }

// syncMockEmbeddingService implements driven.EmbeddingService.
type syncMockEmbeddingService struct {
	embedding []float32
	err       error
}

func (e *syncMockEmbeddingService) Embed(_ context.Context, _ string) ([]float32, error) {
	if e.err != nil {
		return nil, e.err
	}
	if e.embedding != nil {
		return e.embedding, nil
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (e *syncMockEmbeddingService) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		emb, err := e.Embed(context.Background(), texts[i])
		if err != nil {
			return nil, err
		}
		result[i] = emb
	}
	return result, nil
}

func (e *syncMockEmbeddingService) Dimensions() int              { return 3 }
func (e *syncMockEmbeddingService) ModelName() string            { return "mock" }
func (e *syncMockEmbeddingService) Ping(_ context.Context) error { return nil }
func (e *syncMockEmbeddingService) Close() error                 { return nil }

// --- Tests ---

func TestNewSyncOrchestrator(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		nil, nil, nil, nil, nil, nil,
	)

	require.NotNil(t, orchestrator)
	assert.NotNil(t, orchestrator.sourceStore)
	assert.NotNil(t, orchestrator.syncStore)
	assert.NotNil(t, orchestrator.docStore)
	assert.NotNil(t, orchestrator.exclusionStore)
	assert.NotNil(t, orchestrator.activeSyncs)
}

func TestSyncOrchestrator_Sync_SourceNotFound(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		nil, nil, nil, nil, nil, nil,
	)

	err := orchestrator.Sync(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get source")
}

func TestSyncOrchestrator_Sync_ConnectorFactoryMissing(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()

	ctx := context.Background()

	// Add source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "filesystem"}
	require.NoError(t, sourceStore.Save(ctx, source))

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		nil, // no factory
		nil, nil, nil, nil, nil,
	)

	err := orchestrator.Sync(ctx, "src-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create connector")
}

func TestSyncOrchestrator_Sync_FullSync_Success(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Setup connector with documents
	factory.connectors["src-1"] = &syncMockConnector{
		sourceID: "src-1",
		connType: "mock",
		fullSyncDocs: []domain.RawDocument{
			{SourceID: "src-1", URI: "file1.txt", MIMEType: "text/plain", Content: []byte("content 1")},
			{SourceID: "src-1", URI: "file2.txt", MIMEType: "text/plain", Content: []byte("content 2")},
		},
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.Sync(ctx, "src-1")

	require.NoError(t, err)

	// Verify documents were saved
	docs, err := docStore.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs, 2)

	// Verify sync state was saved
	state, err := syncStore.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "src-1", state.SourceID)
	assert.False(t, state.LastSync.IsZero())

	// Verify chunks were indexed
	assert.Len(t, searchEngine.indexed, 2)
}

func TestSyncOrchestrator_Sync_WithExclusions(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Setup connector with documents
	factory.connectors["src-1"] = &syncMockConnector{
		sourceID: "src-1",
		connType: "mock",
		fullSyncDocs: []domain.RawDocument{
			{SourceID: "src-1", URI: "file1.txt", MIMEType: "text/plain", Content: []byte("content 1")},
			{SourceID: "src-1", URI: "file2.txt", MIMEType: "text/plain", Content: []byte("content 2")},
			{SourceID: "src-1", URI: "excluded.txt", MIMEType: "text/plain", Content: []byte("excluded")},
		},
	}

	// Add exclusion for one file
	exclusion := &domain.Exclusion{
		ID:         "exc-1",
		SourceID:   "src-1",
		DocumentID: "doc-excluded",
		URI:        "excluded.txt",
		Reason:     "test exclusion",
	}
	require.NoError(t, exclusionStore.Add(ctx, exclusion))

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.Sync(ctx, "src-1")

	require.NoError(t, err)

	// Verify only non-excluded documents were saved
	docs, err := docStore.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs, 2) // Only 2 docs, excluded one skipped
}

func TestSyncOrchestrator_Sync_WithEmbeddings(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()
	vectorIndex := newSyncMockVectorIndex()
	embeddingService := &syncMockEmbeddingService{
		embedding: []float32{0.5, 0.5, 0.5},
	}

	ctx := context.Background()

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Setup connector
	factory.connectors["src-1"] = &syncMockConnector{
		sourceID: "src-1",
		connType: "mock",
		fullSyncDocs: []domain.RawDocument{
			{SourceID: "src-1", URI: "file1.txt", MIMEType: "text/plain", Content: []byte("content 1")},
		},
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, vectorIndex, embeddingService,
	)

	err := orchestrator.Sync(ctx, "src-1")

	require.NoError(t, err)

	// Verify vectors were indexed
	assert.Len(t, vectorIndex.vectors, 1)
}

func TestSyncOrchestrator_Sync_IncrementalSync(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Setup existing sync state with cursor (triggers incremental sync)
	existingState := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-123",
		LastSync: time.Now().Add(-time.Hour),
	}
	require.NoError(t, syncStore.Save(ctx, existingState))

	// Setup connector with incremental support
	factory.connectors["src-1"] = &syncMockConnector{
		sourceID: "src-1",
		connType: "mock",
		capabilities: driven.ConnectorCapabilities{
			SupportsIncremental: true,
		},
		incSyncDocs: []domain.RawDocumentChange{
			{
				Type:     domain.ChangeCreated,
				Document: domain.RawDocument{SourceID: "src-1", URI: "new.txt", MIMEType: "text/plain", Content: []byte("new content")},
			},
		},
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.Sync(ctx, "src-1")

	require.NoError(t, err)

	// Verify new document was saved
	docs, err := docStore.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestSyncOrchestrator_SyncAll_Success(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup multiple sources
	sources := []domain.Source{
		{ID: "src-1", Name: "Source 1", Type: "mock"},
		{ID: "src-2", Name: "Source 2", Type: "mock"},
	}

	for _, src := range sources {
		require.NoError(t, sourceStore.Save(ctx, src))
		factory.connectors[src.ID] = &syncMockConnector{
			sourceID: src.ID,
			connType: "mock",
			fullSyncDocs: []domain.RawDocument{
				{SourceID: src.ID, URI: "file.txt", MIMEType: "text/plain", Content: []byte("content")},
			},
		}
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.SyncAll(ctx)

	require.NoError(t, err)

	// Verify both sources were synced
	for _, src := range sources {
		docs, err := docStore.ListDocuments(ctx, src.ID)
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	}
}

func TestSyncOrchestrator_SyncAll_NoSources(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		nil, nil, nil, nil, nil, nil,
	)

	err := orchestrator.SyncAll(context.Background())

	// No sources means nothing to sync - should succeed
	assert.NoError(t, err)
}

func TestSyncOrchestrator_SyncAll_PartialFailure(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup sources - one will fail
	require.NoError(t, sourceStore.Save(ctx, domain.Source{ID: "src-1", Name: "Good", Type: "mock"}))
	require.NoError(t, sourceStore.Save(ctx, domain.Source{ID: "src-2", Name: "Bad", Type: "mock"}))

	factory.connectors["src-1"] = &syncMockConnector{
		sourceID: "src-1",
		connType: "mock",
		fullSyncDocs: []domain.RawDocument{
			{SourceID: "src-1", URI: "file.txt", MIMEType: "text/plain", Content: []byte("content")},
		},
	}
	factory.connectors["src-2"] = &syncMockConnector{
		sourceID:    "src-2",
		connType:    "mock",
		fullSyncErr: errors.New("connector error"),
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.SyncAll(ctx)

	// Should return error for failed source
	require.Error(t, err)
	assert.Contains(t, err.Error(), "src-2")

	// But first source should have succeeded
	docs, _ := docStore.ListDocuments(ctx, "src-1")
	assert.Len(t, docs, 1)
}

func TestSyncOrchestrator_Status_NotRunning(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		nil, nil, nil, nil, nil, nil,
	)

	status, err := orchestrator.Status(context.Background(), "src-1")

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "src-1", status.SourceID)
	assert.False(t, status.Running)
}

func TestSyncOrchestrator_Status_WhileRunning(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		nil, nil, nil, nil, nil, nil,
	)

	// Manually set status to simulate running
	orchestrator.mu.Lock()
	orchestrator.activeSyncs["src-1"] = &driving.SyncStatus{
		SourceID:           "src-1",
		Running:            true,
		DocumentsProcessed: 5,
		ErrorCount:         1,
	}
	orchestrator.mu.Unlock()

	status, err := orchestrator.Status(context.Background(), "src-1")

	require.NoError(t, err)
	assert.True(t, status.Running)
	assert.Equal(t, 5, status.DocumentsProcessed)
	assert.Equal(t, 1, status.ErrorCount)
}

func TestSyncOrchestrator_Sync_ConnectorClosed(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Setup connector
	connector := &syncMockConnector{
		sourceID:     "src-1",
		connType:     "mock",
		fullSyncDocs: []domain.RawDocument{},
	}
	factory.connectors["src-1"] = connector

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.Sync(ctx, "src-1")

	require.NoError(t, err)
	assert.True(t, connector.closed, "connector should be closed after sync")
}

func TestSyncOrchestrator_Sync_ContextCancellation(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx, cancel := context.WithCancel(context.Background())

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Setup connector with many documents
	docs := make([]domain.RawDocument, 100)
	for i := range docs {
		docs[i] = domain.RawDocument{
			SourceID: "src-1",
			URI:      "file" + string(rune(i)) + ".txt",
			MIMEType: "text/plain",
			Content:  []byte("content"),
		}
	}
	factory.connectors["src-1"] = &syncMockConnector{
		sourceID:     "src-1",
		connType:     "mock",
		fullSyncDocs: docs,
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	// Cancel immediately
	cancel()

	err := orchestrator.Sync(ctx, "src-1")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestSyncOrchestrator_ProcessChanges_Delete(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	factory := newSyncMockConnectorFactory()
	registry := &syncMockNormaliserRegistry{}
	searchEngine := newSyncMockSearchEngine()

	ctx := context.Background()

	// Setup source
	source := domain.Source{ID: "src-1", Name: "Test", Type: "mock"}
	require.NoError(t, sourceStore.Save(ctx, source))

	// Add existing document
	existingDoc := domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		URI:      "existing.txt",
		Title:    "Existing",
	}
	require.NoError(t, docStore.SaveDocument(ctx, &existingDoc))
	chunk := domain.Chunk{ID: "chunk-1", DocumentID: "doc-1", Content: "content"}
	require.NoError(t, docStore.SaveChunks(ctx, []domain.Chunk{chunk}))
	require.NoError(t, searchEngine.Index(ctx, chunk))

	// Setup existing sync state
	existingState := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-123",
		LastSync: time.Now().Add(-time.Hour),
	}
	require.NoError(t, syncStore.Save(ctx, existingState))

	// Setup connector with delete change
	factory.connectors["src-1"] = &syncMockConnector{
		sourceID: "src-1",
		connType: "mock",
		capabilities: driven.ConnectorCapabilities{
			SupportsIncremental: true,
		},
		incSyncDocs: []domain.RawDocumentChange{
			{
				Type:     domain.ChangeDeleted,
				Document: domain.RawDocument{SourceID: "src-1", URI: "existing.txt"},
			},
		},
	}

	orchestrator := NewSyncOrchestrator(
		sourceStore, syncStore, docStore, exclusionStore,
		factory, registry, &syncMockPostProcessorPipeline{}, searchEngine, nil, nil,
	)

	err := orchestrator.Sync(ctx, "src-1")

	require.NoError(t, err)

	// Verify document was deleted
	docs, err := docStore.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs, 0)

	// Verify search index was cleaned
	assert.Len(t, searchEngine.indexed, 0)
}
