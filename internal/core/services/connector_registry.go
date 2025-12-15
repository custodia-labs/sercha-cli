package services

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/connectors/dropbox"
	"github.com/custodia-labs/sercha-cli/internal/connectors/filesystem"
	"github.com/custodia-labs/sercha-cli/internal/connectors/github"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google/calendar"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google/drive"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google/gmail"
	mscalendar "github.com/custodia-labs/sercha-cli/internal/connectors/microsoft/calendar"
	"github.com/custodia-labs/sercha-cli/internal/connectors/microsoft/onedrive"
	"github.com/custodia-labs/sercha-cli/internal/connectors/microsoft/outlook"
	"github.com/custodia-labs/sercha-cli/internal/connectors/notion"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ensure ConnectorRegistry implements the interface.
var _ driving.ConnectorRegistry = (*ConnectorRegistry)(nil)

// ConnectorRegistry provides information about available connector types.
type ConnectorRegistry struct {
	connectors       map[string]domain.ConnectorType
	connectorFactory driven.ConnectorFactory
}

// NewConnectorRegistry creates a new connector registry with built-in connectors.
func NewConnectorRegistry(connectorFactory driven.ConnectorFactory) *ConnectorRegistry {
	r := &ConnectorRegistry{
		connectors:       make(map[string]domain.ConnectorType),
		connectorFactory: connectorFactory,
	}
	r.registerBuiltinConnectors()
	return r
}

func (r *ConnectorRegistry) registerBuiltinConnectors() {
	r.registerFilesystem()
	r.registerGitHub()
	r.registerGoogleDrive()
	r.registerGmail()
	r.registerGoogleCalendar()
	r.registerOutlook()
	r.registerOneDrive()
	r.registerMicrosoftCalendar()
	r.registerDropbox()
	r.registerNotion()
}

func (r *ConnectorRegistry) registerFilesystem() {
	r.connectors["filesystem"] = domain.ConnectorType{
		ID:             "filesystem",
		Name:           "Local Filesystem",
		Description:    "Index files from a local directory",
		ProviderType:   domain.ProviderLocal,
		AuthCapability: domain.AuthCapNone,
		AuthMethod:     domain.AuthMethodNone,
		ConfigKeys:     filesystemConfigKeys(),
		WebURLResolver: filesystem.ResolveWebURL,
	}
}

func filesystemConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "path",
			Label:       "Directory Path",
			Description: "Path to the directory to index",
			Required:    true,
		},
		{
			Key:         "patterns",
			Label:       "File Patterns",
			Description: "Glob patterns to match (e.g., *.md,*.txt)",
		},
	}
}

func (r *ConnectorRegistry) registerGitHub() {
	r.connectors["github"] = domain.ConnectorType{
		ID:             "github",
		Name:           "GitHub",
		Description:    "Index repositories, issues, PRs, and wikis from GitHub",
		ProviderType:   domain.ProviderGitHub,
		AuthCapability: domain.AuthCapPAT | domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodPAT,
		ConfigKeys:     githubConfigKeys(),
		WebURLResolver: github.ResolveWebURL,
	}
}

func githubConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "content_types",
			Label:       "Content Types",
			Description: "Content to index: files,issues,prs,wikis",
			Default:     "files",
		},
		{
			Key:         "file_patterns",
			Label:       "File Patterns",
			Description: "Glob patterns for files to include",
			Default:     "*",
		},
	}
}

func (r *ConnectorRegistry) registerGoogleDrive() {
	r.connectors["google-drive"] = domain.ConnectorType{
		ID:             "google-drive",
		Name:           "Google Drive",
		Description:    "Index documents from Google Drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     driveConfigKeys(),
		WebURLResolver: drive.ResolveWebURL,
	}
}

func driveConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "content_types",
			Label:       "Content Types",
			Description: "Content to sync: files,docs,sheets",
			Default:     "files,docs,sheets",
		},
		{
			Key:         "folder_ids",
			Label:       "Folder IDs",
			Description: "Specific folder IDs to sync (optional)",
		},
		{
			Key:         "mime_types",
			Label:       "MIME Types",
			Description: "Filter by MIME types (optional)",
		},
	}
}

func (r *ConnectorRegistry) registerGmail() {
	r.connectors["gmail"] = domain.ConnectorType{
		ID:             "gmail",
		Name:           "Gmail",
		Description:    "Index emails from Gmail",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     gmailConfigKeys(),
		WebURLResolver: gmail.ResolveWebURL,
	}
}

func gmailConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "label_ids",
			Label:       "Label IDs",
			Description: "Labels to sync: INBOX,SENT,etc",
			Default:     "INBOX",
		},
		{
			Key:         "query",
			Label:       "Search Query",
			Description: "Gmail search query to filter emails",
		},
		{
			Key:         "include_spam_trash",
			Label:       "Include Spam/Trash",
			Description: "Include spam and trash (true/false)",
			Default:     "false",
		},
	}
}

func (r *ConnectorRegistry) registerGoogleCalendar() {
	r.connectors["google-calendar"] = domain.ConnectorType{
		ID:             "google-calendar",
		Name:           "Google Calendar",
		Description:    "Index events from Google Calendar",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     calendarConfigKeys(),
		WebURLResolver: calendar.ResolveWebURL,
	}
}

func calendarConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "calendar_ids",
			Label:       "Calendar IDs",
			Description: "Specific calendar IDs to sync (optional)",
		},
		{
			Key:         "single_events",
			Label:       "Expand Recurring",
			Description: "Expand recurring events (true/false)",
			Default:     "true",
		},
	}
}

func (r *ConnectorRegistry) registerOutlook() {
	r.connectors["outlook"] = domain.ConnectorType{
		ID:             "outlook",
		Name:           "Outlook",
		Description:    "Index emails from Microsoft Outlook",
		ProviderType:   domain.ProviderMicrosoft,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     outlookConfigKeys(),
		WebURLResolver: outlook.ResolveWebURL,
	}
}

func outlookConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "folder_ids",
			Label:       "Folder IDs",
			Description: "Folder IDs to sync (optional, defaults to Inbox)",
		},
		{
			Key:         "query",
			Label:       "Search Query",
			Description: "OData filter query to filter emails",
		},
	}
}

func (r *ConnectorRegistry) registerOneDrive() {
	r.connectors["onedrive"] = domain.ConnectorType{
		ID:             "onedrive",
		Name:           "OneDrive",
		Description:    "Index files from Microsoft OneDrive",
		ProviderType:   domain.ProviderMicrosoft,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     onedriveConfigKeys(),
		WebURLResolver: onedrive.ResolveWebURL,
	}
}

func onedriveConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "folder_path",
			Label:       "Folder Path",
			Description: "Path to folder to sync (optional, defaults to root)",
		},
	}
}

func (r *ConnectorRegistry) registerMicrosoftCalendar() {
	r.connectors["microsoft-calendar"] = domain.ConnectorType{
		ID:             "microsoft-calendar",
		Name:           "Microsoft Calendar",
		Description:    "Index events from Microsoft Calendar",
		ProviderType:   domain.ProviderMicrosoft,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     msCalendarConfigKeys(),
		WebURLResolver: mscalendar.ResolveWebURL,
	}
}

func msCalendarConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "calendar_ids",
			Label:       "Calendar IDs",
			Description: "Specific calendar IDs to sync (optional)",
		},
	}
}

func (r *ConnectorRegistry) registerDropbox() {
	r.connectors["dropbox"] = domain.ConnectorType{
		ID:             "dropbox",
		Name:           "Dropbox",
		Description:    "Index files from Dropbox",
		ProviderType:   domain.ProviderDropbox,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     dropboxConfigKeys(),
		WebURLResolver: dropbox.ResolveWebURL,
	}
}

func dropboxConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "folder_path",
			Label:       "Folder Path",
			Description: "Root folder path to sync (optional, defaults to root)",
		},
		{
			Key:         "recursive",
			Label:       "Recursive",
			Description: "Include subfolders (true/false)",
			Default:     "true",
		},
		{
			Key:         "mime_types",
			Label:       "MIME Types",
			Description: "Filter by MIME types (optional)",
		},
	}
}

func (r *ConnectorRegistry) registerNotion() {
	r.connectors["notion"] = domain.ConnectorType{
		ID:             "notion",
		Name:           "Notion",
		Description:    "Index pages and databases from Notion",
		ProviderType:   domain.ProviderNotion,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
		ConfigKeys:     notionConfigKeys(),
		WebURLResolver: notion.ResolveWebURL,
	}
}

