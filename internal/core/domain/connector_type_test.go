package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthMethod_Constants tests all auth method constants
func TestAuthMethod_Constants(t *testing.T) {
	tests := []struct {
		name     string
		method   AuthMethod
		expected string
	}{
		{
			name:     "none auth method",
			method:   AuthMethodNone,
			expected: "none",
		},
		{
			name:     "pat auth method",
			method:   AuthMethodPAT,
			expected: "pat",
		},
		{
			name:     "oauth auth method",
			method:   AuthMethodOAuth,
			expected: "oauth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.method))
		})
	}
}

// TestConnectorType_Fields tests ConnectorType structure
func TestConnectorType_Fields(t *testing.T) {
	connector := ConnectorType{
		ID:           "filesystem",
		Name:         "Filesystem",
		Description:  "Index local files and directories",
		ProviderType: ProviderLocal,
		AuthMethod:   AuthMethodNone,
		ConfigKeys: []ConfigKey{
			{
				Key:         "root_path",
				Label:       "Root Path",
				Description: "The root directory to index",
				Required:    true,
				Secret:      false,
			},
		},
	}

	assert.Equal(t, "filesystem", connector.ID)
	assert.Equal(t, "Filesystem", connector.Name)
	assert.Equal(t, "Index local files and directories", connector.Description)
	assert.Equal(t, ProviderLocal, connector.ProviderType)
	assert.Equal(t, AuthMethodNone, connector.AuthMethod)
	require.Len(t, connector.ConfigKeys, 1)
	assert.Equal(t, "root_path", connector.ConfigKeys[0].Key)
}

// TestConnectorType_GitHubExample tests GitHub connector type
func TestConnectorType_GitHubExample(t *testing.T) {
	connector := ConnectorType{
		ID:           "github",
		Name:         "GitHub",
		Description:  "Index GitHub repositories and issues",
		ProviderType: ProviderGitHub,
		AuthMethod:   AuthMethodPAT,
		ConfigKeys: []ConfigKey{
			{
				Key:         "repository",
				Label:       "Repository",
				Description: "Repository in format owner/repo",
				Required:    true,
				Secret:      false,
			},
			{
				Key:         "include_issues",
				Label:       "Include Issues",
				Description: "Whether to index issues",
				Required:    false,
				Secret:      false,
			},
		},
	}

	assert.Equal(t, "github", connector.ID)
	assert.Equal(t, "GitHub", connector.Name)
	assert.Equal(t, ProviderGitHub, connector.ProviderType)
	assert.Equal(t, AuthMethodPAT, connector.AuthMethod)
	require.Len(t, connector.ConfigKeys, 2)
}

// TestConnectorType_GoogleDriveExample tests Google Drive connector type
func TestConnectorType_GoogleDriveExample(t *testing.T) {
	connector := ConnectorType{
		ID:           "google-drive",
		Name:         "Google Drive",
		Description:  "Index Google Drive documents",
		ProviderType: ProviderGoogle,
		AuthMethod:   AuthMethodOAuth,
		ConfigKeys: []ConfigKey{
			{
				Key:         "folder_id",
				Label:       "Folder ID",
				Description: "Google Drive folder ID to index",
				Required:    false,
				Secret:      false,
			},
			{
				Key:         "shared_drives",
				Label:       "Include Shared Drives",
				Description: "Whether to include shared drives",
				Required:    false,
				Secret:      false,
			},
		},
	}

	assert.Equal(t, "google-drive", connector.ID)
	assert.Equal(t, "Google Drive", connector.Name)
	assert.Equal(t, ProviderGoogle, connector.ProviderType)
	assert.Equal(t, AuthMethodOAuth, connector.AuthMethod)
	assert.NotEmpty(t, connector.ConfigKeys)
}

