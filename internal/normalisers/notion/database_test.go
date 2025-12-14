package notion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNewDatabase(t *testing.T) {
	normaliser := NewDatabase()
	require.NotNil(t, normaliser)
	assert.IsType(t, &DatabaseNormaliser{}, normaliser)
}

func TestDatabaseNormaliser_SupportedMIMETypes(t *testing.T) {
	normaliser := NewDatabase()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, MIMETypeNotionDB)
	assert.Equal(t, "application/vnd.notion.database+json", MIMETypeNotionDB)
	assert.Len(t, mimeTypes, 1)
}

func TestDatabaseNormaliser_SupportedConnectorTypes(t *testing.T) {
	normaliser := NewDatabase()
	connectorTypes := normaliser.SupportedConnectorTypes()

	require.NotEmpty(t, connectorTypes)
	assert.Contains(t, connectorTypes, "notion")
	assert.Len(t, connectorTypes, 1)
}

func TestDatabaseNormaliser_Priority(t *testing.T) {
	normaliser := NewDatabase()
	assert.Equal(t, 95, normaliser.Priority())
}

func TestDatabaseNormaliser_Normalise_Success(t *testing.T) {
	normaliser := NewDatabase()
	ctx := context.Background()

	dbContent := `# Project Tracker

## Description
Track all ongoing projects and their status.

## Properties
- Name: Title
- Status: Select
- Due Date: Date
- Owner: Person`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/database/456",
		MIMEType: MIMETypeNotionDB,
		Content:  []byte(dbContent),
		Metadata: map[string]any{
			"title": "Project Tracker",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, raw.SourceID, doc.SourceID)
	assert.Equal(t, raw.URI, doc.URI)
	assert.Equal(t, "Project Tracker", doc.Title)
	assert.Equal(t, dbContent, doc.Content)
	assert.NotNil(t, doc.Metadata)
	assert.Equal(t, MIMETypeNotionDB, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_database", doc.Metadata["format"])
	assert.NotZero(t, doc.CreatedAt)
	assert.NotZero(t, doc.UpdatedAt)
}

func TestDatabaseNormaliser_Normalise_NilDocument(t *testing.T) {
	normaliser := NewDatabase()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestDatabaseNormaliser_Normalise_EmptyContent(t *testing.T) {
	normaliser := NewDatabase()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/database/456",
		MIMEType: MIMETypeNotionDB,
		Content:  []byte(""),
		Metadata: map[string]any{
			"title": "Empty Database",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Empty Database", doc.Title)
	assert.Empty(t, doc.Content)
}

func TestDatabaseNormaliser_Normalise_UntitledFallback(t *testing.T) {
	tests := []struct {
		name          string
		metadata      map[string]any
		expectedTitle string
	}{
		{
			name:          "no title in metadata",
			metadata:      map[string]any{},
			expectedTitle: "Untitled Database",
		},
		{
			name: "empty title string",
			metadata: map[string]any{
				"title": "",
			},
			expectedTitle: "Untitled Database",
		},
		{
			name: "title not a string",
			metadata: map[string]any{
				"title": 123,
			},
			expectedTitle: "Untitled Database",
		},
		{
			name:          "nil metadata",
			metadata:      nil,
			expectedTitle: "Untitled Database",
		},
	}

	normaliser := NewDatabase()
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := &domain.RawDocument{
				SourceID: "test-source",
				URI:      "https://notion.so/database/456",
				MIMEType: MIMETypeNotionDB,
				Content:  []byte("Database content"),
				Metadata: tc.metadata,
			}

			result, err := normaliser.Normalise(ctx, raw)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTitle, result.Document.Title)
		})
	}
}

func TestDatabaseNormaliser_Normalise_MetadataPreserved(t *testing.T) {
	normaliser := NewDatabase()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/database/456",
		MIMEType: MIMETypeNotionDB,
		Content:  []byte("Database schema"),
		Metadata: map[string]any{
			"title":       "Tasks",
			"description": "Team task tracker",
			"created_at":  "2024-01-01",
			"properties":  []string{"Name", "Status", "Priority"},
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)

	doc := result.Document
	// Original metadata should be preserved
	assert.Equal(t, "Team task tracker", doc.Metadata["description"])
	assert.Equal(t, "2024-01-01", doc.Metadata["created_at"])
	assert.Equal(t, []string{"Name", "Status", "Priority"}, doc.Metadata["properties"])

	// New metadata should be added
	assert.Equal(t, MIMETypeNotionDB, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_database", doc.Metadata["format"])
}

func TestDatabaseNormaliser_Normalise_MetadataNotMutated(t *testing.T) {
	normaliser := NewDatabase()
	ctx := context.Background()

	originalMetadata := map[string]any{
		"title":  "Database",
		"custom": "value",
	}

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/database/456",
		MIMEType: MIMETypeNotionDB,
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

func TestDatabaseNormaliser_Normalise_ComplexDatabase(t *testing.T) {
	normaliser := NewDatabase()
	ctx := context.Background()

	complexContent := `# Customer Database

## Description
Comprehensive customer relationship management database.

## Properties

### Basic Information
- **Name**: Title (required)
- **Email**: Email
- **Phone**: Phone Number
- **Company**: Text

### Relationship Data
- **Account Manager**: Person
- **Status**: Select (Active, Inactive, Pending)
- **Type**: Multi-select (Enterprise, SMB, Startup)

### Metrics
- **Revenue**: Number (currency)
- **Contract Start**: Date
- **Contract End**: Date
- **Renewal Date**: Date

### Additional
- **Notes**: Long Text
- **Tags**: Multi-select
- **Last Contact**: Date`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/database/complex",
		MIMEType: MIMETypeNotionDB,
		Content:  []byte(complexContent),
		Metadata: map[string]any{
			"title":       "Customer Database",
			"description": "Comprehensive customer relationship management database",
			"created_at":  "2024-01-15",
			"owner":       "Sales Team",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.Equal(t, "Customer Database", doc.Title)
	assert.Equal(t, complexContent, doc.Content)
	assert.Contains(t, doc.Content, "Properties")
	assert.Contains(t, doc.Content, "Basic Information")
	assert.Contains(t, doc.Content, "Revenue")

	// Verify metadata
	assert.Equal(t, "Sales Team", doc.Metadata["owner"])
	assert.Equal(t, MIMETypeNotionDB, doc.Metadata["mime_type"])
	assert.Equal(t, "notion_database", doc.Metadata["format"])
}

func TestDatabaseNormaliser_InterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*DatabaseNormaliser)(nil)
}

func BenchmarkDatabaseNormaliser_Normalise(b *testing.B) {
	normaliser := NewDatabase()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "https://notion.so/database/456",
		MIMEType: MIMETypeNotionDB,
		Content:  []byte("Database schema with properties"),
		Metadata: map[string]any{
			"title":       "Tasks",
			"description": "Task tracker",
			"properties":  []string{"Name", "Status"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normaliser.Normalise(ctx, raw)
	}
}
