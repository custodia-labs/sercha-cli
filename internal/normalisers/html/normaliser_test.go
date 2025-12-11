package html

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
	assert.Contains(t, mimeTypes, "text/html")
	assert.Contains(t, mimeTypes, "application/xhtml+xml")
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
		URI:      "/path/to/document.html",
		MIMEType: "text/html",
		Content:  []byte("<html><head><title>Test Page</title></head><body><p>Hello World</p></body></html>"),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "Test Page", doc.Title)
	assert.Contains(t, doc.Content, "Hello World")
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, "text/html", doc.Metadata["mime_type"])
	assert.Equal(t, "html", doc.Metadata["format"])
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
		URI:      "/path/to/empty.html",
		MIMEType: "text/html",
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
			name:          "title tag",
			content:       "<html><head><title>My Document</title></head><body></body></html>",
			uri:           "/doc.html",
			expectedTitle: "My Document",
		},
		{
			name:          "title with extra spaces",
			content:       "<title>   Spaced Title   </title>",
			uri:           "/doc.html",
			expectedTitle: "Spaced Title",
		},
		{
			name:          "title with HTML entities",
			content:       "<title>Tom &amp; Jerry</title>",
			uri:           "/doc.html",
			expectedTitle: "Tom & Jerry",
		},
		{
			name:          "no title - fallback to filename",
			content:       "<html><body>Just content</body></html>",
			uri:           "/my_document.html",
			expectedTitle: "my document",
		},
		{
			name:          "empty title - fallback to filename",
			content:       "<title></title><body>Content</body>",
			uri:           "/readme.html",
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
				MIMEType: "text/html",
				Content:  []byte(tc.content),
			}

			result, err := normaliser.Normalise(ctx, raw)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTitle, result.Document.Title)
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple paragraph",
			input:    "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "nested tags",
			input:    "<div><p><strong>Bold</strong> text</p></div>",
			expected: "Bold text",
		},
		{
			name:     "script removed",
			input:    "<p>Before</p><script>alert('evil');</script><p>After</p>",
			expected: "Before\nAfter",
		},
		{
			name:     "style removed",
			input:    "<style>.foo { color: red; }</style><p>Content</p>",
			expected: "Content",
		},
		{
			name:     "noscript removed",
			input:    "<p>Content</p><noscript>No JS fallback</noscript>",
			expected: "Content",
		},
		{
			name:     "head removed",
			input:    "<head><meta charset='utf-8'><title>Title</title></head><body>Content</body>",
			expected: "Content",
		},
		{
			name:     "br to newline",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "block elements create newlines",
			input:    "<div>Block 1</div><div>Block 2</div>",
			expected: "Block 1\nBlock 2",
		},
		{
			name:     "HTML entities decoded",
			input:    "<p>&lt;tag&gt; &amp; &quot;quotes&quot;</p>",
			expected: "<tag> & \"quotes\"",
		},
		{
			name:     "comments removed",
			input:    "<p>Before</p><!-- comment --><p>After</p>",
			expected: "Before\nAfter",
		},
		{
			name:     "list items",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "Item 1\nItem 2",
		},
		{
			name:     "headings",
			input:    "<h1>Title</h1><h2>Subtitle</h2><p>Content</p>",
			expected: "Title\nSubtitle\nContent",
		},
		{
			name:     "links - text preserved",
			input:    `<a href="https://example.com">Click here</a>`,
			expected: "Click here",
		},
		{
			name:     "images removed",
			input:    `<p>See <img src="image.png" alt="Image"> here</p>`,
			expected: "See here",
		},
		{
			name:     "table",
			input:    "<table><tr><td>Cell 1</td><td>Cell 2</td></tr></table>",
			expected: "Cell 1Cell 2",
		},
		{
			name:     "svg removed",
			input:    `<p>Before</p><svg width="100"><circle cx="50"/></svg><p>After</p>`,
			expected: "Before\nAfter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stripHTML(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalise_ComplexHTML(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	complexHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Complex Page</title>
    <style>
        body { font-family: Arial; }
        .highlight { background: yellow; }
    </style>
</head>
<body>
    <header>
        <h1>Main Title</h1>
        <nav>
            <a href="/home">Home</a>
            <a href="/about">About</a>
        </nav>
    </header>

    <main>
        <article>
            <h2>Article Title</h2>
            <p>This is a <strong>paragraph</strong> with <em>emphasis</em>.</p>

            <ul>
                <li>First item</li>
                <li>Second item</li>
            </ul>

            <blockquote>
                A famous quote here.
            </blockquote>
        </article>
    </main>

    <script>
        console.log('This should be removed');
    </script>

    <!-- This is a comment that should be removed -->

    <footer>
        <p>&copy; 2024 Example Corp</p>
    </footer>
</body>
</html>`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/complex.html",
		MIMEType: "text/html",
		Content:  []byte(complexHTML),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Complex Page", doc.Title)

	// Verify content is stripped of HTML
	assert.NotContains(t, doc.Content, "<strong>")
	assert.Contains(t, doc.Content, "paragraph")
	assert.NotContains(t, doc.Content, "console.log")
	assert.NotContains(t, doc.Content, "font-family")
	assert.NotContains(t, doc.Content, "<!--")
	assert.Contains(t, doc.Content, "Main Title")
	assert.Contains(t, doc.Content, "First item")
	assert.Contains(t, doc.Content, "2024 Example Corp") // Entity decoded
}

func TestNormalise_MetadataPreserved(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/document.html",
		MIMEType: "text/html",
		Content:  []byte("<html><body>Test</body></html>"),
		Metadata: map[string]any{
			"author": "test",
			"tags":   []string{"html", "test"},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Equal(t, "test", doc.Metadata["author"])
	assert.Equal(t, []string{"html", "test"}, doc.Metadata["tags"])
	assert.Equal(t, "text/html", doc.Metadata["mime_type"])
	assert.Equal(t, "html", doc.Metadata["format"])
}

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}

func BenchmarkNormalise(b *testing.B) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/document.html",
		MIMEType: "text/html",
		Content:  []byte("<html><head><title>Test</title></head><body><p>Test content</p></body></html>"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normaliser.Normalise(ctx, raw)
	}
}

func BenchmarkStripHTML(b *testing.B) {
	content := `<html>
<head><title>Test</title><style>body{}</style></head>
<body>
<h1>Heading</h1>
<p>Paragraph with <strong>bold</strong> and <em>italic</em>.</p>
<ul><li>Item 1</li><li>Item 2</li></ul>
<script>alert('test');</script>
</body>
</html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stripHTML(content)
	}
}
