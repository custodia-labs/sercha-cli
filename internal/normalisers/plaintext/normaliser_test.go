package plaintext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNew(t *testing.T) {
	normaliser := New()
	require.NotNil(t, normaliser)
	assert.IsType(t, &Normaliser{}, normaliser)
}

func TestSupportedMIMETypes(t *testing.T) {
	normaliser := New()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, "text/plain")
	assert.Contains(t, mimeTypes, "text/x-go")
	assert.Contains(t, mimeTypes, "application/json")
}

func TestSupportedConnectorTypes(t *testing.T) {
	normaliser := New()
	assert.Nil(t, normaliser.SupportedConnectorTypes())
}

func TestPriority(t *testing.T) {
	normaliser := New()
	assert.Equal(t, 5, normaliser.Priority())
}

func TestNormalise_Success(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/document.txt",
		MIMEType: "text/plain",
		Content:  []byte("This is plain text content."),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "document", doc.Title)
	assert.Equal(t, "This is plain text content.", doc.Content)
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, "text/plain", doc.Metadata["mime_type"])
}

func TestNormalise_NilDocument(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestNormalise_EmptyContent(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/empty.txt",
		MIMEType: "text/plain",
		Content:  []byte(""),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Document.Content)
}

func TestNormalise_TitleExtraction(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		expectedTitle string
	}{
		{
			name:          "simple filename",
			uri:           "/path/to/document.txt",
			expectedTitle: "document",
		},
		{
			name:          "underscores to spaces",
			uri:           "/path/my_document_name.txt",
			expectedTitle: "my document name",
		},
		{
			name:          "dashes to spaces",
			uri:           "/path/my-document-name.txt",
			expectedTitle: "my document name",
		},
		{
			name:          "code file",
			uri:           "/src/main.go",
			expectedTitle: "main",
		},
	}

	normaliser := New()
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := &domain.RawDocument{
				SourceID: "test-source",
				URI:      tc.uri,
				MIMEType: "text/plain",
				Content:  []byte("content"),
			}

			result, err := normaliser.Normalise(ctx, raw)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTitle, result.Document.Title)
		})
	}
}

func TestNormalise_MetadataPreserved(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/document.txt",
		MIMEType: "text/plain",
		Content:  []byte("content"),
		Metadata: map[string]any{
			"author":     "test",
			"line_count": 100,
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Equal(t, "test", doc.Metadata["author"])
	assert.Equal(t, 100, doc.Metadata["line_count"])
	assert.Equal(t, "text/plain", doc.Metadata["mime_type"])
}

func TestNormalise_UnicodeContent(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	unicodeContent := `多语言文本测试
こんにちは世界
مرحبا بالعالم
Привет мир
🚀 Emoji test 🎉`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/unicode.txt",
		MIMEType: "text/plain",
		Content:  []byte(unicodeContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	assert.Equal(t, unicodeContent, result.Document.Content)
}

func TestNormalise_LargeContent(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/large.txt",
		MIMEType: "text/plain",
		Content:  largeContent,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	assert.Len(t, result.Document.Content, 1024*1024)
}

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}

func BenchmarkNormalise(b *testing.B) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/document.txt",
		MIMEType: "text/plain",
		Content:  []byte("This is test content for benchmarking."),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normaliser.Normalise(ctx, raw)
	}
}