// TestConnectorType_EmptyConfigKeys tests connector with no config keys
func TestConnectorType_EmptyConfigKeys(t *testing.T) {
	connector := ConnectorType{
		ID:           "simple",
		Name:         "Simple Connector",
		Description:  "A connector with no configuration",
		ProviderType: ProviderLocal,
		AuthMethod:   AuthMethodNone,
		ConfigKeys:   []ConfigKey{},
	}

	assert.Equal(t, "simple", connector.ID)
	assert.Empty(t, connector.ConfigKeys)
}

// TestConnectorType_NilConfigKeys tests connector with nil config keys
func TestConnectorType_NilConfigKeys(t *testing.T) {
	connector := ConnectorType{
		ID:           "simple",
		Name:         "Simple Connector",
		Description:  "A connector with nil configuration",
		ProviderType: ProviderLocal,
		AuthMethod:   AuthMethodNone,
		ConfigKeys:   nil,
	}

	assert.Equal(t, "simple", connector.ID)
	assert.Nil(t, connector.ConfigKeys)
}

// TestConfigKey_Fields tests ConfigKey structure
func TestConfigKey_Fields(t *testing.T) {
	config := ConfigKey{
		Key:         "api_token",
		Label:       "API Token",
		Description: "Your API authentication token",
		Required:    true,
		Secret:      true,
	}

	assert.Equal(t, "api_token", config.Key)
	assert.Equal(t, "API Token", config.Label)
	assert.Equal(t, "Your API authentication token", config.Description)
	assert.True(t, config.Required)
	assert.True(t, config.Secret)
}

// TestConfigKey_RequiredField tests required configuration field
func TestConfigKey_RequiredField(t *testing.T) {
	config := ConfigKey{
		Key:         "endpoint",
		Label:       "Endpoint URL",
		Description: "The API endpoint URL",
		Required:    true,
		Secret:      false,
	}

	assert.True(t, config.Required)
	assert.False(t, config.Secret)
}

// TestConfigKey_OptionalField tests optional configuration field
func TestConfigKey_OptionalField(t *testing.T) {
	config := ConfigKey{
		Key:         "timeout",
		Label:       "Timeout",
		Description: "Request timeout in seconds",
		Required:    false,
		Secret:      false,
	}

	assert.False(t, config.Required)
	assert.False(t, config.Secret)
}

// TestConfigKey_SecretField tests secret configuration field
func TestConfigKey_SecretField(t *testing.T) {
	config := ConfigKey{
		Key:         "password",
		Label:       "Password",
		Description: "Your account password",
		Required:    true,
		Secret:      true,
	}

	assert.True(t, config.Required)
	assert.True(t, config.Secret)
}

// TestConfigKey_EmptyStrings tests config key with empty strings
func TestConfigKey_EmptyStrings(t *testing.T) {
	config := ConfigKey{
		Key:         "",
		Label:       "",
		Description: "",
		Required:    false,
		Secret:      false,
	}

	assert.Empty(t, config.Key)
	assert.Empty(t, config.Label)
	assert.Empty(t, config.Description)
}

// TestConnectorType_MultipleConfigKeys tests connector with multiple config keys
func TestConnectorType_MultipleConfigKeys(t *testing.T) {
	connector := ConnectorType{
		ID:           "slack",
		Name:         "Slack",
		Description:  "Index Slack messages and files",
		ProviderType: ProviderSlack,
		AuthMethod:   AuthMethodOAuth,
		ConfigKeys: []ConfigKey{
			{
				Key:         "workspace_id",
				Label:       "Workspace ID",
				Description: "Your Slack workspace ID",
				Required:    true,
				Secret:      false,
			},
			{
				Key:         "channel_filter",
				Label:       "Channel Filter",
				Description: "Comma-separated list of channels to index",
				Required:    false,
				Secret:      false,
			},
			{
				Key:         "include_private",
				Label:       "Include Private Channels",
				Description: "Whether to include private channels",
				Required:    false,
				Secret:      false,
			},
		},
	}

	require.Len(t, connector.ConfigKeys, 3)
	assert.Equal(t, "workspace_id", connector.ConfigKeys[0].Key)
	assert.Equal(t, "channel_filter", connector.ConfigKeys[1].Key)
	assert.Equal(t, "include_private", connector.ConfigKeys[2].Key)

	// Verify required flags
	assert.True(t, connector.ConfigKeys[0].Required)
	assert.False(t, connector.ConfigKeys[1].Required)
	assert.False(t, connector.ConfigKeys[2].Required)

	// Verify secret flags
	assert.False(t, connector.ConfigKeys[0].Secret)
	assert.False(t, connector.ConfigKeys[1].Secret)
	assert.False(t, connector.ConfigKeys[2].Secret)
}

