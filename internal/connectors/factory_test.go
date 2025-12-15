package connectors

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// mockTokenProviderFactory implements TokenProviderFactory for testing.
type mockTokenProviderFactory struct{}

func (f *mockTokenProviderFactory) CreateTokenProvider(_ context.Context, _ *domain.Source) (driven.TokenProvider, error) {
	return &mockTokenProvider{}, nil
}

// mockTokenProvider implements driven.TokenProvider for testing.
type mockTokenProvider struct{}

func (p *mockTokenProvider) GetToken(_ context.Context) (string, error) { return "", nil }
func (p *mockTokenProvider) AuthorizationID() string                    { return "local" }
func (p *mockTokenProvider) AuthMethod() domain.AuthMethod              { return domain.AuthMethodNone }
func (p *mockTokenProvider) IsAuthenticated() bool                      { return true }

// mockConnector implements the driven.Connector interface for testing.
type mockConnector struct {
	sourceID string
	connType string
}

func (m *mockConnector) Type() string {
	return m.connType
}

func (m *mockConnector) SourceID() string {
	return m.sourceID
}

func (m *mockConnector) Capabilities() driven.ConnectorCapabilities {
	return driven.ConnectorCapabilities{}
}

func (m *mockConnector) Validate(_ context.Context) error {
	return nil
}

func (m *mockConnector) FullSync(_ context.Context) (<-chan domain.RawDocument, <-chan error) {
	docs := make(chan domain.RawDocument)
	errs := make(chan error)
	close(docs)
	close(errs)
	return docs, errs
}

func (m *mockConnector) IncrementalSync(_ context.Context, _ domain.SyncState) (<-chan domain.RawDocumentChange, <-chan error) {
	changes := make(chan domain.RawDocumentChange)
	errs := make(chan error)
	close(changes)
	close(errs)
	return changes, errs
}

func (m *mockConnector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	changes := make(chan domain.RawDocumentChange)
	close(changes)
	return changes, nil
}

func (m *mockConnector) Close() error {
	return nil
}

func (m *mockConnector) GetAccountIdentifier(_ context.Context, _ string) (string, error) {
	return "", nil
}

func TestNewFactory(t *testing.T) {
	t.Run("creates factory with default builders", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		require.NotNil(t, factory)
		assert.NotNil(t, factory.builders)

		// Verify filesystem connector is registered by default
		supportedTypes := factory.SupportedTypes()
		assert.Contains(t, supportedTypes, "filesystem")
	})

	t.Run("factory implements ConnectorFactory interface", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})
		var _ driven.ConnectorFactory = factory
	})
}

func TestFactory_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates filesystem connector with valid config", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		source := domain.Source{
			ID:   "test-source",
			Type: "filesystem",
			Name: "Test Source",
			Config: map[string]string{
				"path": "/tmp/test",
			},
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.NoError(t, err)
		require.NotNil(t, connector)
		assert.Equal(t, "filesystem", connector.Type())
		assert.Equal(t, "test-source", connector.SourceID())
	})

	t.Run("returns error for unknown connector type", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		source := domain.Source{
			ID:   "test-source",
			Type: "unknown-type",
			Name: "Test Source",
			Config: map[string]string{
				"path": "/tmp/test",
			},
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.Error(t, err)
		assert.Nil(t, connector)
		assert.ErrorIs(t, err, domain.ErrUnsupportedType)
		assert.Contains(t, err.Error(), "unknown-type")
	})

	t.Run("returns error when filesystem path config is missing", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		source := domain.Source{
			ID:              "test-source",
			Type:            "filesystem",
			Name:            "Test Source",
			Config:          map[string]string{}, // Missing 'path'
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.Error(t, err)
		assert.Nil(t, connector)
		assert.Contains(t, err.Error(), "filesystem source requires 'path' config")
	})

	t.Run("creates connector with custom builder", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		// Register a custom builder
		factory.Register("custom", func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return &mockConnector{
				sourceID: source.ID,
				connType: "custom",
			}, nil
		})

		source := domain.Source{
			ID:              "custom-source",
			Type:            "custom",
			Name:            "Custom Source",
			Config:          map[string]string{},
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.NoError(t, err)
		require.NotNil(t, connector)
		assert.Equal(t, "custom", connector.Type())
		assert.Equal(t, "custom-source", connector.SourceID())
	})

	t.Run("builder error is propagated", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		expectedErr := fmt.Errorf("custom builder error")
		factory.Register("error-builder", func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return nil, expectedErr
		})

		source := domain.Source{
			ID:              "error-source",
			Type:            "error-builder",
			Name:            "Error Source",
			Config:          map[string]string{},
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.Error(t, err)
		assert.Nil(t, connector)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("handles nil config map", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		source := domain.Source{
			ID:              "test-source",
			Type:            "filesystem",
			Name:            "Test Source",
			Config:          nil, // nil config
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.Error(t, err)
		assert.Nil(t, connector)
		assert.Contains(t, err.Error(), "filesystem source requires 'path' config")
	})
}

