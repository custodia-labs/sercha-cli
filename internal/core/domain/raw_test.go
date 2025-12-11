package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRawDocument_Fields tests RawDocument structure fields
func TestRawDocument_Fields(t *testing.T) {
	parentURI := "file:///parent.pdf"
	raw := RawDocument{
		SourceID:  "source-123",
		URI:       "file:///document.pdf",
		MIMEType:  "application/pdf",
		Content:   []byte("PDF content here"),
		ParentURI: &parentURI,
		Metadata:  map[string]any{"size": 1024},
	}

	assert.Equal(t, "source-123", raw.SourceID)
	assert.Equal(t, "file:///document.pdf", raw.URI)
	assert.Equal(t, "application/pdf", raw.MIMEType)
	assert.Equal(t, []byte("PDF content here"), raw.Content)
	assert.NotNil(t, raw.ParentURI)
	assert.Equal(t, "file:///parent.pdf", *raw.ParentURI)
	assert.Equal(t, 1024, raw.Metadata["size"])
}

// TestRawDocument_NoParent tests RawDocument without parent
func TestRawDocument_NoParent(t *testing.T) {
	raw := RawDocument{
		SourceID:  "source-123",
		URI:       "file:///standalone.txt",
		MIMEType:  "text/plain",
		Content:   []byte("Text content"),
		ParentURI: nil,
	}

	assert.Nil(t, raw.ParentURI)
}

// TestRawDocument_EmptyContent tests RawDocument with empty content
func TestRawDocument_EmptyContent(t *testing.T) {
	raw := RawDocument{
		SourceID: "source-123",
		URI:      "file:///empty.txt",
		MIMEType: "text/plain",
		Content:  []byte{},
	}

	assert.NotNil(t, raw.Content)
	assert.Empty(t, raw.Content)
}

// TestRawDocument_NilContent tests RawDocument with nil content
func TestRawDocument_NilContent(t *testing.T) {
	raw := RawDocument{
		SourceID: "source-123",
		URI:      "file:///nil.txt",
		MIMEType: "text/plain",
		Content:  nil,
	}

	assert.Nil(t, raw.Content)
}

// TestRawDocument_LargeContent tests RawDocument with large binary content
func TestRawDocument_LargeContent(t *testing.T) {
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	raw := RawDocument{
		SourceID: "source-123",
		URI:      "file:///large.bin",
		MIMEType: "application/octet-stream",
		Content:  largeContent,
	}

	assert.Len(t, raw.Content, 1024*1024)
}

// TestRawDocument_MIMETypes tests various MIME types
func TestRawDocument_MIMETypes(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		content  []byte
	}{
		{"text file", "text/plain", []byte("text content")},
		{"html file", "text/html", []byte("<html></html>")},
		{"pdf file", "application/pdf", []byte("%PDF-1.4")},
		{"json file", "application/json", []byte("{}")},
		{"xml file", "application/xml", []byte("<root/>")},
		{"image", "image/png", []byte{0x89, 0x50, 0x4E, 0x47}},
		{"video", "video/mp4", []byte("ftyp")},
		{"empty mime", "", []byte("content")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := RawDocument{
				SourceID: "source-123",
				URI:      "file:///test",
				MIMEType: tt.mimeType,
				Content:  tt.content,
			}
			assert.Equal(t, tt.mimeType, raw.MIMEType)
			assert.Equal(t, tt.content, raw.Content)
		})
	}
}

// TestRawDocument_Metadata tests various metadata scenarios
func TestRawDocument_Metadata(t *testing.T) {
	raw := RawDocument{
		SourceID: "source-123",
		URI:      "file:///test.txt",
		MIMEType: "text/plain",
		Content:  []byte("content"),
		Metadata: map[string]any{
			"size":        1024,
			"modified":    "2024-01-01T00:00:00Z",
			"author":      "Test Author",
			"tags":        []string{"tag1", "tag2"},
			"is_archived": false,
			"nested":      map[string]any{"key": "value"},
		},
	}

	assert.Equal(t, 1024, raw.Metadata["size"])
	assert.Equal(t, "Test Author", raw.Metadata["author"])
	assert.IsType(t, []string{}, raw.Metadata["tags"])
	assert.False(t, raw.Metadata["is_archived"].(bool))
}

