package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchMode_IsValid tests all valid and invalid search modes
func TestSearchMode_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		mode     SearchMode
		expected bool
	}{
		{
			name:     "text_only is valid",
			mode:     SearchModeTextOnly,
			expected: true,
		},
		{
			name:     "hybrid is valid",
			mode:     SearchModeHybrid,
			expected: true,
		},
		{
			name:     "llm_assisted is valid",
			mode:     SearchModeLLMAssisted,
			expected: true,
		},
		{
			name:     "full is valid",
			mode:     SearchModeFull,
			expected: true,
		},
		{
			name:     "empty string is invalid",
			mode:     SearchMode(""),
			expected: false,
		},
		{
			name:     "unknown mode is invalid",
			mode:     SearchMode("unknown"),
			expected: false,
		},
		{
			name:     "invalid mode is invalid",
			mode:     SearchMode("invalid_mode"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSearchMode_RequiresEmbedding tests embedding requirements
func TestSearchMode_RequiresEmbedding(t *testing.T) {
	tests := []struct {
		name     string
		mode     SearchMode
		expected bool
	}{
		{
			name:     "text_only does not require embedding",
			mode:     SearchModeTextOnly,
			expected: false,
		},
		{
			name:     "hybrid requires embedding",
			mode:     SearchModeHybrid,
			expected: true,
		},
		{
			name:     "llm_assisted does not require embedding",
			mode:     SearchModeLLMAssisted,
			expected: false,
		},
		{
			name:     "full requires embedding",
			mode:     SearchModeFull,
			expected: true,
		},
		{
			name:     "unknown mode does not require embedding",
			mode:     SearchMode("unknown"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.RequiresEmbedding()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSearchMode_RequiresLLM tests LLM requirements
func TestSearchMode_RequiresLLM(t *testing.T) {
	tests := []struct {
		name     string
		mode     SearchMode
		expected bool
	}{
		{
			name:     "text_only does not require LLM",
			mode:     SearchModeTextOnly,
			expected: false,
		},
		{
			name:     "hybrid does not require LLM",
			mode:     SearchModeHybrid,
			expected: false,
		},
		{
			name:     "llm_assisted requires LLM",
			mode:     SearchModeLLMAssisted,
			expected: true,
		},
		{
			name:     "full requires LLM",
			mode:     SearchModeFull,
			expected: true,
		},
		{
			name:     "unknown mode does not require LLM",
			mode:     SearchMode("unknown"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.RequiresLLM()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSearchMode_String tests string representation
func TestSearchMode_String(t *testing.T) {
	tests := []struct {
		name     string
		mode     SearchMode
		expected string
	}{
		{
			name:     "text_only string",
			mode:     SearchModeTextOnly,
			expected: "text_only",
		},
		{
			name:     "hybrid string",
			mode:     SearchModeHybrid,
			expected: "hybrid",
		},
		{
			name:     "llm_assisted string",
			mode:     SearchModeLLMAssisted,
			expected: "llm_assisted",
		},
		{
			name:     "full string",
			mode:     SearchModeFull,
			expected: "full",
		},
		{
			name:     "unknown returns as-is",
			mode:     SearchMode("unknown"),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSearchMode_Description tests human-readable descriptions
func TestSearchMode_Description(t *testing.T) {
	tests := []struct {
		name     string
		mode     SearchMode
		expected string
	}{
		{
			name:     "text_only description",
			mode:     SearchModeTextOnly,
			expected: "Text Only (keyword search)",
		},
		{
			name:     "hybrid description",
			mode:     SearchModeHybrid,
			expected: "Hybrid (text + semantic search)",
		},
		{
			name:     "llm_assisted description",
			mode:     SearchModeLLMAssisted,
			expected: "LLM Assisted (text + query expansion)",
		},
		{
			name:     "full description",
			mode:     SearchModeFull,
			expected: "Full (text + semantic + LLM)",
		},
		{
			name:     "unknown returns Unknown",
			mode:     SearchMode("unknown"),
			expected: unknownDescription,
		},
		{
			name:     "empty string returns Unknown",
			mode:     SearchMode(""),
			expected: unknownDescription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.Description()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAIProvider_IsValid tests all valid and invalid AI providers
func TestAIProvider_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		provider AIProvider
		expected bool
	}{
		{
			name:     "ollama is valid",
			provider: AIProviderOllama,
			expected: true,
		},
		{
			name:     "openai is valid",
			provider: AIProviderOpenAI,
			expected: true,
		},
		{
			name:     "anthropic is valid",
			provider: AIProviderAnthropic,
			expected: true,
		},
		{
			name:     "empty string is invalid",
			provider: AIProvider(""),
			expected: false,
		},
		{
			name:     "unknown provider is invalid",
			provider: AIProvider("unknown"),
			expected: false,
		},
		{
			name:     "invalid provider is invalid",
			provider: AIProvider("invalid_provider"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAIProvider_RequiresAPIKey tests API key requirements
func TestAIProvider_RequiresAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider AIProvider
		expected bool
	}{
		{
			name:     "ollama does not require API key",
			provider: AIProviderOllama,
			expected: false,
		},
		{
			name:     "openai requires API key",
			provider: AIProviderOpenAI,
			expected: true,
		},
		{
			name:     "anthropic requires API key",
			provider: AIProviderAnthropic,
			expected: true,
		},
		{
			name:     "unknown does not require API key",
			provider: AIProvider("unknown"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.RequiresAPIKey()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAIProvider_IsLocal tests local provider identification
func TestAIProvider_IsLocal(t *testing.T) {
	tests := []struct {
		name     string
		provider AIProvider
		expected bool
	}{
		{
			name:     "ollama is local",
			provider: AIProviderOllama,
			expected: true,
		},
		{
			name:     "openai is not local",
			provider: AIProviderOpenAI,
			expected: false,
		},
		{
			name:     "anthropic is not local",
			provider: AIProviderAnthropic,
			expected: false,
		},
		{
			name:     "unknown is not local",
			provider: AIProvider("unknown"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.IsLocal()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAIProvider_String tests string representation
func TestAIProvider_String(t *testing.T) {
	tests := []struct {
		name     string
		provider AIProvider
		expected string
	}{
		{
			name:     "ollama string",
			provider: AIProviderOllama,
			expected: "ollama",
		},
		{
			name:     "openai string",
			provider: AIProviderOpenAI,
			expected: "openai",
		},
		{
			name:     "anthropic string",
			provider: AIProviderAnthropic,
			expected: "anthropic",
		},
		{
			name:     "unknown returns as-is",
			provider: AIProvider("unknown"),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAIProvider_Description tests human-readable descriptions
func TestAIProvider_Description(t *testing.T) {
	tests := []struct {
		name     string
		provider AIProvider
		expected string
	}{
		{
			name:     "ollama description",
			provider: AIProviderOllama,
			expected: "Ollama (local)",
		},
		{
			name:     "openai description",
			provider: AIProviderOpenAI,
			expected: "OpenAI (cloud)",
		},
		{
			name:     "anthropic description",
			provider: AIProviderAnthropic,
			expected: "Anthropic (cloud)",
		},
		{
			name:     "unknown returns Unknown",
			provider: AIProvider("unknown"),
			expected: unknownDescription,
		},
		{
			name:     "empty string returns Unknown",
			provider: AIProvider(""),
			expected: unknownDescription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Description()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEmbeddingSettings_IsConfigured tests embedding configuration validation
func TestEmbeddingSettings_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		settings EmbeddingSettings
		expected bool
	}{
		{
			name: "valid ollama configuration",
			settings: EmbeddingSettings{
				Provider: AIProviderOllama,
				Model:    "nomic-embed-text",
				BaseURL:  "http://localhost:11434",
			},
			expected: true,
		},
		{
			name: "valid openai configuration with API key",
			settings: EmbeddingSettings{
				Provider: AIProviderOpenAI,
				Model:    "text-embedding-3-small",
				APIKey:   "sk-test123",
			},
			expected: true,
		},
		{
			name: "invalid provider",
			settings: EmbeddingSettings{
				Provider: AIProvider("invalid"),
				Model:    "some-model",
			},
			expected: false,
		},
		{
			name: "openai without API key",
			settings: EmbeddingSettings{
				Provider: AIProviderOpenAI,
				Model:    "text-embedding-3-small",
				APIKey:   "",
			},
			expected: false,
		},
		{
			name: "empty provider",
			settings: EmbeddingSettings{
				Provider: AIProvider(""),
				Model:    "some-model",
			},
			expected: false,
		},
		{
			name: "ollama with empty API key is valid",
			settings: EmbeddingSettings{
				Provider: AIProviderOllama,
				Model:    "nomic-embed-text",
				APIKey:   "",
			},
			expected: true,
		},
		{
			name:     "empty settings",
			settings: EmbeddingSettings{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.settings.IsConfigured()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLLMSettings_IsConfigured tests LLM configuration validation
func TestLLMSettings_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		settings LLMSettings
		expected bool
	}{
		{
			name: "valid ollama configuration",
			settings: LLMSettings{
				Provider: AIProviderOllama,
				Model:    "llama3.2",
				BaseURL:  "http://localhost:11434",
			},
			expected: true,
		},
		{
			name: "valid openai configuration with API key",
			settings: LLMSettings{
				Provider: AIProviderOpenAI,
				Model:    "gpt-4o-mini",
				APIKey:   "sk-test123",
			},
			expected: true,
		},
		{
			name: "valid anthropic configuration with API key",
			settings: LLMSettings{
				Provider: AIProviderAnthropic,
				Model:    "claude-3-5-sonnet-latest",
				APIKey:   "sk-ant-test123",
			},
			expected: true,
		},
		{
			name: "invalid provider",
			settings: LLMSettings{
				Provider: AIProvider("invalid"),
				Model:    "some-model",
			},
			expected: false,
		},
		{
			name: "openai without API key",
			settings: LLMSettings{
				Provider: AIProviderOpenAI,
				Model:    "gpt-4o-mini",
				APIKey:   "",
			},
			expected: false,
		},
		{
			name: "anthropic without API key",
			settings: LLMSettings{
				Provider: AIProviderAnthropic,
				Model:    "claude-3-5-sonnet-latest",
				APIKey:   "",
			},
			expected: false,
		},
		{
			name: "empty provider",
			settings: LLMSettings{
				Provider: AIProvider(""),
				Model:    "some-model",
			},
			expected: false,
		},
		{
			name: "ollama with empty API key is valid",
			settings: LLMSettings{
				Provider: AIProviderOllama,
				Model:    "llama3.2",
				APIKey:   "",
			},
			expected: true,
		},
		{
			name:     "empty settings",
			settings: LLMSettings{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.settings.IsConfigured()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultAppSettings tests default settings creation
func TestDefaultAppSettings(t *testing.T) {
	settings := DefaultAppSettings()

	// Test search settings
	assert.Equal(t, SearchModeTextOnly, settings.Search.Mode)

	// Test embedding settings - should be unconfigured by default
	assert.Empty(t, settings.Embedding.Provider)
	assert.Empty(t, settings.Embedding.Model)
	assert.Empty(t, settings.Embedding.BaseURL)
	assert.Empty(t, settings.Embedding.APIKey)
	assert.False(t, settings.Embedding.IsConfigured())

	// Test LLM settings - should be unconfigured by default
	assert.Empty(t, settings.LLM.Provider)
	assert.Empty(t, settings.LLM.Model)
	assert.Empty(t, settings.LLM.BaseURL)
	assert.Empty(t, settings.LLM.APIKey)
	assert.False(t, settings.LLM.IsConfigured())

	// Test vector index settings
	assert.False(t, settings.VectorIndex.Enabled)
	assert.Equal(t, 768, settings.VectorIndex.Dimensions)
}

// TestAllSearchModes tests complete list of search modes
func TestAllSearchModes(t *testing.T) {
	modes := AllSearchModes()

	require.Len(t, modes, 4)
	assert.Contains(t, modes, SearchModeTextOnly)
	assert.Contains(t, modes, SearchModeHybrid)
	assert.Contains(t, modes, SearchModeLLMAssisted)
	assert.Contains(t, modes, SearchModeFull)

	// Verify all modes are valid
	for _, mode := range modes {
		assert.True(t, mode.IsValid(), "Mode %s should be valid", mode)
	}
}

// TestAllEmbeddingProviders tests complete list of embedding providers
func TestAllEmbeddingProviders(t *testing.T) {
	providers := AllEmbeddingProviders()

	require.Len(t, providers, 2)
	assert.Contains(t, providers, AIProviderOllama)
	assert.Contains(t, providers, AIProviderOpenAI)
	assert.NotContains(t, providers, AIProviderAnthropic, "Anthropic should not be in embedding providers")

	// Verify all providers are valid
	for _, provider := range providers {
		assert.True(t, provider.IsValid(), "Provider %s should be valid", provider)
	}
}

// TestAllLLMProviders tests complete list of LLM providers
func TestAllLLMProviders(t *testing.T) {
	providers := AllLLMProviders()

	require.Len(t, providers, 3)
	assert.Contains(t, providers, AIProviderOllama)
	assert.Contains(t, providers, AIProviderOpenAI)
	assert.Contains(t, providers, AIProviderAnthropic)

	// Verify all providers are valid
	for _, provider := range providers {
		assert.True(t, provider.IsValid(), "Provider %s should be valid", provider)
	}
}

// TestDefaultEmbeddingModels tests default embedding model mappings
func TestDefaultEmbeddingModels(t *testing.T) {
	models := DefaultEmbeddingModels()

	require.Len(t, models, 2)
	assert.Equal(t, "nomic-embed-text", models[AIProviderOllama])
	assert.Equal(t, "text-embedding-3-small", models[AIProviderOpenAI])
	assert.NotContains(t, models, AIProviderAnthropic)
}

// TestDefaultLLMModels tests default LLM model mappings
func TestDefaultLLMModels(t *testing.T) {
	models := DefaultLLMModels()

	require.Len(t, models, 3)
	assert.Equal(t, "llama3.2", models[AIProviderOllama])
	assert.Equal(t, "gpt-4o-mini", models[AIProviderOpenAI])
	assert.Equal(t, "claude-3-5-sonnet-latest", models[AIProviderAnthropic])
}

// TestEmbeddingDimensions tests embedding dimensions mapping
func TestEmbeddingDimensions(t *testing.T) {
	dimensions := EmbeddingDimensions()

	require.NotEmpty(t, dimensions)

	// Test Ollama models
	assert.Equal(t, 768, dimensions["nomic-embed-text"])
	assert.Equal(t, 1024, dimensions["mxbai-embed-large"])
	assert.Equal(t, 384, dimensions["all-minilm"])

	// Test OpenAI models
	assert.Equal(t, 1536, dimensions["text-embedding-3-small"])
	assert.Equal(t, 3072, dimensions["text-embedding-3-large"])
	assert.Equal(t, 1536, dimensions["text-embedding-ada-002"])

	// Test unknown model
	_, exists := dimensions["unknown-model"]
	assert.False(t, exists)
}

// TestSearchSettings_Fields tests SearchSettings structure
func TestSearchSettings_Fields(t *testing.T) {
	settings := SearchSettings{
		Mode: SearchModeHybrid,
	}

	assert.Equal(t, SearchModeHybrid, settings.Mode)
}

// TestEmbeddingSettings_Fields tests EmbeddingSettings structure
func TestEmbeddingSettings_Fields(t *testing.T) {
	settings := EmbeddingSettings{
		Provider: AIProviderOpenAI,
		Model:    "text-embedding-3-small",
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test123",
	}

	assert.Equal(t, AIProviderOpenAI, settings.Provider)
	assert.Equal(t, "text-embedding-3-small", settings.Model)
	assert.Equal(t, "https://api.openai.com", settings.BaseURL)
	assert.Equal(t, "sk-test123", settings.APIKey)
}

// TestLLMSettings_Fields tests LLMSettings structure
func TestLLMSettings_Fields(t *testing.T) {
	settings := LLMSettings{
		Provider: AIProviderAnthropic,
		Model:    "claude-3-5-sonnet-latest",
		BaseURL:  "https://api.anthropic.com",
		APIKey:   "sk-ant-test123",
	}

	assert.Equal(t, AIProviderAnthropic, settings.Provider)
	assert.Equal(t, "claude-3-5-sonnet-latest", settings.Model)
	assert.Equal(t, "https://api.anthropic.com", settings.BaseURL)
	assert.Equal(t, "sk-ant-test123", settings.APIKey)
}

// TestVectorIndexSettings_Fields tests VectorIndexSettings structure
func TestVectorIndexSettings_Fields(t *testing.T) {
	settings := VectorIndexSettings{
		Enabled:    true,
		Dimensions: 1536,
	}

	assert.True(t, settings.Enabled)
	assert.Equal(t, 1536, settings.Dimensions)
}

// TestAppSettings_CompleteStructure tests full AppSettings structure
func TestAppSettings_CompleteStructure(t *testing.T) {
	settings := AppSettings{
		Search: SearchSettings{
			Mode: SearchModeFull,
		},
		Embedding: EmbeddingSettings{
			Provider: AIProviderOpenAI,
			Model:    "text-embedding-3-small",
			BaseURL:  "https://api.openai.com",
			APIKey:   "sk-embed-test",
		},
		LLM: LLMSettings{
			Provider: AIProviderAnthropic,
			Model:    "claude-3-5-sonnet-latest",
			BaseURL:  "https://api.anthropic.com",
			APIKey:   "sk-ant-llm-test",
		},
		VectorIndex: VectorIndexSettings{
			Enabled:    true,
			Dimensions: 1536,
		},
	}

	// Verify search settings
	assert.Equal(t, SearchModeFull, settings.Search.Mode)

	// Verify embedding settings
	assert.Equal(t, AIProviderOpenAI, settings.Embedding.Provider)
	assert.Equal(t, "text-embedding-3-small", settings.Embedding.Model)
	assert.True(t, settings.Embedding.IsConfigured())

	// Verify LLM settings
	assert.Equal(t, AIProviderAnthropic, settings.LLM.Provider)
	assert.Equal(t, "claude-3-5-sonnet-latest", settings.LLM.Model)
	assert.True(t, settings.LLM.IsConfigured())

	// Verify vector index settings
	assert.True(t, settings.VectorIndex.Enabled)
	assert.Equal(t, 1536, settings.VectorIndex.Dimensions)
}

// TestUnknownDescription_Constant tests the private constant
func TestUnknownDescription_Constant(t *testing.T) {
	// Test that all Description methods return this constant for invalid values
	assert.Equal(t, unknownDescription, SearchMode("invalid").Description())
	assert.Equal(t, unknownDescription, AIProvider("invalid").Description())
}
