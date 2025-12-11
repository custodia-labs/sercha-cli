package domain

// AuthMethod defines how a connector authenticates.
type AuthMethod string

const (
	// AuthMethodNone requires no authentication (e.g., filesystem).
	AuthMethodNone AuthMethod = "none"
	// AuthMethodPAT uses a Personal Access Token.
	AuthMethodPAT AuthMethod = "pat"
	// AuthMethodOAuth uses OAuth 2.0 with PKCE.
	AuthMethodOAuth AuthMethod = "oauth"
)

// ConnectorType describes a supported connector.
type ConnectorType struct {
	// ID is the unique identifier (e.g., "filesystem", "github", "google-drive").
	ID string
	// Name is the human-readable display name.
	Name string
	// Description provides a brief explanation of the connector.
	Description string
	// ProviderType identifies which auth provider this connector uses.
	ProviderType ProviderType
	// AuthCapability specifies what authentication methods this connector supports.
	// Use this to determine if user should be given a choice of auth methods.
	AuthCapability AuthCapability
	// AuthMethod specifies how the connector authenticates (derived from provider).
	// Deprecated: Use AuthCapability instead. Kept for backward compatibility.
	AuthMethod AuthMethod
	// ConfigKeys lists the configuration fields required by this connector.
	ConfigKeys []ConfigKey
	// WebURLResolver converts document URIs to web-openable URLs.
	// If nil, falls back to legacy URI conversion.
	WebURLResolver WebURLResolver
}

// WebURLResolver converts a document URI to a web-openable URL.
// Returns empty string if the URI cannot be resolved.
// Parameters:
//   - uri: The document URI (e.g., "gmail://messages/abc123")
//   - metadata: Document metadata (may contain pre-stored web links)
type WebURLResolver func(uri string, metadata map[string]any) string

// RequiresAuth returns true if this connector requires authentication.
func (c *ConnectorType) RequiresAuth() bool {
	return c.AuthCapability.RequiresAuth()
}

// ConfigKey describes a configuration field for a connector.
type ConfigKey struct {
	// Key is the configuration key name.
	Key string
	// Label is the human-readable label for UI display.
	Label string
	// Description explains what this field is for.
	Description string
	// Default is the default value for this field (shown in placeholder).
	Default string
	// Required indicates whether this field must be provided.
	Required bool
	// Secret indicates whether this field should be masked in UI (e.g., tokens).
	Secret bool
}