// TestRawDocument_NilMetadata tests RawDocument with nil metadata
func TestRawDocument_NilMetadata(t *testing.T) {
	raw := RawDocument{
		SourceID: "source-123",
		URI:      "file:///test.txt",
		MIMEType: "text/plain",
		Content:  []byte("content"),
		Metadata: nil,
	}

	assert.Nil(t, raw.Metadata)
}

// TestRawDocument_EmptyMetadata tests RawDocument with empty metadata
func TestRawDocument_EmptyMetadata(t *testing.T) {
	raw := RawDocument{
		SourceID: "source-123",
		URI:      "file:///test.txt",
		MIMEType: "text/plain",
		Content:  []byte("content"),
		Metadata: map[string]any{},
	}

	assert.NotNil(t, raw.Metadata)
	assert.Empty(t, raw.Metadata)
}

// TestChangeType_Constants tests all ChangeType constants
func TestChangeType_Constants(t *testing.T) {
	assert.Equal(t, ChangeType(0), ChangeCreated)
	assert.Equal(t, ChangeType(1), ChangeUpdated)
	assert.Equal(t, ChangeType(2), ChangeDeleted)
}

// TestChangeType_Values tests that change types have expected values
func TestChangeType_Values(t *testing.T) {
	// iota starts at 0 and increments
	assert.Equal(t, 0, int(ChangeCreated))
	assert.Equal(t, 1, int(ChangeUpdated))
	assert.Equal(t, 2, int(ChangeDeleted))
}

// TestChangeType_Distinct tests that change types are distinct
func TestChangeType_Distinct(t *testing.T) {
	assert.NotEqual(t, ChangeCreated, ChangeUpdated)
	assert.NotEqual(t, ChangeUpdated, ChangeDeleted)
	assert.NotEqual(t, ChangeCreated, ChangeDeleted)
}

// TestRawDocumentChange_Created tests change with created type
func TestRawDocumentChange_Created(t *testing.T) {
	change := RawDocumentChange{
		Type: ChangeCreated,
		Document: RawDocument{
			SourceID: "source-123",
			URI:      "file:///new.txt",
			MIMEType: "text/plain",
			Content:  []byte("new content"),
		},
	}

	assert.Equal(t, ChangeCreated, change.Type)
	assert.Equal(t, "file:///new.txt", change.Document.URI)
}

// TestRawDocumentChange_Updated tests change with updated type
func TestRawDocumentChange_Updated(t *testing.T) {
	change := RawDocumentChange{
		Type: ChangeUpdated,
		Document: RawDocument{
			SourceID: "source-123",
			URI:      "file:///existing.txt",
			MIMEType: "text/plain",
			Content:  []byte("updated content"),
		},
	}

	assert.Equal(t, ChangeUpdated, change.Type)
	assert.Equal(t, "updated content", string(change.Document.Content))
}

// TestRawDocumentChange_Deleted tests change with deleted type
func TestRawDocumentChange_Deleted(t *testing.T) {
	change := RawDocumentChange{
		Type: ChangeDeleted,
		Document: RawDocument{
			SourceID: "source-123",
			URI:      "file:///deleted.txt",
			MIMEType: "text/plain",
			Content:  nil, // Content may be nil for deleted documents
		},
	}

	assert.Equal(t, ChangeDeleted, change.Type)
	assert.Equal(t, "file:///deleted.txt", change.Document.URI)
	assert.Nil(t, change.Document.Content)
}