// TestAuthMethod_TypeSafety tests that AuthMethod is a distinct type
func TestAuthMethod_TypeSafety(t *testing.T) {
	var method AuthMethod = AuthMethodPAT

	// Should be able to compare with constants
	assert.Equal(t, AuthMethodPAT, method)
	assert.NotEqual(t, AuthMethodNone, method)
	assert.NotEqual(t, AuthMethodOAuth, method)

	// Should be able to convert to string
	assert.Equal(t, "pat", string(method))
}

// TestProviderType_AllProviders tests all provider type constants
func TestProviderType_AllProviders(t *testing.T) {
	providers := []ProviderType{
		ProviderLocal,
		ProviderGoogle,
		ProviderGitHub,
		ProviderSlack,
		ProviderNotion,
	}

	expected := []string{"local", "google", "github", "slack", "notion"}

	require.Len(t, providers, len(expected))
	for i, provider := range providers {
		assert.Equal(t, expected[i], string(provider))
	}
}

// TestConnectorType_NotionExample tests Notion connector type
func TestConnectorType_NotionExample(t *testing.T) {
	connector := ConnectorType{
		ID:           "notion",
		Name:         "Notion",
		Description:  "Index Notion pages and databases",
		ProviderType: ProviderNotion,
		AuthMethod:   AuthMethodOAuth,
		ConfigKeys: []ConfigKey{
			{
				Key:         "workspace_id",
				Label:       "Workspace ID",
				Description: "Notion workspace to index",
				Required:    false,
				Secret:      false,
			},
		},
	}

	assert.Equal(t, "notion", connector.ID)
	assert.Equal(t, ProviderNotion, connector.ProviderType)
	assert.Equal(t, AuthMethodOAuth, connector.AuthMethod)
}

// TestConfigKey_LongDescription tests config key with long description
func TestConfigKey_LongDescription(t *testing.T) {
	config := ConfigKey{
		Key:   "advanced_option",
		Label: "Advanced Option",
		Description: "This is a very long description that explains in great detail " +
			"what this configuration option does and how it should be used. " +
			"It may span multiple lines and contain lots of helpful information " +
			"for the user to understand the option.",
		Required: false,
		Secret:   false,
	}

	assert.NotEmpty(t, config.Description)
	assert.Contains(t, config.Description, "configuration option")
}

// TestConnectorType_MixedRequiredOptional tests connector with mix of required and optional fields
func TestConnectorType_MixedRequiredOptional(t *testing.T) {
	connector := ConnectorType{
		ID:           "custom",
		Name:         "Custom Connector",
		Description:  "A custom connector with mixed fields",
		ProviderType: ProviderLocal,
		AuthMethod:   AuthMethodNone,
		ConfigKeys: []ConfigKey{
			{Key: "required1", Required: true, Secret: false},
			{Key: "optional1", Required: false, Secret: false},
			{Key: "required2", Required: true, Secret: true},
			{Key: "optional2", Required: false, Secret: true},
		},
	}

	require.Len(t, connector.ConfigKeys, 4)

	// Count required vs optional
	requiredCount := 0
	secretCount := 0
	for _, key := range connector.ConfigKeys {
		if key.Required {
			requiredCount++
		}
		if key.Secret {
			secretCount++
		}
	}

	assert.Equal(t, 2, requiredCount)
	assert.Equal(t, 2, secretCount)
}
