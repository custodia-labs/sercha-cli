package messages

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// TestQueryChanged tests the QueryChanged message type
func TestQueryChanged(t *testing.T) {
	t.Run("with valid query", func(t *testing.T) {
		msg := QueryChanged{Query: "test query"}
		assert.Equal(t, "test query", msg.Query)
	})

	t.Run("with empty query", func(t *testing.T) {
		msg := QueryChanged{Query: ""}
		assert.Equal(t, "", msg.Query)
	})

	t.Run("with special characters", func(t *testing.T) {
		msg := QueryChanged{Query: "test@#$%^&*()"}
		assert.Equal(t, "test@#$%^&*()", msg.Query)
	})
}

// TestSearchRequested tests the SearchRequested message type
func TestSearchRequested(t *testing.T) {
	t.Run("with hybrid search options", func(t *testing.T) {
		opts := domain.SearchOptions{Limit: 10, Hybrid: true}
		msg := SearchRequested{Query: "search", Options: opts}

		assert.Equal(t, "search", msg.Query)
		assert.Equal(t, 10, msg.Options.Limit)
		assert.True(t, msg.Options.Hybrid)
	})

	t.Run("with semantic search options", func(t *testing.T) {
		opts := domain.SearchOptions{Limit: 50, Semantic: true}
		msg := SearchRequested{Query: "semantic query", Options: opts}

		assert.Equal(t, "semantic query", msg.Query)
		assert.Equal(t, 50, msg.Options.Limit)
		assert.True(t, msg.Options.Semantic)
	})

	t.Run("with source IDs filter", func(t *testing.T) {
		opts := domain.SearchOptions{
			Limit:     25,
			SourceIDs: []string{"src1", "src2", "src3"},
		}
		msg := SearchRequested{Query: "filtered search", Options: opts}

		assert.Equal(t, "filtered search", msg.Query)
		require.Len(t, msg.Options.SourceIDs, 3)
		assert.Contains(t, msg.Options.SourceIDs, "src1")
	})

	t.Run("with offset", func(t *testing.T) {
		opts := domain.SearchOptions{Limit: 10, Offset: 20}
		msg := SearchRequested{Query: "paginated", Options: opts}

		assert.Equal(t, 20, msg.Options.Offset)
		assert.Equal(t, 10, msg.Options.Limit)
	})
}

// TestSearchCompleted tests the SearchCompleted message type
func TestSearchCompleted_WithResults(t *testing.T) {
	results := []domain.SearchResult{
		{Document: domain.Document{Title: "Doc 1"}, Score: 0.9},
		{Document: domain.Document{Title: "Doc 2"}, Score: 0.8},
	}
	msg := SearchCompleted{Results: results, Err: nil}

	assert.Len(t, msg.Results, 2)
	assert.NoError(t, msg.Err)
}

func TestSearchCompleted_WithError(t *testing.T) {
	err := errors.New("search failed")
	msg := SearchCompleted{Results: nil, Err: err}

	assert.Nil(t, msg.Results)
	assert.Error(t, msg.Err)
	assert.Equal(t, "search failed", msg.Err.Error())
}

func TestSearchCompleted_EmptyResults(t *testing.T) {
	msg := SearchCompleted{Results: []domain.SearchResult{}, Err: nil}

	assert.NotNil(t, msg.Results)
	assert.Empty(t, msg.Results)
	assert.NoError(t, msg.Err)
}

// TestResultSelected tests the ResultSelected message type
func TestResultSelected(t *testing.T) {
	t.Run("with positive index", func(t *testing.T) {
		msg := ResultSelected{Index: 5}
		assert.Equal(t, 5, msg.Index)
	})

	t.Run("with zero index", func(t *testing.T) {
		msg := ResultSelected{Index: 0}
		assert.Equal(t, 0, msg.Index)
	})

	t.Run("with negative index", func(t *testing.T) {
		msg := ResultSelected{Index: -1}
		assert.Equal(t, -1, msg.Index)
	})
}

// TestViewChanged tests the ViewChanged message type
func TestViewChanged(t *testing.T) {
	t.Run("to sources view", func(t *testing.T) {
		msg := ViewChanged{View: ViewSources}
		assert.Equal(t, ViewSources, msg.View)
	})

	t.Run("to search view", func(t *testing.T) {
		msg := ViewChanged{View: ViewSearch}
		assert.Equal(t, ViewSearch, msg.View)
	})

	t.Run("to help view", func(t *testing.T) {
		msg := ViewChanged{View: ViewHelp}
		assert.Equal(t, ViewHelp, msg.View)
	})
}

