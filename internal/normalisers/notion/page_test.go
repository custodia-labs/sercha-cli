package notion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNewPage(t *testing.T) {
	normaliser := NewPage()
	require.NotNil(t, normaliser)
	assert.IsType(t, &PageNormaliser{}, normaliser)
}

func TestPageNormaliser_SupportedMIMETypes(t *testing.T) {
	normaliser := NewPage()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, MIMETypeNotionPage)
	assert.Equal(t, "application/vnd.notion.page+json", MIMETypeNotionPage)
	assert.Len(t, mimeTypes, 1)
}

func TestPageNormaliser_SupportedConnectorTypes(t *testing.T) {
	normaliser := NewPage()
	connectorTypes := normaliser.SupportedConnectorTypes()

	require.NotEmpty(t, connectorTypes)
	assert.Contains(t, connectorTypes, "notion")
	assert.Len(t, connectorTypes, 1)
}

func TestPageNormaliser_Priority(t *testing.T) {
	normaliser := NewPage()
	assert.Equal(t, 95, normaliser.Priority())
}

func TestPageNormaliser_Normalise_Success(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("This is the page content."),
		Metadata: map[string]any{
			"title": "My Page Title",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "My Page Title", doc.Title)
	assert.Contains(t, doc.Content, "# My Page Title")
	assert.Contains(t, doc.Content, "This is the page content.")
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, MIMETypeNotionPage, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_page", doc.Metadata["format"])
	assert.NotZero(t, doc.CreatedAt)
	assert.NotZero(t, doc.UpdatedAt)
}

func TestPageNormaliser_Normalise_NilDocument(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestPageNormaliser_Normalise_EmptyContent(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte(""),
		Metadata: map[string]any{
			"title": "Empty Page",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Empty Page", doc.Title)
	assert.Contains(t, doc.Content, "# Empty Page")
	// Content should have at least the title header
	assert.NotEmpty(t, doc.Content)
}

func TestPageNormaliser_Normalise_UntitledFallback(t *testing.T) {
	tests := []struct {
		name          string
		metadata      map[string]any
		expectedTitle string
	}{
		{
			name:          "no title in metadata",
			metadata:      map[string]any{},
			expectedTitle: "Untitled",
		},
		{
			name: "empty title string",
			metadata: map[string]any{
				"title": "",
			},
			expectedTitle: "Untitled",
		},
		{
			name: "title not a string",
			metadata: map[string]any{
				"title": 123,
			},
			expectedTitle: "Untitled",
		},
		{
			name:          "nil metadata",
			metadata:      nil,
			expectedTitle: "Untitled",
		},
	}

	normaliser := NewPage()
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := &domain.RawDocument{
				SourceID: "test-source",
				URI:      "https://notion.so/page/123",
				MIMEType: MIMETypeNotionPage,
				Content:  []byte("Content"),
				Metadata: tc.metadata,
			}

			result, err := normaliser.Normalise(ctx, raw)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTitle, result.Document.Title)
			assert.Contains(t, result.Document.Content, "# "+tc.expectedTitle)
		})
	}
}

func TestPageNormaliser_Normalise_WithIcon(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("Page content here."),
		Metadata: map[string]any{
			"title": "Document",
			"icon":  "📄",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Contains(t, doc.Content, "📄")
	assert.Contains(t, doc.Content, "# Document")
}

func TestPageNormaliser_Normalise_WithEmptyIcon(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("Page content here."),
		Metadata: map[string]any{
			"title": "Document",
			"icon":  "",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Empty icon should not be included
	assert.NotContains(t, doc.Content, "  ") // No double space from empty icon
}

func TestPageNormaliser_Normalise_WithComments(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("Main content."),
		Metadata: map[string]any{
			"title": "Page with Comments",
			"comments": []string{
				"First comment",
				"Second comment",
				"Third comment",
			},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Contains(t, doc.Content, "## Comments")
	assert.Contains(t, doc.Content, "- First comment")
	assert.Contains(t, doc.Content, "- Second comment")
	assert.Contains(t, doc.Content, "- Third comment")
	assert.Contains(t, doc.Content, "---") // Separator before comments
}

func TestPageNormaliser_Normalise_WithEmptyComments(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("Main content."),
		Metadata: map[string]any{
			"title":    "Page",
			"comments": []string{},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.NotContains(t, doc.Content, "## Comments")
	assert.NotContains(t, doc.Content, "---")
}

func TestPageNormaliser_Normalise_MetadataPreserved(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("Content"),
		Metadata: map[string]any{
			"title":      "Test Page",
			"author":     "John Doe",
			"created_at": "2024-01-01",
			"tags":       []string{"notion", "test"},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Original metadata should be preserved
	assert.Equal(t, "John Doe", doc.Metadata["author"])
	assert.Equal(t, "2024-01-01", doc.Metadata["created_at"])
	assert.Equal(t, []string{"notion", "test"}, doc.Metadata["tags"])

	// New metadata should be added
	assert.Equal(t, MIMETypeNotionPage, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_page", doc.Metadata["format"])
}

func TestPageNormaliser_Normalise_MetadataNotMutated(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	originalMetadata := map[string]any{
		"title":  "Test",
		"custom": "value",
	}

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("Content"),
		Metadata: originalMetadata,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	// Original metadata should not be modified
	assert.NotContains(t, originalMetadata, "mime_type")
	assert.NotContains(t, originalMetadata, "format")

	// Result metadata should have new fields
	assert.Contains(t, result.Document.Metadata, "mime_type")
	assert.Contains(t, result.Document.Metadata, "format")
}

func TestPageNormaliser_Normalise_ComplexPage(t *testing.T) {
	normaliser := NewPage()
	ctx := context.Background()

	complexContent := `This is a complex page with multiple paragraphs.

It includes lists, code blocks, and other formatting.

Some bullet points:
- Point 1
- Point 2
- Point 3

And some more text at the end.`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/complex",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte(complexContent),
		Metadata: map[string]any{
			"title": "Complex Document",
			"icon":  "📚",
			"comments": []string{
				"Great work!",
				"Needs review",
			},
			"author":     "Jane Doe",
			"created_at": "2024-01-15",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Complex Document", doc.Title)
	assert.Contains(t, doc.Content, "# Complex Document")
	assert.Contains(t, doc.Content, "📚")
	assert.Contains(t, doc.Content, complexContent)
	assert.Contains(t, doc.Content, "## Comments")
	assert.Contains(t, doc.Content, "- Great work!")
	assert.Contains(t, doc.Content, "- Needs review")

	// Verify metadata
	assert.Equal(t, "Jane Doe", doc.Metadata["author"])
	assert.Equal(t, MIMETypeNotionPage, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_page", doc.Metadata["format"])
}

func TestPageNormaliser_InterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*PageNormaliser)(nil)
}

func BenchmarkPageNormaliser_Normalise(b *testing.B) {
	normaliser := NewPage()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/123",
		MIMEType: MIMETypeNotionPage,
		Content:  []byte("This is test content with some text."),
		Metadata: map[string]any{
			"title": "Test Page",
			"icon":  "📄",
			"comments": []string{
				"Comment 1",
				"Comment 2",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normaliser.Normalise(ctx, raw)
	}
}
