package notion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// TestAllNormalisers_MIMETypeConstants verifies MIME type constants are correct.
func TestAllNormalisers_MIMETypeConstants(t *testing.T) {
	assert.Equal(t, "application/vnd.notion.page+json", MIMETypeNotionPage)
	assert.Equal(t, "application/vnd.notion.database+json", MIMETypeNotionDB)
	assert.Equal(t, "application/vnd.notion.database-item+json", MIMETypeNotionDBItem)
}

// TestAllNormalisers_InterfaceCompliance verifies all normalisers implement the interface.
func TestAllNormalisers_InterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*PageNormaliser)(nil)
	var _ driven.Normaliser = (*DatabaseNormaliser)(nil)
	var _ driven.Normaliser = (*DatabaseItemNormaliser)(nil)
}

// TestAllNormalisers_SamePriority verifies all connector-specific normalisers have the same priority.
func TestAllNormalisers_SamePriority(t *testing.T) {
	page := NewPage()
	db := NewDatabase()
	item := NewDatabaseItem()

	assert.Equal(t, page.Priority(), db.Priority())
	assert.Equal(t, page.Priority(), item.Priority())
	assert.Equal(t, 95, page.Priority())
}

// TestAllNormalisers_SameConnectorType verifies all normalisers support the "notion" connector.
func TestAllNormalisers_SameConnectorType(t *testing.T) {
	page := NewPage()
	db := NewDatabase()
	item := NewDatabaseItem()

	assert.Equal(t, []string{"notion"}, page.SupportedConnectorTypes())
	assert.Equal(t, []string{"notion"}, db.SupportedConnectorTypes())
	assert.Equal(t, []string{"notion"}, item.SupportedConnectorTypes())
}

// TestAllNormalisers_UniqueMIMETypes verifies each normaliser has unique MIME types.
func TestAllNormalisers_UniqueMIMETypes(t *testing.T) {
	page := NewPage()
	db := NewDatabase()
	item := NewDatabaseItem()

	pageMIMEs := page.SupportedMIMETypes()
	dbMIMEs := db.SupportedMIMETypes()
	itemMIMEs := item.SupportedMIMETypes()

	// Each should have exactly one MIME type
	assert.Len(t, pageMIMEs, 1)
	assert.Len(t, dbMIMEs, 1)
	assert.Len(t, itemMIMEs, 1)

	// All should be different
	assert.NotEqual(t, pageMIMEs[0], dbMIMEs[0])
	assert.NotEqual(t, pageMIMEs[0], itemMIMEs[0])
	assert.NotEqual(t, dbMIMEs[0], itemMIMEs[0])
}

