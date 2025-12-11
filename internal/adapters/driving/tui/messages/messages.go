// Package messages defines Bubbletea message types for the TUI.
// Messages represent events and commands that flow through the Elm architecture.
package messages

import (
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// QueryChanged is sent when the search query input changes.
type QueryChanged struct {
	Query string
}

// SearchRequested is a command to perform a search.
type SearchRequested struct {
	Query   string
	Options domain.SearchOptions
}

// SearchCompleted carries search results back to the model.
type SearchCompleted struct {
	Results []domain.SearchResult
	Err     error
}

// ResultSelected is sent when a search result is selected.
type ResultSelected struct {
	Index int
}

// ViewChanged is sent when navigating between views.
type ViewChanged struct {
	View ViewType
}

// ViewType identifies which view is currently active.
type ViewType int

const (
	// ViewMenu is the main navigation menu.
	ViewMenu ViewType = iota
	// ViewSearch is the search input and results view.
	ViewSearch
	// ViewSources is the source management view.
	ViewSources
	// ViewHelp is the help/keybindings view.
	ViewHelp
	// ViewSourceDetail shows details for a single source.
	ViewSourceDetail
	// ViewDocuments lists documents for a source.
	ViewDocuments
	// ViewDocContent shows document content.
	ViewDocContent
	// ViewDocDetails shows document metadata.
	ViewDocDetails
	// ViewAddSource is the add source wizard.
	ViewAddSource
	// ViewSettings is the settings configuration view.
	ViewSettings
)

// String returns the string representation of the view type.
func (v ViewType) String() string {
	switch v {
	case ViewMenu:
		return "menu"
	case ViewSearch:
		return "search"
	case ViewSources:
		return "sources"
	case ViewHelp:
		return "help"
	case ViewSourceDetail:
		return "source_detail"
	case ViewDocuments:
		return "documents"
	case ViewDocContent:
		return "doc_content"
	case ViewDocDetails:
		return "doc_details"
	case ViewAddSource:
		return "add_source"
	case ViewSettings:
		return "settings"
	default:
		return "unknown"
	}
}

// ErrorOccurred signals that an error happened.
type ErrorOccurred struct {
	Err error
}

// Quit signals the application should exit.
type Quit struct{}

// SourcesLoaded carries the list of sources from the service.
type SourcesLoaded struct {
	Sources []domain.Source
	Err     error
}

// SourceAdded signals a source was added.
type SourceAdded struct {
	Source domain.Source
	Err    error
}

// SourceRemoved signals a source was removed.
type SourceRemoved struct {
	ID  string
	Err error
}

// SourceSelected signals a source was selected for detail view.
type SourceSelected struct {
	Source domain.Source
}

// DocumentsLoaded carries the list of documents for a source.
type DocumentsLoaded struct {
	SourceID  string
	Documents []domain.Document
	Err       error
}

// DocumentSelected signals a document was selected.
type DocumentSelected struct {
	Document domain.Document
}

// DocumentContentLoaded carries the content of a document.
type DocumentContentLoaded struct {
	DocumentID string
	Content    string
	Err        error
}

// DocumentDetailsLoaded carries the metadata of a document.
type DocumentDetailsLoaded struct {
	DocumentID string
	Details    interface{} // *driving.DocumentDetails
	Err        error
}

// DocumentExcluded signals a document was excluded.
type DocumentExcluded struct {
	DocumentID string
	Err        error
}

// DocumentRefreshed signals a document refresh completed.
type DocumentRefreshed struct {
	DocumentID string
	Err        error
}

// AuthProvidersLoaded carries the list of OAuth app configurations.
type AuthProvidersLoaded struct {
	AuthProviders []domain.AuthProvider
	Err           error
}

// AuthProviderCreated signals an OAuth app configuration was created.
type AuthProviderCreated struct {
	AuthProvider domain.AuthProvider
	Err          error
}

// AuthProviderDeleted signals an OAuth app configuration was deleted.
type AuthProviderDeleted struct {
	ID  string
	Err error
}

// OAuthFlowCompleted signals OAuth flow finished.
type OAuthFlowCompleted struct {
	CredentialsID string
	Err           error
}

// SettingsLoaded carries the application settings.
type SettingsLoaded struct {
	Settings *domain.AppSettings
	Err      error
}

// SettingsSaved signals settings were saved.
type SettingsSaved struct {
	Err error
}
