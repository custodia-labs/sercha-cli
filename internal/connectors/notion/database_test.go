package notion

import (
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseToRawDocument(t *testing.T) {
	db := &notionapi.Database{
		ID:          "db-123",
		Title:       []notionapi.RichText{{PlainText: "My Database"}},
		Description: []notionapi.RichText{{PlainText: "A test database"}},
		Properties: map[string]notionapi.PropertyConfig{
			"Name": &notionapi.TitlePropertyConfig{},
		},
		URL: "https://notion.so/db-123",
	}

	doc := DatabaseToRawDocument(db, "source-123")

	require.NotNil(t, doc)
	assert.Equal(t, "source-123", doc.SourceID)
	assert.Equal(t, "notion://databases/db-123", doc.URI)
	assert.Equal(t, MIMETypeNotionDB, doc.MIMEType)
	assert.Contains(t, string(doc.Content), "My Database")
	assert.Equal(t, "My Database", doc.Metadata["title"])
}

func TestExtractDatabaseTitle(t *testing.T) {
	tests := []struct {
		name     string
		db       *notionapi.Database
		expected string
	}{
		{
			name:     "with title",
			db:       &notionapi.Database{Title: []notionapi.RichText{{PlainText: "Test DB"}}},
			expected: "Test DB",
		},
		{
			name:     "empty title",
			db:       &notionapi.Database{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDatabaseTitle(tt.db)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildDatabaseParentURI(t *testing.T) {
	tests := []struct {
		name     string
		db       *notionapi.Database
		expected *string
	}{
		{
			name: "workspace parent",
			db: &notionapi.Database{
				Parent: notionapi.Parent{Type: notionapi.ParentTypeWorkspace},
			},
			expected: nil,
		},
		{
			name: "page parent",
			db: &notionapi.Database{
				Parent: notionapi.Parent{
					Type:   notionapi.ParentTypePageID,
					PageID: "page-123",
				},
			},
			expected: strPtr("notion://pages/page-123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDatabaseParentURI(tt.db)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestExtractPropertyValues_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		props    notionapi.Properties
		expected map[string]any
	}{
		{
			name: "title property",
			props: notionapi.Properties{
				"Name": &notionapi.TitleProperty{
					Title: []notionapi.RichText{{PlainText: "Test Title"}},
				},
			},
			expected: map[string]any{"Name": "Test Title"},
		},
		{
			name: "rich text property",
			props: notionapi.Properties{
				"Description": &notionapi.RichTextProperty{
					RichText: []notionapi.RichText{{PlainText: "Some text"}},
				},
			},
			expected: map[string]any{"Description": "Some text"},
		},
		{
			name: "number property",
			props: notionapi.Properties{
				"Count": &notionapi.NumberProperty{Number: 42},
			},
			expected: map[string]any{"Count": float64(42)},
		},
		{
			name: "checkbox property",
			props: notionapi.Properties{
				"Done": &notionapi.CheckboxProperty{Checkbox: true},
			},
			expected: map[string]any{"Done": true},
		},
		{
			name: "url property",
			props: notionapi.Properties{
				"Link": &notionapi.URLProperty{URL: "https://example.com"},
			},
			expected: map[string]any{"Link": "https://example.com"},
		},
		{
			name: "email property",
			props: notionapi.Properties{
				"Email": &notionapi.EmailProperty{Email: "test@example.com"},
			},
			expected: map[string]any{"Email": "test@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPropertyValues(tt.props)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPropertyValues_SelectTypes(t *testing.T) {
	tests := []struct {
		name     string
		props    notionapi.Properties
		expected map[string]any
	}{
		{
			name: "select property",
			props: notionapi.Properties{
				"Status": &notionapi.SelectProperty{
					Select: notionapi.Option{Name: "Active"},
				},
			},
			expected: map[string]any{"Status": "Active"},
		},
		{
			name: "multi-select property",
			props: notionapi.Properties{
				"Tags": &notionapi.MultiSelectProperty{
					MultiSelect: []notionapi.Option{
						{Name: "Tag1"},
						{Name: "Tag2"},
					},
				},
			},
			expected: map[string]any{"Tags": []string{"Tag1", "Tag2"}},
		},
		{
			name: "status property",
			props: notionapi.Properties{
				"Progress": &notionapi.StatusProperty{
					Status: notionapi.Option{Name: "In Progress"},
				},
			},
			expected: map[string]any{"Progress": "In Progress"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPropertyValues(tt.props)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPropertyValues_RelationProperty(t *testing.T) {
	props := notionapi.Properties{
		"Related": &notionapi.RelationProperty{
			Relation: []notionapi.Relation{
				{ID: "page-1"},
				{ID: "page-2"},
			},
		},
	}

	result := extractPropertyValues(props)
	assert.Equal(t, []string{"page-1", "page-2"}, result["Related"])
}

func strPtr(s string) *string {
	return &s
}
