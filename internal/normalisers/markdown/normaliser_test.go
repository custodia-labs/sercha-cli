package markdown

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
	assert.Contains(t, mimeTypes, "text/markdown")
	assert.Contains(t, mimeTypes, "text/x-markdown")
	assert.Len(t, mimeTypes, 2)
}

func TestSupportedConnectorTypes(t *testing.T) {
	normaliser := New()
	assert.Nil(t, normaliser.SupportedConnectorTypes())
}

func TestPriority(t *testing.T) {
	normaliser := New()
	assert.Equal(t, 50, normaliser.Priority())
}

func TestNormalise_Success(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/document.md",
		MIMEType: "text/markdown",
		Content:  []byte("# Hello World\n\nThis is a test."),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "Hello World", doc.Title) // Title from first H1
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, "text/markdown", doc.Metadata["mime_type"])
	assert.Equal(t, "markdown", doc.Metadata["format"])
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
		URI:      "/path/to/empty.md",
		MIMEType: "text/markdown",
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
		content       string
		uri           string
		expectedTitle string
	}{
		{
			name:          "H1 heading",
			content:       "# My Document\n\nContent here.",
			uri:           "/doc.md",
			expectedTitle: "My Document",
		},
		{
			name:          "H1 with extra spaces",
			content:       "#   Spaced Title   \n\nContent",
			uri:           "/doc.md",
			expectedTitle: "Spaced Title",
		},
		{
			name:          "no heading - fallback to filename",
			content:       "Just some content without heading.",
			uri:           "/my_document.md",
			expectedTitle: "my document",
		},
		{
			name:          "H2 first - fallback to filename",
			content:       "## Second Level\n\nNo H1.",
			uri:           "/readme.md",
			expectedTitle: "readme",
		},
	}

	normaliser := New()
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := &domain.RawDocument{
				SourceID: "test-source",
				URI:      tc.uri,
				MIMEType: "text/markdown",
				Content:  []byte(tc.content),
			}

			result, err := normaliser.Normalise(ctx, raw)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTitle, result.Document.Title)
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "headings removed",
			input:    "# Title\n## Subtitle\n### Third",
			expected: "Title\nSubtitle\nThird",
		},
		{
			name:     "bold removed",
			input:    "This is **bold** text",
			expected: "This is bold text",
		},
		{
			name:     "links converted",
			input:    "Click [here](https://example.com)",
			expected: "Click here",
		},
		{
			name:     "images removed",
			input:    "See ![alt text](image.png) here",
			expected: "See  here",
		},
		{
			name:     "code blocks removed",
			input:    "Before\n```go\ncode here\n```\nAfter",
			expected: "Before\n\nAfter",
		},
		{
			name:     "inline code removed",
			input:    "Use `code` here",
			expected: "Use  here",
		},
		{
			name:     "blockquotes cleaned",
			input:    "> This is a quote",
			expected: "This is a quote",
		},
		{
			name:     "list markers removed",
			input:    "- Item 1\n- Item 2",
			expected: "Item 1\nItem 2",
		},
		{
			name:     "numbered list markers removed",
			input:    "1. First\n2. Second",
			expected: "First\nSecond",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stripMarkdown(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalise_ComplexMarkdown(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	complexMarkdown := `# Main Title

## Section 1

This is a paragraph with **bold** and *italic* text.

- List item 1
- List item 2
  - Nested item

### Subsection 1.1

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Section 2

| Column 1 | Column 2 |
|----------|----------|
| Data 1   | Data 2   |

[Link](https://example.com)

![Image](image.png)
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/complex.md",
		MIMEType: "text/markdown",
		Content:  []byte(complexMarkdown),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Main Title", doc.Title)

	// Verify content is stripped of markdown
	assert.NotContains(t, doc.Content, "**bold**")
	assert.Contains(t, doc.Content, "bold")
	assert.NotContains(t, doc.Content, "[Link]")
	assert.Contains(t, doc.Content, "Link")
	assert.NotContains(t, doc.Content, "```")
}

func TestNormalise_MetadataPreserved(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/document.md",
		MIMEType: "text/markdown",
		Content:  []byte("# Test"),
		Metadata: map[string]any{
			"author": "test",
			"tags":   []string{"markdown", "test"},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Equal(t, "test", doc.Metadata["author"])
	assert.Equal(t, []string{"markdown", "test"}, doc.Metadata["tags"])
	assert.Equal(t, "text/markdown", doc.Metadata["mime_type"])
	assert.Equal(t, "markdown", doc.Metadata["format"])
}

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}

func BenchmarkNormalise(b *testing.B) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/document.md",
		MIMEType: "text/markdown",
		Content:  []byte("# Test Document\n\nThis is test content with **bold** and *italic*."),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normaliser.Normalise(ctx, raw)
	}
}

func BenchmarkStripMarkdown(b *testing.B) {
	content := `# Heading

Paragraph with **bold** and *italic*.

- List item 1
- List item 2

[Link](https://example.com)

` + "```" + `
code block
` + "```"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stripMarkdown(content)
	}
}