// TestRawDocumentChange_Fields tests RawDocumentChange structure
func TestRawDocumentChange_Fields(t *testing.T) {
	doc := RawDocument{
		SourceID: "source-123",
		URI:      "file:///test.txt",
		MIMEType: "text/plain",
		Content:  []byte("content"),
	}

	change := RawDocumentChange{
		Type:     ChangeCreated,
		Document: doc,
	}

	assert.Equal(t, ChangeCreated, change.Type)
	assert.Equal(t, doc, change.Document)
}

// TestRawDocumentChange_MultipleChanges tests sequence of changes
func TestRawDocumentChange_MultipleChanges(t *testing.T) {
	changes := []RawDocumentChange{
		{
			Type: ChangeCreated,
			Document: RawDocument{
				SourceID: "source-123",
				URI:      "file:///file1.txt",
				Content:  []byte("content1"),
			},
		},
		{
			Type: ChangeUpdated,
			Document: RawDocument{
				SourceID: "source-123",
				URI:      "file:///file1.txt",
				Content:  []byte("updated content1"),
			},
		},
		{
			Type: ChangeDeleted,
			Document: RawDocument{
				SourceID: "source-123",
				URI:      "file:///file1.txt",
			},
		},
	}

	assert.Len(t, changes, 3)
	assert.Equal(t, ChangeCreated, changes[0].Type)
	assert.Equal(t, ChangeUpdated, changes[1].Type)
	assert.Equal(t, ChangeDeleted, changes[2].Type)
}

// TestRawDocument_BinaryContent tests various binary content types
func TestRawDocument_BinaryContent(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		content  []byte
	}{
		{
			name:     "PNG image",
			mimeType: "image/png",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
		},
		{
			name:     "JPEG image",
			mimeType: "image/jpeg",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0},
		},
		{
			name:     "ZIP archive",
			mimeType: "application/zip",
			content:  []byte{0x50, 0x4B, 0x03, 0x04},
		},
		{
			name:     "null bytes",
			mimeType: "application/octet-stream",
			content:  []byte{0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := RawDocument{
				SourceID: "source-123",
				URI:      "file:///binary",
				MIMEType: tt.mimeType,
				Content:  tt.content,
			}
			assert.Equal(t, tt.content, raw.Content)
		})
	}
}

// TestRawDocument_URIFormats tests various URI formats
func TestRawDocument_URIFormats(t *testing.T) {
	uris := []string{
		"file:///path/to/file.txt",
		"https://example.com/document",
		"drive://file-id",
		"gmail://message-id",
		"slack://channel/message",
		"/absolute/path",
		"relative/path",
		"",
	}

	for _, uri := range uris {
		t.Run(uri, func(t *testing.T) {
			raw := RawDocument{
				SourceID: "source-123",
				URI:      uri,
				MIMEType: "text/plain",
				Content:  []byte("content"),
			}
			assert.Equal(t, uri, raw.URI)
		})
	}
}

// TestRawDocument_ParentURIRelationship tests parent-child relationships
func TestRawDocument_ParentURIRelationship(t *testing.T) {
	parentURI := "file:///parent.pdf"

	children := []RawDocument{
		{
			SourceID:  "source-123",
			URI:       "file:///parent.pdf#page1",
			MIMEType:  "application/pdf",
			Content:   []byte("page 1 content"),
			ParentURI: &parentURI,
		},
		{
			SourceID:  "source-123",
			URI:       "file:///parent.pdf#page2",
			MIMEType:  "application/pdf",
			Content:   []byte("page 2 content"),
			ParentURI: &parentURI,
		},
	}

	for _, child := range children {
		assert.NotNil(t, child.ParentURI)
		assert.Equal(t, parentURI, *child.ParentURI)
	}
}

// TestChangeType_InvalidValue tests invalid ChangeType values
func TestChangeType_InvalidValue(t *testing.T) {
	invalidChange := ChangeType(999)
	assert.NotEqual(t, ChangeCreated, invalidChange)
	assert.NotEqual(t, ChangeUpdated, invalidChange)
	assert.NotEqual(t, ChangeDeleted, invalidChange)
}
