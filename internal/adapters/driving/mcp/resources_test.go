package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestExtractSourceID(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "valid source documents URI",
			uri:      "sercha://sources/src-123/documents",
			expected: "src-123",
		},
		{
			name:     "invalid prefix",
			uri:      "file://sources/src-123/documents",
			expected: "",
		},
		{
			name:     "missing documents suffix",
			uri:      "sercha://sources/src-123",
			expected: "",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSourceID(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDocumentID(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "valid document URI",
			uri:      "sercha://documents/doc-456",
			expected: "doc-456",
		},
		{
			name:     "invalid prefix",
			uri:      "file://documents/doc-456",
			expected: "",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDocumentID(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper to create a ReadResourceRequest with the given URI.
func makeReadResourceRequest(uri string) *mcp.ReadResourceRequest {
	return &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: uri,
		},
	}
}

func TestServer_handleSourcesResource(t *testing.T) {
	ctx := context.Background()

	t.Run("nil source service returns empty list", func(t *testing.T) {
		ports := &Ports{Search: &mockSearchService{}}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources")
		result, err := server.handleSourcesResource(ctx, req)

		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Equal(t, "[]", result.Contents[0].Text)
	})

	t.Run("returns sources successfully", func(t *testing.T) {
		mockSource := &mockSourceService{
			sources: []domain.Source{
				{
					ID:     "src-1",
					Name:   "My Docs",
					Type:   "filesystem",
					Config: map[string]string{"path": "/home/docs"},
				},
			},
		}

		ports := &Ports{Search: &mockSearchService{}, Source: mockSource}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources")
		result, err := server.handleSourcesResource(ctx, req)

		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Contains(t, result.Contents[0].Text, "src-1")
		assert.Contains(t, result.Contents[0].Text, "My Docs")
		assert.Contains(t, result.Contents[0].Text, "/home/docs")
	})

	t.Run("returns error on list failure", func(t *testing.T) {
		mockSource := &mockSourceService{
			err: errors.New("database error"),
		}

		ports := &Ports{Search: &mockSearchService{}, Source: mockSource}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources")
		_, err = server.handleSourcesResource(ctx, req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing sources")
	})

	t.Run("handles source without path config", func(t *testing.T) {
		mockSource := &mockSourceService{
			sources: []domain.Source{
				{
					ID:     "src-2",
					Name:   "API Source",
					Type:   "api",
					Config: map[string]string{"url": "https://api.example.com"},
				},
			},
		}

		ports := &Ports{Search: &mockSearchService{}, Source: mockSource}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources")
		result, err := server.handleSourcesResource(ctx, req)

		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		// URI should be empty since there's no "path" in config
		assert.Contains(t, result.Contents[0].Text, `"uri": ""`)
	})
}

func TestServer_handleDocumentsResource(t *testing.T) {
	ctx := context.Background()

	t.Run("nil document service returns not found", func(t *testing.T) {
		ports := &Ports{Search: &mockSearchService{}}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources/src-123/documents")
		_, err = server.handleDocumentsResource(ctx, req)

		require.Error(t, err)
	})

	t.Run("invalid URI returns not found", func(t *testing.T) {
		mockDoc := &mockDocumentService{}
		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://invalid/uri")
		_, err = server.handleDocumentsResource(ctx, req)

		require.Error(t, err)
	})

	t.Run("returns documents successfully", func(t *testing.T) {
		mockDoc := &mockDocumentService{
			documents: []domain.Document{
				{ID: "doc-1", Title: "README.md", URI: "/path/to/readme.md"},
				{ID: "doc-2", Title: "Guide.md", URI: "/path/to/guide.md"},
			},
		}

		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources/src-123/documents")
		result, err := server.handleDocumentsResource(ctx, req)

		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Contains(t, result.Contents[0].Text, "doc-1")
		assert.Contains(t, result.Contents[0].Text, "README.md")
		assert.Contains(t, result.Contents[0].Text, "doc-2")
	})

	t.Run("returns error on list failure", func(t *testing.T) {
		mockDoc := &mockDocumentService{
			err: errors.New("storage error"),
		}

		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources/src-123/documents")
		_, err = server.handleDocumentsResource(ctx, req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing documents")
	})

	t.Run("handles empty document list", func(t *testing.T) {
		mockDoc := &mockDocumentService{
			documents: []domain.Document{},
		}

		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://sources/src-123/documents")
		result, err := server.handleDocumentsResource(ctx, req)

		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Equal(t, "[]", result.Contents[0].Text)
	})
}

func TestServer_handleDocumentContentResource(t *testing.T) {
	ctx := context.Background()

	t.Run("nil document service returns not found", func(t *testing.T) {
		ports := &Ports{Search: &mockSearchService{}}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://documents/doc-123")
		_, err = server.handleDocumentContentResource(ctx, req)

		require.Error(t, err)
	})

	t.Run("invalid URI returns not found", func(t *testing.T) {
		mockDoc := &mockDocumentService{}
		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://invalid/uri")
		_, err = server.handleDocumentContentResource(ctx, req)

		require.Error(t, err)
	})

	t.Run("returns content successfully", func(t *testing.T) {
		mockDoc := &mockDocumentService{
			content: "# Hello World\n\nThis is the document content.",
		}

		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://documents/doc-123")
		result, err := server.handleDocumentContentResource(ctx, req)

		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Equal(t, "# Hello World\n\nThis is the document content.", result.Contents[0].Text)
		assert.Equal(t, "text/plain", result.Contents[0].MIMEType)
	})

	t.Run("returns error on get content failure", func(t *testing.T) {
		mockDoc := &mockDocumentService{
			err: errors.New("content not found"),
		}

		ports := &Ports{Search: &mockSearchService{}, Document: mockDoc}
		server, err := NewServer(ports)
		require.NoError(t, err)

		req := makeReadResourceRequest("sercha://documents/doc-123")
		_, err = server.handleDocumentContentResource(ctx, req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting document content")
	})
}
