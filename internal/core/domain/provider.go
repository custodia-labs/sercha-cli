package domain

// ProviderType identifies the provider (google, github, slack, etc.).
type ProviderType string

const (
	// ProviderLocal is for local filesystem sources.
	ProviderLocal ProviderType = "local"
	// ProviderGoogle is for Google services (Drive, Gmail, Calendar).
	ProviderGoogle ProviderType = "google"
	// ProviderGitHub is for GitHub repositories and issues.
	ProviderGitHub ProviderType = "github"
	// ProviderSlack is for Slack workspaces.
	ProviderSlack ProviderType = "slack"
	// ProviderNotion is for Notion workspaces.
	ProviderNotion ProviderType = "notion"
	// ProviderMicrosoft is for Microsoft 365 services (Outlook, OneDrive, Calendar).
	ProviderMicrosoft ProviderType = "microsoft"
	// ProviderDropbox is for Dropbox file storage.
	ProviderDropbox ProviderType = "dropbox"
)
