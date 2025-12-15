package connectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/connectors/dropbox"
	"github.com/custodia-labs/sercha-cli/internal/connectors/filesystem"
	"github.com/custodia-labs/sercha-cli/internal/connectors/github"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google/calendar"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google/drive"
	"github.com/custodia-labs/sercha-cli/internal/connectors/google/gmail"
	"github.com/custodia-labs/sercha-cli/internal/connectors/microsoft"
	mscalendar "github.com/custodia-labs/sercha-cli/internal/connectors/microsoft/calendar"
	"github.com/custodia-labs/sercha-cli/internal/connectors/microsoft/onedrive"
	"github.com/custodia-labs/sercha-cli/internal/connectors/microsoft/outlook"
	"github.com/custodia-labs/sercha-cli/internal/connectors/notion"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Factory implements the interface.
var _ driven.ConnectorFactory = (*Factory)(nil)

// TokenProviderFactory creates TokenProviders for sources.
// This interface is satisfied by auth.Factory.
type TokenProviderFactory interface {
	CreateTokenProvider(ctx context.Context, source *domain.Source) (driven.TokenProvider, error)
}

// Factory creates connectors based on source configuration.
type Factory struct {
	mu                   sync.RWMutex
	builders             map[string]driven.ConnectorBuilder
	oauthHandlers        map[string]OAuthHandler
	tokenProviderFactory TokenProviderFactory
}

// NewFactory creates a new connector factory with default builders registered.
// The tokenProviderFactory is used to resolve authorization IDs to TokenProviders.
func NewFactory(tokenProviderFactory TokenProviderFactory) *Factory {
	f := &Factory{
		builders:             make(map[string]driven.ConnectorBuilder),
		oauthHandlers:        make(map[string]OAuthHandler),
		tokenProviderFactory: tokenProviderFactory,
	}
	f.registerDefaultBuilders()
	f.registerOAuthHandlers()
	return f
}

// registerDefaultBuilders registers all built-in connector builders.
func (f *Factory) registerDefaultBuilders() {
	f.Register("filesystem", func(source domain.Source, _ driven.TokenProvider) (driven.Connector, error) {
		path, ok := source.Config["path"]
		if !ok {
			return nil, fmt.Errorf("filesystem source requires 'path' config")
		}
		return filesystem.New(source.ID, path), nil
	})

	f.Register("github", func(source domain.Source, tokenProvider driven.TokenProvider) (driven.Connector, error) {
		cfg, err := github.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("github config: %w", err)
		}
		return github.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("google-drive", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := drive.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("google-drive config: %w", err)
		}
		return drive.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("gmail", func(source domain.Source, tokenProvider driven.TokenProvider) (driven.Connector, error) {
		cfg, err := gmail.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("gmail config: %w", err)
		}
		return gmail.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("google-calendar", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := calendar.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("google-calendar config: %w", err)
		}
		return calendar.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("outlook", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := outlook.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("outlook config: %w", err)
		}
		return outlook.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("onedrive", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := onedrive.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("onedrive config: %w", err)
		}
		return onedrive.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("microsoft-calendar", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := mscalendar.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("microsoft-calendar config: %w", err)
		}
		return mscalendar.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("dropbox", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := dropbox.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("dropbox config: %w", err)
		}
		return dropbox.New(source.ID, cfg, tokenProvider), nil
	})

	f.Register("notion", func(
		source domain.Source, tokenProvider driven.TokenProvider,
	) (driven.Connector, error) {
		cfg, err := notion.ParseConfig(source)
		if err != nil {
			return nil, fmt.Errorf("notion config: %w", err)
		}
		return notion.New(source.ID, cfg, tokenProvider), nil
	})
}

// registerOAuthHandlers registers OAuth handlers for all connector types that support OAuth.
func (f *Factory) registerOAuthHandlers() {
	// Google OAuth handler for all Google connectors
	googleOAuth := google.NewOAuthHandler()
	f.RegisterOAuthHandler("google-drive", googleOAuth)
	f.RegisterOAuthHandler("gmail", googleOAuth)
	f.RegisterOAuthHandler("google-calendar", googleOAuth)

	// GitHub OAuth handler
	f.RegisterOAuthHandler("github", github.NewOAuthHandler())

	// Microsoft OAuth handler for all Microsoft connectors
	microsoftOAuth := microsoft.NewOAuthHandler()
	f.RegisterOAuthHandler("outlook", microsoftOAuth)
	f.RegisterOAuthHandler("onedrive", microsoftOAuth)
	f.RegisterOAuthHandler("microsoft-calendar", microsoftOAuth)

	// Dropbox OAuth handler
	f.RegisterOAuthHandler("dropbox", dropbox.NewOAuthHandler())

	// Notion OAuth handler
	f.RegisterOAuthHandler("notion", notion.NewOAuthHandler())
}