// TestAllNormalisers_TableDriven tests all normalisers with various scenarios.
func TestAllNormalisers_TableDriven(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		normaliser     driven.Normaliser
		raw            *domain.RawDocument
		expectError    bool
		expectedTitle  string
		expectedFormat string
		contentCheck   func(t *testing.T, content string)
	}{
		{
			name:       "Page with full metadata",
			normaliser: NewPage(),
			raw: &domain.RawDocument{
				SourceID: "src-1",
				URI:      "https://notion.so/page/1",
				MIMEType: MIMETypeNotionPage,
				Content:  []byte("Page content"),
				Metadata: map[string]any{
					"title": "Test Page",
					"icon":  "📄",
					"comments": []string{
						"Comment 1",
					},
				},
			},
			expectError:    false,
			expectedTitle:  "Test Page",
			expectedFormat: "notion_page",
			contentCheck: func(t *testing.T, content string) {
				assert.Contains(t, content, "# Test Page")
				assert.Contains(t, content, "📄")
				assert.Contains(t, content, "Page content")
				assert.Contains(t, content, "## Comments")
			},
		},
		{
			name:       "Database with schema",
			normaliser: NewDatabase(),
			raw: &domain.RawDocument{
				SourceID: "src-2",
				URI:      "https://notion.so/db/2",
				MIMEType: MIMETypeNotionDB,
				Content:  []byte("# Database Schema\n\nProperties:\n- Name\n- Status"),
				Metadata: map[string]any{
					"title": "Tasks DB",
				},
			},
			expectError:    false,
			expectedTitle:  "Tasks DB",
			expectedFormat: "notion_database",
			contentCheck: func(t *testing.T, content string) {
				assert.Contains(t, content, "Database Schema")
				assert.Contains(t, content, "Properties")
			},
		},
		{
			name:       "Database Item with properties",
			normaliser: NewDatabaseItem(),
			raw: &domain.RawDocument{
				SourceID: "src-3",
				URI:      "https://notion.so/page/3",
				MIMEType: MIMETypeNotionDBItem,
				Content:  []byte("Task description"),
				Metadata: map[string]any{
					"title":        "My Task",
					"prop_Status":  "Done",
					"prop_Owner":   "Alice",
					"prop_DueDate": "2024-12-31",
				},
			},
			expectError:    false,
			expectedTitle:  "My Task",
			expectedFormat: "notion_database_item",
			contentCheck: func(t *testing.T, content string) {
				assert.Contains(t, content, "# My Task")
				assert.Contains(t, content, "## Properties")
				assert.Contains(t, content, "Status:")
				assert.Contains(t, content, "Owner:")
				assert.Contains(t, content, "## Content")
				assert.Contains(t, content, "Task description")
			},
		},
		{
			name:          "Page with nil input",
			normaliser:    NewPage(),
			raw:           nil,
			expectError:   true,
			expectedTitle: "",
		},
		{
			name:          "Database with nil input",
			normaliser:    NewDatabase(),
			raw:           nil,
			expectError:   true,
			expectedTitle: "",
		},
		{
			name:          "Database Item with nil input",
			normaliser:    NewDatabaseItem(),
			raw:           nil,
			expectError:   true,
			expectedTitle: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.normaliser.Normalise(ctx, tc.raw)

			if tc.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			doc := result.Document
			assert.NotEmpty(t, doc.ID)
			assert.Equal(t, tc.expectedTitle, doc.Title)
			assert.Equal(t, tc.raw.SourceID, doc.SourceID)
			assert.Equal(t, tc.raw.URI, doc.URI)
			assert.NotZero(t, doc.CreatedAt)
			assert.NotZero(t, doc.UpdatedAt)

			// Check metadata
			assert.Equal(t, tc.raw.MIMEType, doc.Metadata["mime_type"])
			assert.Equal(t, tc.expectedFormat, doc.Metadata["format"])

			// Run content check if provided
			if tc.contentCheck != nil {
				tc.contentCheck(t, doc.Content)
			}
		})
	}
}

// TestAllNormalisers_MetadataIsolation verifies that all normalisers properly copy metadata.
func TestAllNormalisers_MetadataIsolation(t *testing.T) {
	ctx := context.Background()

	normalisers := []struct {
		name       string
		normaliser driven.Normaliser
		mimeType   string
	}{
		{"Page", NewPage(), MIMETypeNotionPage},
		{"Database", NewDatabase(), MIMETypeNotionDB},
		{"DatabaseItem", NewDatabaseItem(), MIMETypeNotionDBItem},
	}

	for _, tc := range normalisers {
		t.Run(tc.name, func(t *testing.T) {
			originalMeta := map[string]any{
				"title":  "Test",
				"custom": "value",
			}

			raw := &domain.RawDocument{
				SourceID: "test",
				URI:      "https://notion.so/test",
				MIMEType: tc.mimeType,
				Content:  []byte("content"),
				Metadata: originalMeta,
			}

			result, err := tc.normaliser.Normalise(ctx, raw)
			require.NoError(t, err)

			// Original should not be modified
			assert.NotContains(t, originalMeta, "mime_type")
			assert.NotContains(t, originalMeta, "format")

			// Result should have new fields
			assert.Contains(t, result.Document.Metadata, "mime_type")
			assert.Contains(t, result.Document.Metadata, "format")
		})
	}
}