func TestFactory_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("registers new connector type", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		builder := func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return &mockConnector{
				sourceID: source.ID,
				connType: "gmail",
			}, nil
		}

		factory.Register("gmail", builder)

		supportedTypes := factory.SupportedTypes()
		assert.Contains(t, supportedTypes, "gmail")
	})

	t.Run("overwrites existing connector type", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		firstBuilder := func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return &mockConnector{
				sourceID: source.ID,
				connType: "first",
			}, nil
		}

		secondBuilder := func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return &mockConnector{
				sourceID: source.ID,
				connType: "second",
			}, nil
		}

		factory.Register("test-type", firstBuilder)
		factory.Register("test-type", secondBuilder)

		source := domain.Source{
			ID:              "test-source",
			Type:            "test-type",
			Name:            "Test Source",
			Config:          map[string]string{},
			AuthorizationID: "local",
		}

		connector, err := factory.Create(ctx, source)

		require.NoError(t, err)
		require.NotNil(t, connector)
		// Should use the second builder
		assert.Equal(t, "second", connector.Type())
	})

	t.Run("thread-safe registration", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})
		var wg sync.WaitGroup

		// Register multiple connectors concurrently
		connectorTypes := []string{"type1", "type2", "type3", "type4", "type5"}
		for _, connType := range connectorTypes {
			wg.Add(1)
			go func(ct string) {
				defer wg.Done()
				factory.Register(ct, func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
					return &mockConnector{
						sourceID: source.ID,
						connType: ct,
					}, nil
				})
			}(connType)
		}

		wg.Wait()

		supportedTypes := factory.SupportedTypes()
		for _, ct := range connectorTypes {
			assert.Contains(t, supportedTypes, ct)
		}
	})
}

func TestFactory_SupportedTypes(t *testing.T) {
	t.Run("returns all registered types", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})

		supportedTypes := factory.SupportedTypes()

		// All default connectors: filesystem, github, google-drive, gmail, google-calendar,
		// outlook, onedrive, microsoft-calendar, dropbox, notion
		assert.Len(t, supportedTypes, 10)
		assert.Contains(t, supportedTypes, "filesystem")
		assert.Contains(t, supportedTypes, "github")
		assert.Contains(t, supportedTypes, "google-drive")
		assert.Contains(t, supportedTypes, "gmail")
		assert.Contains(t, supportedTypes, "google-calendar")
		assert.Contains(t, supportedTypes, "outlook")
		assert.Contains(t, supportedTypes, "onedrive")
		assert.Contains(t, supportedTypes, "microsoft-calendar")
		assert.Contains(t, supportedTypes, "dropbox")
		assert.Contains(t, supportedTypes, "notion")
	})

	t.Run("returns empty slice for factory with no builders", func(t *testing.T) {
		factory := &Factory{
			builders: make(map[string]driven.ConnectorBuilder),
		}

		supportedTypes := factory.SupportedTypes()

		assert.NotNil(t, supportedTypes)
		assert.Empty(t, supportedTypes)
	})

	t.Run("thread-safe reading", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})
		var wg sync.WaitGroup

		// Register a few types
		factory.Register("type1", func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return &mockConnector{}, nil
		})

		// Read supported types concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				types := factory.SupportedTypes()
				assert.NotEmpty(t, types)
			}()
		}

		wg.Wait()
	})

	t.Run("returns a copy not the internal map", func(t *testing.T) {
		factory := NewFactory(&mockTokenProviderFactory{})
		factory.Register("type1", func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
			return &mockConnector{}, nil
		})

		types1 := factory.SupportedTypes()
		types2 := factory.SupportedTypes()

		// Should contain same elements (order may vary due to map iteration)
		assert.ElementsMatch(t, types1, types2)
		assert.NotSame(t, &types1, &types2)
	})
}

func TestFactory_ConcurrentCreateAndRegister(t *testing.T) {
	t.Run("concurrent create and register operations", func(t *testing.T) {
		ctx := context.Background()
		factory := NewFactory(&mockTokenProviderFactory{})
		var wg sync.WaitGroup

		// Register connectors concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			connType := fmt.Sprintf("type%d", i)
			go func(ct string) {
				defer wg.Done()
				factory.Register(ct, func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
					return &mockConnector{
						sourceID: source.ID,
						connType: ct,
					}, nil
				})
			}(connType)
		}

		// Create connectors concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				source := domain.Source{
					ID:   fmt.Sprintf("source-%d", idx),
					Type: "filesystem",
					Name: fmt.Sprintf("Source %d", idx),
					Config: map[string]string{
						"path": fmt.Sprintf("/tmp/test-%d", idx),
					},
					AuthorizationID: "local",
				}
				_, _ = factory.Create(ctx, source)
			}(i)
		}

		// Get supported types concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = factory.SupportedTypes()
			}()
		}

		wg.Wait()

		// Verify all types were registered
		supportedTypes := factory.SupportedTypes()
		assert.GreaterOrEqual(t, len(supportedTypes), 6) // At least filesystem + 5 custom types
	})
}