// Create instantiates a connector for the given source.
// Resolves TokenProvider from source credentials internally.
func (f *Factory) Create(ctx context.Context, source domain.Source) (driven.Connector, error) {
	f.mu.RLock()
	builder, ok := f.builders[source.Type]
	f.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedType, source.Type)
	}

	// Resolve TokenProvider for this source
	tokenProvider, err := f.tokenProviderFactory.CreateTokenProvider(ctx, &source)
	if err != nil {
		return nil, fmt.Errorf("create token provider for source %s: %w", source.ID, err)
	}

	return builder(source, tokenProvider)
}

// Register adds a connector builder for the given type.
func (f *Factory) Register(connectorType string, builder driven.ConnectorBuilder) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.builders[connectorType] = builder
}

// SupportedTypes returns all registered connector types.
func (f *Factory) SupportedTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	types := make([]string, 0, len(f.builders))
	for t := range f.builders {
		types = append(types, t)
	}
	return types
}

// RegisterOAuthHandler adds an OAuth handler for a connector type.
func (f *Factory) RegisterOAuthHandler(connectorType string, handler OAuthHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.oauthHandlers[connectorType] = handler
}

// BuildAuthURL constructs the OAuth authorization URL for a connector type.
// Includes provider-specific parameters (e.g., access_type=offline for Google).
func (f *Factory) BuildAuthURL(
	connectorType string,
	authProvider *domain.AuthProvider,
	redirectURI, state, codeChallenge string,
) (string, error) {
	f.mu.RLock()
	handler, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("no OAuth handler for connector type: %s", connectorType)
	}
	return handler.BuildAuthURL(authProvider, redirectURI, state, codeChallenge), nil
}

// ExchangeCode exchanges an authorization code for tokens.
func (f *Factory) ExchangeCode(
	ctx context.Context,
	connectorType string,
	authProvider *domain.AuthProvider,
	code, redirectURI, codeVerifier string,
) (*domain.OAuthToken, error) {
	f.mu.RLock()
	handler, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no OAuth handler for connector type: %s", connectorType)
	}
	return handler.ExchangeCode(ctx, authProvider, code, redirectURI, codeVerifier)
}

// RefreshToken refreshes an expired access token using a refresh token.
func (f *Factory) RefreshToken(
	ctx context.Context,
	connectorType string,
	authProvider *domain.AuthProvider,
	refreshToken string,
) (*domain.OAuthToken, error) {
	f.mu.RLock()
	handler, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no OAuth handler for connector type: %s", connectorType)
	}
	return handler.RefreshToken(ctx, authProvider, refreshToken)
}

// GetUserInfo fetches the account identifier (email/username) for a connector type.
func (f *Factory) GetUserInfo(
	ctx context.Context,
	connectorType string,
	accessToken string,
) (string, error) {
	f.mu.RLock()
	handler, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("no OAuth handler for connector type: %s", connectorType)
	}
	return handler.GetUserInfo(ctx, accessToken)
}

// GetDefaultOAuthConfig returns default OAuth URLs and scopes for a connector type.
// Returns nil if the connector type doesn't support OAuth.
func (f *Factory) GetDefaultOAuthConfig(connectorType string) *driven.OAuthDefaults {
	f.mu.RLock()
	handler, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	if !ok {
		return nil
	}
	defaults := handler.DefaultConfig()
	return &defaults
}

// SupportsOAuth returns true if the connector type supports OAuth authentication.
func (f *Factory) SupportsOAuth(connectorType string) bool {
	f.mu.RLock()
	_, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	return ok
}

// GetSetupHint returns guidance text for setting up OAuth/PAT with a provider.
// Returns empty string if no hint is available.
func (f *Factory) GetSetupHint(connectorType string) string {
	f.mu.RLock()
	handler, ok := f.oauthHandlers[connectorType]
	f.mu.RUnlock()
	if !ok {
		return ""
	}
	return handler.SetupHint()
}
