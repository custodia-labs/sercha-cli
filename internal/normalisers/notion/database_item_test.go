package notion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNewDatabaseItem(t *testing.T) {
	normaliser := NewDatabaseItem()
	require.NotNil(t, normaliser)
	assert.IsType(t, &DatabaseItemNormaliser{}, normaliser)
}

func TestDatabaseItemNormaliser_SupportedMIMETypes(t *testing.T) {
	normaliser := NewDatabaseItem()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, MIMETypeNotionDBItem)
	assert.Equal(t, "application/vnd.notion.database-item+json", MIMETypeNotionDBItem)
	assert.Len(t, mimeTypes, 1)
}

func TestDatabaseItemNormaliser_SupportedConnectorTypes(t *testing.T) {
	normaliser := NewDatabaseItem()
	connectorTypes := normaliser.SupportedConnectorTypes()

	require.NotEmpty(t, connectorTypes)
	assert.Contains(t, connectorTypes, "notion")
	assert.Len(t, connectorTypes, 1)
}

func TestDatabaseItemNormaliser_Priority(t *testing.T) {
	normaliser := NewDatabaseItem()
	assert.Equal(t, 95, normaliser.Priority())
}

func TestDatabaseItemNormaliser_Normalise_Success(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("This is the task description and notes."),
		Metadata: map[string]any{
			"title":        "Complete project documentation",
			"prop_Status":  "In Progress",
			"prop_Owner":   "John Doe",
			"prop_DueDate": "2024-12-31",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "Complete project documentation", doc.Title)
	assert.Contains(t, doc.Content, "# Complete project documentation")
	assert.Contains(t, doc.Content, "## Properties")
	assert.Contains(t, doc.Content, "**Status:** In Progress")
	assert.Contains(t, doc.Content, "**Owner:** John Doe")
	assert.Contains(t, doc.Content, "**DueDate:** 2024-12-31")
	assert.Contains(t, doc.Content, "## Content")
	assert.Contains(t, doc.Content, "This is the task description and notes.")
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, MIMETypeNotionDBItem, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_database_item", doc.Metadata["format"])
	assert.NotZero(t, doc.CreatedAt)
	assert.NotZero(t, doc.UpdatedAt)
}