// TestViewType_String tests all ViewType string representations
func TestViewType_String(t *testing.T) {
	tests := []struct {
		name     string
		view     ViewType
		expected string
	}{
		{"ViewMenu", ViewMenu, "menu"},
		{"ViewSearch", ViewSearch, "search"},
		{"ViewSources", ViewSources, "sources"},
		{"ViewHelp", ViewHelp, "help"},
		{"ViewSourceDetail", ViewSourceDetail, "source_detail"},
		{"ViewDocuments", ViewDocuments, "documents"},
		{"ViewDocContent", ViewDocContent, "doc_content"},
		{"ViewDocDetails", ViewDocDetails, "doc_details"},
		{"ViewAddSource", ViewAddSource, "add_source"},
		{"ViewSettings", ViewSettings, "settings"},
		{"UnknownView", ViewType(99), "unknown"},
		{"NegativeView", ViewType(-1), "unknown"},
		{"LargeView", ViewType(1000), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.view.String())
		})
	}
}

// TestErrorOccurred tests the ErrorOccurred message type
func TestErrorOccurred(t *testing.T) {
	t.Run("with standard error", func(t *testing.T) {
		err := errors.New("something went wrong")
		msg := ErrorOccurred{Err: err}

		assert.Error(t, msg.Err)
		assert.Equal(t, "something went wrong", msg.Err.Error())
	})

	t.Run("with nil error", func(t *testing.T) {
		msg := ErrorOccurred{Err: nil}
		assert.Nil(t, msg.Err)
	})

	t.Run("with wrapped error", func(t *testing.T) {
		baseErr := errors.New("base error")
		wrappedErr := errors.Join(baseErr, errors.New("additional context"))
		msg := ErrorOccurred{Err: wrappedErr}

		assert.Error(t, msg.Err)
		assert.Contains(t, msg.Err.Error(), "base error")
	})
}

// TestQuit tests the Quit message type
func TestQuit(t *testing.T) {
	msg := Quit{}
	// Quit is an empty struct, just verify it can be created
	assert.NotNil(t, msg)
}

// TestSourcesLoaded tests the SourcesLoaded message type
func TestSourcesLoaded(t *testing.T) {
	t.Run("with sources", func(t *testing.T) {
		sources := []domain.Source{
			{ID: "src1", Name: "Source 1", Type: "filesystem"},
			{ID: "src2", Name: "Source 2", Type: "gmail"},
		}
		msg := SourcesLoaded{Sources: sources, Err: nil}

		require.Len(t, msg.Sources, 2)
		assert.Equal(t, "src1", msg.Sources[0].ID)
		assert.Equal(t, "Source 2", msg.Sources[1].Name)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("failed to load sources")
		msg := SourcesLoaded{Sources: nil, Err: err}

		assert.Nil(t, msg.Sources)
		assert.Error(t, msg.Err)
		assert.Equal(t, "failed to load sources", msg.Err.Error())
	})

	t.Run("with empty sources list", func(t *testing.T) {
		msg := SourcesLoaded{Sources: []domain.Source{}, Err: nil}

		assert.NotNil(t, msg.Sources)
		assert.Empty(t, msg.Sources)
		assert.NoError(t, msg.Err)
	})
}

// TestSourceAdded tests the SourceAdded message type
func TestSourceAdded(t *testing.T) {
	t.Run("successful addition", func(t *testing.T) {
		source := domain.Source{
			ID:   "new-src",
			Name: "New Source",
			Type: "notion",
		}
		msg := SourceAdded{Source: source, Err: nil}

		assert.Equal(t, "new-src", msg.Source.ID)
		assert.Equal(t, "New Source", msg.Source.Name)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("source already exists")
		msg := SourceAdded{Source: domain.Source{}, Err: err}

		assert.Error(t, msg.Err)
		assert.Equal(t, "source already exists", msg.Err.Error())
	})
}