func notionConfigKeys() []domain.ConfigKey {
	return []domain.ConfigKey{
		{
			Key:         "include_comments",
			Label:       "Include Comments",
			Description: "Fetch page comments (true/false)",
			Default:     "true",
		},
		{
			Key:         "content_types",
			Label:       "Content Types",
			Description: "Content to sync: pages,databases",
			Default:     "pages,databases",
		},
		{
			Key:         "max_block_depth",
			Label:       "Max Block Depth",
			Description: "Maximum depth for recursive block fetching",
			Default:     "10",
		},
		{
			Key:         "page_size",
			Label:       "Page Size",
			Description: "Items per API page (max: 100)",
			Default:     "100",
		},
	}
}

// List returns all available connector types.
func (r *ConnectorRegistry) List() []domain.ConnectorType {
	result := make([]domain.ConnectorType, 0, len(r.connectors))
	for _, c := range r.connectors {
		result = append(result, c)
	}
	return result
}

// GetConnectorsForProvider returns all connector types for a given provider.
func (r *ConnectorRegistry) GetConnectorsForProvider(provider domain.ProviderType) []domain.ConnectorType {
	var result []domain.ConnectorType
	for _, c := range r.connectors {
		if c.ProviderType == provider {
			result = append(result, c)
		}
	}
	return result
}

// Get returns a specific connector type by ID.
func (r *ConnectorRegistry) Get(id string) (*domain.ConnectorType, error) {
	c, ok := r.connectors[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &c, nil
}

// ValidateConfig validates configuration for a connector type.
func (r *ConnectorRegistry) ValidateConfig(connectorID string, config map[string]string) error {
	connector, ok := r.connectors[connectorID]
	if !ok {
		return domain.ErrNotFound
	}

	for _, key := range connector.ConfigKeys {
		if key.Required {
			val, exists := config[key.Key]
			if !exists || val == "" {
				return domain.ErrInvalidInput
			}
		}
	}
	return nil
}

// GetOAuthDefaults returns default OAuth URLs and scopes for a connector type.
// Returns nil if the connector type doesn't support OAuth.
func (r *ConnectorRegistry) GetOAuthDefaults(connectorType string) *driving.OAuthDefaults {
	if r.connectorFactory == nil {
		return nil
	}
	defaults := r.connectorFactory.GetDefaultOAuthConfig(connectorType)
	if defaults == nil {
		return nil
	}
	return &driving.OAuthDefaults{
		AuthURL:  defaults.AuthURL,
		TokenURL: defaults.TokenURL,
		Scopes:   defaults.Scopes,
	}
}

// SupportsOAuth returns true if the connector type supports OAuth authentication.
func (r *ConnectorRegistry) SupportsOAuth(connectorType string) bool {
	if r.connectorFactory == nil {
		return false
	}
	return r.connectorFactory.SupportsOAuth(connectorType)
}

// BuildAuthURL constructs the OAuth authorization URL for a connector type.
// Includes provider-specific parameters (e.g., access_type=offline for Google).
func (r *ConnectorRegistry) BuildAuthURL(
	connectorType string,
	authProvider *domain.AuthProvider,
	redirectURI, state, codeChallenge string,
) (string, error) {
	if r.connectorFactory == nil {
		return "", domain.ErrNotFound
	}
	return r.connectorFactory.BuildAuthURL(connectorType, authProvider, redirectURI, state, codeChallenge)
}

// GetUserInfo fetches the account identifier (email/username) for a connector type.
func (r *ConnectorRegistry) GetUserInfo(
	ctx context.Context,
	connectorType string,
	accessToken string,
) (string, error) {
	if r.connectorFactory == nil {
		return "", domain.ErrNotFound
	}
	return r.connectorFactory.GetUserInfo(ctx, connectorType, accessToken)
}

// GetSetupHint returns guidance text for setting up OAuth/PAT with a provider.
func (r *ConnectorRegistry) GetSetupHint(connectorType string) string {
	if r.connectorFactory == nil {
		return ""
	}
	return r.connectorFactory.GetSetupHint(connectorType)
}

// ExchangeCode exchanges an authorization code for tokens using connector-specific logic.
func (r *ConnectorRegistry) ExchangeCode(
	ctx context.Context,
	connectorType string,
	authProvider *domain.AuthProvider,
	code, redirectURI, codeVerifier string,
) (*domain.OAuthToken, error) {
	if r.connectorFactory == nil {
		return nil, domain.ErrNotFound
	}
	return r.connectorFactory.ExchangeCode(ctx, connectorType, authProvider, code, redirectURI, codeVerifier)
}