func TestDatabaseItemNormaliser_Normalise_NilDocument(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestDatabaseItemNormaliser_Normalise_EmptyContent(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte(""),
		Metadata: map[string]any{
			"title":       "Empty Task",
			"prop_Status": "Not Started",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Empty Task", doc.Title)
	assert.Contains(t, doc.Content, "# Empty Task")
	assert.Contains(t, doc.Content, "## Properties")
	// Should not have Content section since content is empty
	assert.NotContains(t, doc.Content, "## Content\n\n\n")
}

func TestDatabaseItemNormaliser_Normalise_UntitledFallback(t *testing.T) {
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

	normaliser := NewDatabaseItem()
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := &domain.RawDocument{
				SourceID: "test-source",
				URI:      "https://notion.so/page/789",
				MIMEType: MIMETypeNotionDBItem,
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

func TestDatabaseItemNormaliser_Normalise_WithIcon(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Task details here."),
		Metadata: map[string]any{
			"title":       "Important Task",
			"icon":        "⭐",
			"prop_Status": "High Priority",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Contains(t, doc.Content, "⭐")
	assert.Contains(t, doc.Content, "# Important Task")
}

func TestDatabaseItemNormaliser_Normalise_WithEmptyIcon(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Task details."),
		Metadata: map[string]any{
			"title": "Task",
			"icon":  "",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Empty icon should not be included
	assert.NotContains(t, doc.Content, "  ") // No double space from empty icon
}

func TestDatabaseItemNormaliser_Normalise_WithProperties(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Main content."),
		Metadata: map[string]any{
			"title":          "Task Item",
			"prop_Status":    "In Progress",
			"prop_Priority":  "High",
			"prop_Assignee":  "Alice",
			"prop_DueDate":   "2024-12-31",
			"prop_Tags":      []string{"urgent", "documentation"},
			"prop_Completed": false,
			"non_prop_field": "should not appear in properties",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Contains(t, doc.Content, "## Properties")

	// Check that properties are included (note: order may vary due to map iteration)
	assert.Contains(t, doc.Content, "Status:")
	assert.Contains(t, doc.Content, "Priority:")
	assert.Contains(t, doc.Content, "Assignee:")
	assert.Contains(t, doc.Content, "DueDate:")
	assert.Contains(t, doc.Content, "Tags:")
	assert.Contains(t, doc.Content, "Completed:")

	// Check that non-prop fields don't appear in properties section
	assert.NotContains(t, doc.Content, "**non_prop_field:**")
	assert.NotContains(t, doc.Content, "should not appear in properties")
}

func TestDatabaseItemNormaliser_Normalise_WithoutProperties(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Content only, no properties."),
		Metadata: map[string]any{
			"title":  "Simple Item",
			"author": "John", // Not a prop_ field
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Should not have Properties section if no prop_ fields
	assert.NotContains(t, doc.Content, "## Properties")
}

func TestDatabaseItemNormaliser_Normalise_WithComments(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Main content."),
		Metadata: map[string]any{
			"title":       "Item with Comments",
			"prop_Status": "Review",
			"comments": []string{
				"Looks good!",
				"Please update the timeline",
				"Approved",
			},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.Contains(t, doc.Content, "## Comments")
	assert.Contains(t, doc.Content, "- Looks good!")
	assert.Contains(t, doc.Content, "- Please update the timeline")
	assert.Contains(t, doc.Content, "- Approved")
	assert.Contains(t, doc.Content, "---") // Separator before comments
}

func TestDatabaseItemNormaliser_Normalise_WithEmptyComments(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Main content."),
		Metadata: map[string]any{
			"title":    "Item",
			"comments": []string{},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	assert.NotContains(t, doc.Content, "## Comments")
	assert.NotContains(t, doc.Content, "---")
}

func TestDatabaseItemNormaliser_Normalise_MetadataPreserved(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Content"),
		Metadata: map[string]any{
			"title":          "Task",
			"prop_Status":    "Done",
			"author":         "Jane Doe",
			"created_at":     "2024-01-01",
			"database_id":    "db-123",
			"parent_page_id": "page-456",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Original metadata should be preserved
	assert.Equal(t, "Jane Doe", doc.Metadata["author"])
	assert.Equal(t, "2024-01-01", doc.Metadata["created_at"])
	assert.Equal(t, "db-123", doc.Metadata["database_id"])
	assert.Equal(t, "page-456", doc.Metadata["parent_page_id"])
	assert.Equal(t, "Done", doc.Metadata["prop_Status"])

	// New metadata should be added
	assert.Equal(t, MIMETypeNotionDBItem, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_database_item", doc.Metadata["format"])
}

func TestDatabaseItemNormaliser_Normalise_MetadataNotMutated(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	originalMetadata := map[string]any{
		"title":  "Item",
		"custom": "value",
	}

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
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

func TestDatabaseItemNormaliser_Normalise_ComplexItem(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	complexContent := `# Implementation Details

This task involves implementing the new authentication system.

## Technical Approach
- Use OAuth 2.0
- Implement JWT tokens
- Add refresh token mechanism

## Considerations
- Security best practices
- Performance optimization
- User experience

## Next Steps
1. Design the API
2. Implement backend
3. Create frontend components
4. Write tests
5. Deploy to staging`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/complex-item",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte(complexContent),
		Metadata: map[string]any{
			"title":            "Implement Authentication System",
			"icon":             "🔐",
			"prop_Status":      "In Progress",
			"prop_Priority":    "Critical",
			"prop_Assignee":    "Engineering Team",
			"prop_DueDate":     "2024-03-01",
			"prop_Sprint":      "Sprint 5",
			"prop_StoryPoints": 13,
			"prop_Tags":        []string{"security", "authentication", "infrastructure"},
			"prop_Blocked":     false,
			"comments": []string{
				"Make sure to follow OWASP guidelines",
				"Consider multi-factor authentication",
				"Review with security team before deployment",
			},
			"database_id": "tasks-db-789",
			"created_by":  "Product Manager",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Implement Authentication System", doc.Title)
	assert.Contains(t, doc.Content, "# Implement Authentication System")
	assert.Contains(t, doc.Content, "🔐")

	// Check properties section
	assert.Contains(t, doc.Content, "## Properties")
	assert.Contains(t, doc.Content, "Status:")
	assert.Contains(t, doc.Content, "Priority:")
	assert.Contains(t, doc.Content, "Assignee:")
	assert.Contains(t, doc.Content, "StoryPoints:")

	// Check content section
	assert.Contains(t, doc.Content, "## Content")
	assert.Contains(t, doc.Content, "Implementation Details")
	assert.Contains(t, doc.Content, "Technical Approach")

	// Check comments
	assert.Contains(t, doc.Content, "## Comments")
	assert.Contains(t, doc.Content, "- Make sure to follow OWASP guidelines")
	assert.Contains(t, doc.Content, "- Consider multi-factor authentication")
	assert.Contains(t, doc.Content, "- Review with security team before deployment")

	// Verify metadata
	assert.Equal(t, "tasks-db-789", doc.Metadata["database_id"])
	assert.Equal(t, "Product Manager", doc.Metadata["created_by"])
	assert.Equal(t, MIMETypeNotionDBItem, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_database_item", doc.Metadata["format"])
}

func TestDatabaseItemNormaliser_Normalise_PropertyValueTypes(t *testing.T) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("Content"),
		Metadata: map[string]any{
			"title":            "Property Test",
			"prop_String":      "text value",
			"prop_Number":      42,
			"prop_Float":       3.14,
			"prop_Boolean":     true,
			"prop_Array":       []string{"tag1", "tag2"},
			"prop_Nil":         nil,
			"prop_EmptyString": "",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Properties section should handle different value types
	assert.Contains(t, doc.Content, "## Properties")

	// Verify different types are formatted correctly
	content := doc.Content
	assert.Contains(t, content, "String:")
	assert.Contains(t, content, "Number:")
	assert.Contains(t, content, "Float:")
	assert.Contains(t, content, "Boolean:")
	assert.Contains(t, content, "Array:")
}

func TestDatabaseItemNormaliser_InterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*DatabaseItemNormaliser)(nil)
}

func BenchmarkDatabaseItemNormaliser_Normalise(b *testing.B) {
	normaliser := NewDatabaseItem()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/page/789",
		MIMEType: MIMETypeNotionDBItem,
		Content:  []byte("This is test content with detailed information."),
		Metadata: map[string]any{
			"title":        "Test Task",
			"icon":         "✅",
			"prop_Status":  "Done",
			"prop_Owner":   "Tester",
			"prop_DueDate": "2024-12-31",
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
