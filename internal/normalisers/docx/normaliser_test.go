package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// createTestDOCX creates a minimal valid DOCX file in memory.
func createTestDOCX(documentXML, coreXML string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add [Content_Types].xml (required for valid DOCX)
	contentTypes, _ := w.Create("[Content_Types].xml")
	contentTypes.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="xml" ContentType="application/xml"/>
</Types>`))

	// Add word/document.xml
	if documentXML != "" {
		doc, _ := w.Create("word/document.xml")
		doc.Write([]byte(documentXML))
	}

	// Add docProps/core.xml if provided
	if coreXML != "" {
		core, _ := w.Create("docProps/core.xml")
		core.Write([]byte(coreXML))
	}

	w.Close()
	return buf.Bytes()
}

func TestNew(t *testing.T) {
	normaliser := New()
	require.NotNil(t, normaliser)
	assert.IsType(t, &Normaliser{}, normaliser)
}

func TestSupportedMIMETypes(t *testing.T) {
	normaliser := New()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	assert.Len(t, mimeTypes, 1)
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

	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>
</w:body>
</w:document>`

	coreXML := `<?xml version="1.0" encoding="UTF-8"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>Test Document</dc:title>
</cp:coreProperties>`

	content := createTestDOCX(docXML, coreXML)

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/document.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "Test Document", doc.Title)
	assert.Contains(t, doc.Content, "Hello World")
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, raw.MIMEType, doc.Metadata["mime_type"])
	assert.Equal(t, "docx", doc.Metadata["format"])
}

func TestNormalise_NilDocument(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestNormalise_InvalidZip(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/invalid.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  []byte("not a zip file"),
	}

	result, err := normaliser.Normalise(ctx, raw)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestNormalise_TitleFallbackToFilename(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>Content</w:t></w:r></w:p>
</w:body>
</w:document>`

	// No core.xml - should fall back to filename
	content := createTestDOCX(docXML, "")

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/my_document.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	assert.Equal(t, "my document", result.Document.Title)
}

func TestNormalise_MultipleParagraphs(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>First paragraph</w:t></w:r></w:p>
<w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p>
<w:p><w:r><w:t>Third paragraph</w:t></w:r></w:p>
</w:body>
</w:document>`

	content := createTestDOCX(docXML, "")

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/doc.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	assert.Contains(t, result.Document.Content, "First paragraph")
	assert.Contains(t, result.Document.Content, "Second paragraph")
	assert.Contains(t, result.Document.Content, "Third paragraph")
	// Paragraphs should be separated by newlines
	assert.Contains(t, result.Document.Content, "\n")
}

func TestNormalise_MultipleRuns(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	// Multiple runs in a single paragraph (e.g., different formatting)
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p>
<w:r><w:t>Hello </w:t></w:r>
<w:r><w:t>World</w:t></w:r>
</w:p>
</w:body>
</w:document>`

	content := createTestDOCX(docXML, "")

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/doc.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result.Document.Content)
}

func TestNormalise_EmptyDocument(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
</w:body>
</w:document>`

	content := createTestDOCX(docXML, "")

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/empty.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	assert.Empty(t, result.Document.Content)
}

func TestNormalise_MetadataPreserved(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body><w:p><w:r><w:t>Test</w:t></w:r></w:p></w:body>
</w:document>`

	content := createTestDOCX(docXML, "")

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/doc.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
		Metadata: map[string]any{
			"author": "test-author",
			"custom": "value",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Equal(t, "test-author", doc.Metadata["author"])
	assert.Equal(t, "value", doc.Metadata["custom"])
	assert.Equal(t, "docx", doc.Metadata["format"])
}

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}

func BenchmarkNormalise(b *testing.B) {
	normaliser := New()
	ctx := context.Background()

	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>
</w:body>
</w:document>`

	content := createTestDOCX(docXML, "")

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/test/document.docx",
		MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:  content,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normaliser.Normalise(ctx, raw)
	}
}