// TestSourceRemoved tests the SourceRemoved message type
func TestSourceRemoved(t *testing.T) {
	t.Run("successful removal", func(t *testing.T) {
		msg := SourceRemoved{ID: "src-123", Err: nil}

		assert.Equal(t, "src-123", msg.ID)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("source not found")
		msg := SourceRemoved{ID: "src-456", Err: err}

		assert.Equal(t, "src-456", msg.ID)
		assert.Error(t, msg.Err)
		assert.Equal(t, "source not found", msg.Err.Error())
	})

	t.Run("with empty ID", func(t *testing.T) {
		msg := SourceRemoved{ID: "", Err: nil}
		assert.Equal(t, "", msg.ID)
	})
}

// TestSourceSelected tests the SourceSelected message type
func TestSourceSelected(t *testing.T) {
	t.Run("with valid source", func(t *testing.T) {
		source := domain.Source{
			ID:   "selected-src",
			Name: "Selected Source",
			Type: "github",
		}
		msg := SourceSelected{Source: source}

		assert.Equal(t, "selected-src", msg.Source.ID)
		assert.Equal(t, "Selected Source", msg.Source.Name)
		assert.Equal(t, "github", msg.Source.Type)
	})

	t.Run("with empty source", func(t *testing.T) {
		msg := SourceSelected{Source: domain.Source{}}
		assert.Equal(t, "", msg.Source.ID)
	})
}

// TestDocumentsLoaded tests the DocumentsLoaded message type
func TestDocumentsLoaded(t *testing.T) {
	t.Run("with documents", func(t *testing.T) {
		docs := []domain.Document{
			{ID: "doc1", Title: "Document 1", SourceID: "src1"},
			{ID: "doc2", Title: "Document 2", SourceID: "src1"},
		}
		msg := DocumentsLoaded{
			SourceID:  "src1",
			Documents: docs,
			Err:       nil,
		}

		assert.Equal(t, "src1", msg.SourceID)
		require.Len(t, msg.Documents, 2)
		assert.Equal(t, "doc1", msg.Documents[0].ID)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("failed to load documents")
		msg := DocumentsLoaded{
			SourceID:  "src2",
			Documents: nil,
			Err:       err,
		}

		assert.Equal(t, "src2", msg.SourceID)
		assert.Nil(t, msg.Documents)
		assert.Error(t, msg.Err)
	})

	t.Run("with empty documents", func(t *testing.T) {
		msg := DocumentsLoaded{
			SourceID:  "src3",
			Documents: []domain.Document{},
			Err:       nil,
		}

		assert.NotNil(t, msg.Documents)
		assert.Empty(t, msg.Documents)
	})
}

// TestDocumentSelected tests the DocumentSelected message type
func TestDocumentSelected(t *testing.T) {
	t.Run("with valid document", func(t *testing.T) {
		doc := domain.Document{
			ID:       "doc-123",
			Title:    "Selected Document",
			SourceID: "src-1",
		}
		msg := DocumentSelected{Document: doc}

		assert.Equal(t, "doc-123", msg.Document.ID)
		assert.Equal(t, "Selected Document", msg.Document.Title)
	})

	t.Run("with empty document", func(t *testing.T) {
		msg := DocumentSelected{Document: domain.Document{}}
		assert.Equal(t, "", msg.Document.ID)
	})
}

// TestDocumentContentLoaded tests the DocumentContentLoaded message type
func TestDocumentContentLoaded(t *testing.T) {
	t.Run("with content", func(t *testing.T) {
		msg := DocumentContentLoaded{
			DocumentID: "doc-123",
			Content:    "This is the document content",
			Err:        nil,
		}

		assert.Equal(t, "doc-123", msg.DocumentID)
		assert.Equal(t, "This is the document content", msg.Content)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("content not found")
		msg := DocumentContentLoaded{
			DocumentID: "doc-456",
			Content:    "",
			Err:        err,
		}

		assert.Equal(t, "doc-456", msg.DocumentID)
		assert.Equal(t, "", msg.Content)
		assert.Error(t, msg.Err)
	})

	t.Run("with empty content", func(t *testing.T) {
		msg := DocumentContentLoaded{
			DocumentID: "doc-789",
			Content:    "",
			Err:        nil,
		}

		assert.Equal(t, "", msg.Content)
		assert.NoError(t, msg.Err)
	})
}

// TestDocumentDetailsLoaded tests the DocumentDetailsLoaded message type
func TestDocumentDetailsLoaded(t *testing.T) {
	t.Run("with details", func(t *testing.T) {
		details := map[string]interface{}{
			"author": "John Doe",
			"date":   "2024-01-01",
		}
		msg := DocumentDetailsLoaded{
			DocumentID: "doc-123",
			Details:    details,
			Err:        nil,
		}

		assert.Equal(t, "doc-123", msg.DocumentID)
		assert.NotNil(t, msg.Details)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("details unavailable")
		msg := DocumentDetailsLoaded{
			DocumentID: "doc-456",
			Details:    nil,
			Err:        err,
		}

		assert.Nil(t, msg.Details)
		assert.Error(t, msg.Err)
	})

	t.Run("with nil details", func(t *testing.T) {
		msg := DocumentDetailsLoaded{
			DocumentID: "doc-789",
			Details:    nil,
			Err:        nil,
		}

		assert.Nil(t, msg.Details)
		assert.NoError(t, msg.Err)
	})
}

// TestDocumentExcluded tests the DocumentExcluded message type
func TestDocumentExcluded(t *testing.T) {
	t.Run("successful exclusion", func(t *testing.T) {
		msg := DocumentExcluded{
			DocumentID: "doc-exclude",
			Err:        nil,
		}

		assert.Equal(t, "doc-exclude", msg.DocumentID)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("exclusion failed")
		msg := DocumentExcluded{
			DocumentID: "doc-fail",
			Err:        err,
		}

		assert.Equal(t, "doc-fail", msg.DocumentID)
		assert.Error(t, msg.Err)
	})
}

// TestDocumentRefreshed tests the DocumentRefreshed message type
func TestDocumentRefreshed(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		msg := DocumentRefreshed{
			DocumentID: "doc-refresh",
			Err:        nil,
		}

		assert.Equal(t, "doc-refresh", msg.DocumentID)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("refresh failed")
		msg := DocumentRefreshed{
			DocumentID: "doc-fail",
			Err:        err,
		}

		assert.Equal(t, "doc-fail", msg.DocumentID)
		assert.Error(t, msg.Err)
		assert.Equal(t, "refresh failed", msg.Err.Error())
	})
}

// TestOAuthFlowCompleted tests the OAuthFlowCompleted message type
func TestOAuthFlowCompleted(t *testing.T) {
	t.Run("successful OAuth flow", func(t *testing.T) {
		msg := OAuthFlowCompleted{
			CredentialsID: "oauth-123",
			Err:           nil,
		}

		assert.Equal(t, "oauth-123", msg.CredentialsID)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("OAuth flow failed")
		msg := OAuthFlowCompleted{
			CredentialsID: "oauth-456",
			Err:           err,
		}

		assert.Equal(t, "oauth-456", msg.CredentialsID)
		assert.Error(t, msg.Err)
		assert.Equal(t, "OAuth flow failed", msg.Err.Error())
	})

	t.Run("with empty CredentialsID", func(t *testing.T) {
		msg := OAuthFlowCompleted{
			CredentialsID: "",
			Err:           errors.New("no credentials ID"),
		}

		assert.Equal(t, "", msg.CredentialsID)
		assert.Error(t, msg.Err)
	})
}

// TestSettingsLoaded tests the SettingsLoaded message type
func TestSettingsLoaded(t *testing.T) {
	t.Run("with settings", func(t *testing.T) {
		settings := &domain.AppSettings{
			Search: domain.SearchSettings{
				Mode: domain.SearchModeHybrid,
			},
		}
		msg := SettingsLoaded{
			Settings: settings,
			Err:      nil,
		}

		assert.NotNil(t, msg.Settings)
		assert.Equal(t, domain.SearchModeHybrid, msg.Settings.Search.Mode)
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("failed to load settings")
		msg := SettingsLoaded{
			Settings: nil,
			Err:      err,
		}

		assert.Nil(t, msg.Settings)
		assert.Error(t, msg.Err)
		assert.Equal(t, "failed to load settings", msg.Err.Error())
	})

	t.Run("with nil settings", func(t *testing.T) {
		msg := SettingsLoaded{
			Settings: nil,
			Err:      nil,
		}

		assert.Nil(t, msg.Settings)
		assert.NoError(t, msg.Err)
	})
}

// TestSettingsSaved tests the SettingsSaved message type
func TestSettingsSaved(t *testing.T) {
	t.Run("successful save", func(t *testing.T) {
		msg := SettingsSaved{Err: nil}
		assert.NoError(t, msg.Err)
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("save failed")
		msg := SettingsSaved{Err: err}

		assert.Error(t, msg.Err)
		assert.Equal(t, "save failed", msg.Err.Error())
	})
}
