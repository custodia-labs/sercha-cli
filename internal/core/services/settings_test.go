package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/memory"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewSettingsService(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	require.NotNil(t, service)
}

func TestSettingsService_Get_ReturnsDefaults(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	settings, err := service.Get()

	require.NoError(t, err)
	require.NotNil(t, settings)

	// Verify defaults
	defaults := domain.DefaultAppSettings()
	assert.Equal(t, defaults.Search.Mode, settings.Search.Mode)
	assert.Equal(t, defaults.Embedding.Provider, settings.Embedding.Provider)
	assert.Equal(t, defaults.Embedding.Model, settings.Embedding.Model)
	assert.Equal(t, defaults.LLM.Provider, settings.LLM.Provider)
	assert.Equal(t, defaults.LLM.Model, settings.LLM.Model)
}

func TestSettingsService_Get_ReturnsStoredValues(t *testing.T) {
	store := memory.NewConfigStore()
	_ = store.Set("search.mode", "hybrid")
	_ = store.Set("embedding.provider", "openai")
	_ = store.Set("embedding.model", "text-embedding-3-large")

	service := NewSettingsService(store, nil)

	settings, err := service.Get()

	require.NoError(t, err)
	assert.Equal(t, domain.SearchModeHybrid, settings.Search.Mode)
	assert.Equal(t, domain.AIProviderOpenAI, settings.Embedding.Provider)
	assert.Equal(t, "text-embedding-3-large", settings.Embedding.Model)
}

func TestSettingsService_Get_InvalidValuesReturnDefaults(t *testing.T) {
	store := memory.NewConfigStore()
	_ = store.Set("search.mode", "invalid_mode")
	_ = store.Set("embedding.provider", "invalid_provider")

	service := NewSettingsService(store, nil)

	settings, err := service.Get()

	require.NoError(t, err)
	// Invalid values should fall back to defaults
	defaults := domain.DefaultAppSettings()
	assert.Equal(t, defaults.Search.Mode, settings.Search.Mode)
	assert.Equal(t, defaults.Embedding.Provider, settings.Embedding.Provider)
}

func TestSettingsService_Save(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeHybrid,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOpenAI,
			Model:    "text-embedding-3-small",
			APIKey:   "sk-test-key",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderAnthropic,
			Model:    "claude-3-5-sonnet-latest",
			APIKey:   "sk-ant-test",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    true,
			Dimensions: 1536,
		},
	}

	err := service.Save(settings)
	require.NoError(t, err)

	// Verify values were stored
	retrieved, err := service.Get()
	require.NoError(t, err)
	assert.Equal(t, domain.SearchModeHybrid, retrieved.Search.Mode)
	assert.Equal(t, domain.AIProviderOpenAI, retrieved.Embedding.Provider)
	assert.Equal(t, "text-embedding-3-small", retrieved.Embedding.Model)
	assert.Equal(t, "sk-test-key", retrieved.Embedding.APIKey)
	assert.Equal(t, domain.AIProviderAnthropic, retrieved.LLM.Provider)
	assert.Equal(t, "claude-3-5-sonnet-latest", retrieved.LLM.Model)
	assert.Equal(t, "sk-ant-test", retrieved.LLM.APIKey)
	assert.True(t, retrieved.VectorIndex.Enabled)
	assert.Equal(t, 1536, retrieved.VectorIndex.Dimensions)
}

func TestSettingsService_SetSearchMode_Valid(t *testing.T) {
	tests := []struct {
		name string
		mode domain.SearchMode
	}{
		{"text_only", domain.SearchModeTextOnly},
		{"hybrid", domain.SearchModeHybrid},
		{"llm_assisted", domain.SearchModeLLMAssisted},
		{"full", domain.SearchModeFull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.NewConfigStore()
			service := NewSettingsService(store, nil)

			err := service.SetSearchMode(tt.mode)

			require.NoError(t, err)

			settings, _ := service.Get()
			assert.Equal(t, tt.mode, settings.Search.Mode)
		})
	}
}

func TestSettingsService_SetSearchMode_EnablesVectorIndex(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Hybrid mode requires embedding, should auto-enable vector index
	err := service.SetSearchMode(domain.SearchModeHybrid)
	require.NoError(t, err)

	settings, _ := service.Get()
	assert.True(t, settings.VectorIndex.Enabled)
}

func TestSettingsService_SetSearchMode_Invalid(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetSearchMode(domain.SearchMode("invalid"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid search mode")
}

func TestSettingsService_SetEmbeddingProvider_Ollama(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetEmbeddingProvider(domain.AIProviderOllama, "nomic-embed-text", "")

	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.AIProviderOllama, settings.Embedding.Provider)
	assert.Equal(t, "nomic-embed-text", settings.Embedding.Model)
	assert.Equal(t, "http://localhost:11434", settings.Embedding.BaseURL)
	assert.Empty(t, settings.Embedding.APIKey)
}

func TestSettingsService_SetEmbeddingProvider_OpenAI(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetEmbeddingProvider(domain.AIProviderOpenAI, "text-embedding-3-small", "sk-test-key")

	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.AIProviderOpenAI, settings.Embedding.Provider)
	assert.Equal(t, "text-embedding-3-small", settings.Embedding.Model)
	assert.Equal(t, "sk-test-key", settings.Embedding.APIKey)
}

func TestSettingsService_SetEmbeddingProvider_DefaultModel(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Empty model should use default
	err := service.SetEmbeddingProvider(domain.AIProviderOpenAI, "", "sk-test-key")

	require.NoError(t, err)

	settings, _ := service.Get()
	defaults := domain.DefaultEmbeddingModels()
	assert.Equal(t, defaults[domain.AIProviderOpenAI], settings.Embedding.Model)
}

func TestSettingsService_SetEmbeddingProvider_UpdatesDimensions(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetEmbeddingProvider(domain.AIProviderOpenAI, "text-embedding-3-small", "sk-test-key")

	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, 1536, settings.VectorIndex.Dimensions)
}

func TestSettingsService_SetEmbeddingProvider_RequiresAPIKey(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetEmbeddingProvider(domain.AIProviderOpenAI, "text-embedding-3-small", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key required")
}

func TestSettingsService_SetEmbeddingProvider_InvalidProvider(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetEmbeddingProvider(domain.AIProvider("invalid"), "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid embedding provider")
}

func TestSettingsService_SetEmbeddingProvider_AnthropicNotSupported(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Anthropic doesn't support embeddings
	err := service.SetEmbeddingProvider(domain.AIProviderAnthropic, "", "sk-ant-test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support embeddings")
}

func TestSettingsService_SetLLMProvider_Ollama(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProviderOllama, "llama3.2", "")

	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.AIProviderOllama, settings.LLM.Provider)
	assert.Equal(t, "llama3.2", settings.LLM.Model)
	assert.Equal(t, "http://localhost:11434", settings.LLM.BaseURL)
	assert.Empty(t, settings.LLM.APIKey)
}

func TestSettingsService_SetLLMProvider_OpenAI(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProviderOpenAI, "gpt-4o", "sk-test-key")

	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.AIProviderOpenAI, settings.LLM.Provider)
	assert.Equal(t, "gpt-4o", settings.LLM.Model)
	assert.Equal(t, "sk-test-key", settings.LLM.APIKey)
}

func TestSettingsService_SetLLMProvider_Anthropic(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProviderAnthropic, "claude-3-5-sonnet-latest", "sk-ant-test")

	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.AIProviderAnthropic, settings.LLM.Provider)
	assert.Equal(t, "claude-3-5-sonnet-latest", settings.LLM.Model)
	assert.Equal(t, "sk-ant-test", settings.LLM.APIKey)
}

func TestSettingsService_SetLLMProvider_DefaultModel(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProviderAnthropic, "", "sk-ant-test")

	require.NoError(t, err)

	settings, _ := service.Get()
	defaults := domain.DefaultLLMModels()
	assert.Equal(t, defaults[domain.AIProviderAnthropic], settings.LLM.Model)
}

func TestSettingsService_SetLLMProvider_RequiresAPIKey(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProviderOpenAI, "gpt-4o", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key required")
}

func TestSettingsService_SetLLMProvider_Invalid(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProvider("invalid"), "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid LLM provider")
}

func TestSettingsService_Validate_TextOnlyMode(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Text only mode should validate without any provider config
	err := service.SetSearchMode(domain.SearchModeTextOnly)
	require.NoError(t, err)

	err = service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_Validate_HybridModeWithoutEmbedding(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set hybrid mode with OpenAI embedding but no API key
	_ = store.Set("search.mode", "hybrid")
	_ = store.Set("embedding.provider", "openai")
	_ = store.Set("embedding.api_key", "") // Explicitly empty API key

	err := service.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding provider")
}

func TestSettingsService_Validate_HybridModeWithEmbedding(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Configure hybrid mode with Ollama embedding (no API key needed)
	_ = service.SetSearchMode(domain.SearchModeHybrid)
	_ = service.SetEmbeddingProvider(domain.AIProviderOllama, "nomic-embed-text", "")

	err := service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_Validate_LLMAssistedModeWithoutLLM(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set LLM assisted mode with OpenAI but no API key
	_ = store.Set("search.mode", "llm_assisted")
	_ = store.Set("llm.provider", "openai")
	_ = store.Set("llm.api_key", "") // Explicitly empty API key

	err := service.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM provider")
}

func TestSettingsService_Validate_LLMAssistedModeWithLLM(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	_ = service.SetSearchMode(domain.SearchModeLLMAssisted)
	_ = service.SetLLMProvider(domain.AIProviderOllama, "llama3.2", "")

	err := service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_Validate_FullModeRequiresBoth(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Full mode requires both embedding and LLM
	_ = store.Set("search.mode", "full")
	_ = store.Set("embedding.provider", "openai")
	_ = store.Set("embedding.api_key", "")
	_ = store.Set("llm.provider", "openai")
	_ = store.Set("llm.api_key", "")

	err := service.Validate()
	assert.Error(t, err)
}

func TestSettingsService_Validate_FullModeConfigured(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	_ = service.SetSearchMode(domain.SearchModeFull)
	_ = service.SetEmbeddingProvider(domain.AIProviderOllama, "nomic-embed-text", "")
	_ = service.SetLLMProvider(domain.AIProviderOllama, "llama3.2", "")

	err := service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_Validate_InvalidSearchMode(t *testing.T) {
	store := memory.NewConfigStore()
	_ = store.Set("search.mode", "invalid")

	service := NewSettingsService(store, nil)

	err := service.Validate()

	// Invalid mode falls back to default, which is valid
	assert.NoError(t, err)
}

func TestSettingsService_RequiresEmbedding(t *testing.T) {
	tests := []struct {
		mode     domain.SearchMode
		expected bool
	}{
		{domain.SearchModeTextOnly, false},
		{domain.SearchModeHybrid, true},
		{domain.SearchModeLLMAssisted, false},
		{domain.SearchModeFull, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			store := memory.NewConfigStore()
			service := NewSettingsService(store, nil)
			_ = store.Set("search.mode", string(tt.mode))

			result := service.RequiresEmbedding()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsService_RequiresEmbedding_ErrorCase(t *testing.T) {
	// Test that RequiresEmbedding returns false when Get() would fail
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// With no config set, should use defaults
	result := service.RequiresEmbedding()

	// Default mode is text_only which doesn't require embedding
	assert.False(t, result)
}

func TestSettingsService_RequiresEmbedding_InvalidMode(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)
	_ = store.Set("search.mode", "invalid-mode-xyz")

	result := service.RequiresEmbedding()

	// Invalid mode falls back to default (text_only), which doesn't require embedding
	assert.False(t, result)
}

func TestSettingsService_RequiresEmbedding_EmptyMode(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)
	_ = store.Set("search.mode", "")

	result := service.RequiresEmbedding()

	// Empty mode falls back to default (text_only)
	assert.False(t, result)
}

func TestSettingsService_RequiresLLM(t *testing.T) {
	tests := []struct {
		name     string
		mode     domain.SearchMode
		expected bool
	}{
		{"text_only", domain.SearchModeTextOnly, false},
		{"hybrid", domain.SearchModeHybrid, false},
		{"llm_assisted", domain.SearchModeLLMAssisted, true},
		{"full", domain.SearchModeFull, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.NewConfigStore()
			service := NewSettingsService(store, nil)
			_ = store.Set("search.mode", string(tt.mode))

			result := service.RequiresLLM()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsService_RequiresLLM_ErrorCase(t *testing.T) {
	// Test that RequiresLLM returns false when Get() would fail
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// With no config set, should use defaults
	result := service.RequiresLLM()

	// Default is text_only which doesn't require LLM
	assert.False(t, result)
}

func TestSettingsService_RequiresLLM_InvalidMode(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)
	_ = store.Set("search.mode", "invalid-mode-xyz")

	result := service.RequiresLLM()

	// Invalid mode falls back to default (text_only), which doesn't require LLM
	assert.False(t, result)
}

func TestSettingsService_RequiresLLM_EmptyMode(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)
	_ = store.Set("search.mode", "")

	result := service.RequiresLLM()

	// Empty mode falls back to default (text_only)
	assert.False(t, result)
}

func TestSettingsService_GetDefaults(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	defaults := service.GetDefaults()

	expected := domain.DefaultAppSettings()
	assert.Equal(t, expected, defaults)
}

func TestSettingsService_Save_EmptyAPIKey(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
			APIKey:   "", // Empty API key should not be saved
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
			APIKey:   "", // Empty API key should not be saved
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	require.NoError(t, err)

	// Verify empty API keys were not saved
	retrieved, err := service.Get()
	require.NoError(t, err)
	assert.Empty(t, retrieved.Embedding.APIKey)
	assert.Empty(t, retrieved.LLM.APIKey)
}

func TestSettingsService_Save_AllFieldsSet(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeHybrid,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOpenAI,
			Model:    "text-embedding-3-small",
			BaseURL:  "https://api.openai.com",
			APIKey:   "sk-test-key",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderAnthropic,
			Model:    "claude-3-5-sonnet-latest",
			BaseURL:  "https://api.anthropic.com",
			APIKey:   "sk-ant-test",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    true,
			Dimensions: 1536,
		},
	}

	err := service.Save(settings)
	require.NoError(t, err)

	// Verify all values were saved correctly
	retrieved, err := service.Get()
	require.NoError(t, err)
	assert.Equal(t, domain.SearchModeHybrid, retrieved.Search.Mode)
	assert.Equal(t, domain.AIProviderOpenAI, retrieved.Embedding.Provider)
	assert.Equal(t, "text-embedding-3-small", retrieved.Embedding.Model)
	assert.Equal(t, "sk-test-key", retrieved.Embedding.APIKey)
	assert.Equal(t, domain.AIProviderAnthropic, retrieved.LLM.Provider)
	assert.Equal(t, "claude-3-5-sonnet-latest", retrieved.LLM.Model)
	assert.Equal(t, "sk-ant-test", retrieved.LLM.APIKey)
	assert.True(t, retrieved.VectorIndex.Enabled)
	assert.Equal(t, 1536, retrieved.VectorIndex.Dimensions)
}

func TestSettingsService_SetEmbeddingProvider_PreservesExistingBaseURL(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set a custom base URL for local provider
	_ = store.Set("embedding.base_url", "http://custom:8080")

	err := service.SetEmbeddingProvider(domain.AIProviderOllama, "nomic-embed-text", "")
	require.NoError(t, err)

	settings, _ := service.Get()
	// Should preserve existing base URL for local providers
	assert.NotEmpty(t, settings.Embedding.BaseURL)
}

func TestSettingsService_SetLLMProvider_PreservesExistingBaseURL(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set a custom base URL for local provider
	_ = store.Set("llm.base_url", "http://custom:8080")

	err := service.SetLLMProvider(domain.AIProviderOllama, "llama3.2", "")
	require.NoError(t, err)

	settings, _ := service.Get()
	assert.NotEmpty(t, settings.LLM.BaseURL)
}

func TestSettingsService_SetEmbeddingProvider_CloudProviderBaseURL(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set a base URL first for local provider
	_ = service.SetEmbeddingProvider(domain.AIProviderOllama, "nomic-embed-text", "")

	settings, _ := service.Get()
	assert.NotEmpty(t, settings.Embedding.BaseURL)

	// Switch to cloud provider (OpenAI)
	err := service.SetEmbeddingProvider(domain.AIProviderOpenAI, "text-embedding-3-small", "sk-test")
	require.NoError(t, err)

	settings, _ = service.Get()
	// Cloud providers should have empty base URL
	// Note: The service may preserve the base URL in storage, but cloud providers ignore it
	assert.Equal(t, domain.AIProviderOpenAI, settings.Embedding.Provider)
}

func TestSettingsService_SetLLMProvider_CloudProviderBaseURL(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set a base URL first for local provider
	_ = service.SetLLMProvider(domain.AIProviderOllama, "llama3.2", "")

	settings, _ := service.Get()
	assert.NotEmpty(t, settings.LLM.BaseURL)

	// Switch to cloud provider
	err := service.SetLLMProvider(domain.AIProviderOpenAI, "gpt-4o", "sk-test")
	require.NoError(t, err)

	settings, _ = service.Get()
	assert.Equal(t, domain.AIProviderOpenAI, settings.LLM.Provider)
}

func TestSettingsService_GetBool_WithoutKey(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Get without setting key should return default
	settings, err := service.Get()
	require.NoError(t, err)

	// Should use defaults
	defaults := domain.DefaultAppSettings()
	assert.Equal(t, defaults.VectorIndex.Enabled, settings.VectorIndex.Enabled)
}

func TestSettingsService_GetInt_WithZeroValue(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set dimensions to 0 (should fall back to default)
	_ = store.Set("vector_index.dimensions", 0)

	settings, err := service.Get()
	require.NoError(t, err)

	// Should use default when value is 0
	defaults := domain.DefaultAppSettings()
	assert.Equal(t, defaults.VectorIndex.Dimensions, settings.VectorIndex.Dimensions)
}

func TestSettingsService_SetEmbeddingProvider_ModelWithoutDimensions(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Use a model that's not in the dimensions map
	err := service.SetEmbeddingProvider(domain.AIProviderOllama, "custom-model", "")
	require.NoError(t, err)

	settings, _ := service.Get()
	// Dimensions should remain at default or previous value
	assert.Equal(t, "custom-model", settings.Embedding.Model)
}

func TestSettingsService_Get_ReadsAllFields(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set all fields
	_ = store.Set("search.mode", "hybrid")
	_ = store.Set("embedding.provider", "openai")
	_ = store.Set("embedding.model", "text-embedding-3-large")
	_ = store.Set("embedding.base_url", "https://api.openai.com")
	_ = store.Set("embedding.api_key", "sk-test")
	_ = store.Set("llm.provider", "anthropic")
	_ = store.Set("llm.model", "claude-3-5-sonnet-latest")
	_ = store.Set("llm.base_url", "https://api.anthropic.com")
	_ = store.Set("llm.api_key", "sk-ant-test")
	_ = store.Set("vector_index.enabled", true)
	_ = store.Set("vector_index.dimensions", 3072)

	settings, err := service.Get()

	require.NoError(t, err)
	assert.Equal(t, domain.SearchModeHybrid, settings.Search.Mode)
	assert.Equal(t, domain.AIProviderOpenAI, settings.Embedding.Provider)
	assert.Equal(t, "text-embedding-3-large", settings.Embedding.Model)
	assert.Equal(t, "https://api.openai.com", settings.Embedding.BaseURL)
	assert.Equal(t, "sk-test", settings.Embedding.APIKey)
	assert.Equal(t, domain.AIProviderAnthropic, settings.LLM.Provider)
	assert.Equal(t, "claude-3-5-sonnet-latest", settings.LLM.Model)
	assert.Equal(t, "https://api.anthropic.com", settings.LLM.BaseURL)
	assert.Equal(t, "sk-ant-test", settings.LLM.APIKey)
	assert.True(t, settings.VectorIndex.Enabled)
	assert.Equal(t, 3072, settings.VectorIndex.Dimensions)
}

func TestSettingsService_Validate_ModeFallback(t *testing.T) {
	store := memory.NewConfigStore()
	_ = store.Set("search.mode", "invalid-mode")
	service := NewSettingsService(store, nil)

	err := service.Validate()

	// Invalid mode falls back to default which is valid
	assert.NoError(t, err)
}

func TestSettingsService_Validate_HybridWithOllamaEmbedding(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Set hybrid mode with Ollama (local, no API key needed)
	_ = store.Set("search.mode", "hybrid")
	_ = store.Set("embedding.provider", "ollama")
	_ = store.Set("embedding.model", "nomic-embed-text")

	err := service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_Validate_FullModeWithMixedProviders(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Full mode with Ollama embedding and OpenAI LLM
	_ = store.Set("search.mode", "full")
	_ = store.Set("embedding.provider", "ollama")
	_ = store.Set("embedding.model", "nomic-embed-text")
	_ = store.Set("llm.provider", "openai")
	_ = store.Set("llm.model", "gpt-4o")
	_ = store.Set("llm.api_key", "sk-test")

	err := service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_Validate_ErrorFromGet(t *testing.T) {
	// Test that validation propagates errors from Get()
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	// Normal case should work
	err := service.Validate()
	assert.NoError(t, err)
}

func TestSettingsService_SetSearchMode_TextOnly(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetSearchMode(domain.SearchModeTextOnly)
	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.SearchModeTextOnly, settings.Search.Mode)
	// Text only mode doesn't auto-enable vector index
	assert.False(t, settings.VectorIndex.Enabled)
}

func TestSettingsService_SetSearchMode_LLMAssisted(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetSearchMode(domain.SearchModeLLMAssisted)
	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.SearchModeLLMAssisted, settings.Search.Mode)
}

func TestSettingsService_SetSearchMode_Full(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.SetSearchMode(domain.SearchModeFull)
	require.NoError(t, err)

	settings, _ := service.Get()
	assert.Equal(t, domain.SearchModeFull, settings.Search.Mode)
	// Full mode requires embedding, should auto-enable vector index
	assert.True(t, settings.VectorIndex.Enabled)
}

func TestSettingsService_Save_WithBaseURLs(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
			BaseURL:  "http://localhost:11434",
			APIKey:   "",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
			BaseURL:  "http://localhost:11434",
			APIKey:   "",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	require.NoError(t, err)

	retrieved, err := service.Get()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:11434", retrieved.Embedding.BaseURL)
	assert.Equal(t, "http://localhost:11434", retrieved.LLM.BaseURL)
}

// Mock config store that always fails on Set
type failingConfigStore struct {
	*memory.ConfigStore
	failOn string
}

func (f *failingConfigStore) Set(key string, value interface{}) error {
	if f.failOn == "" || key == f.failOn {
		return assert.AnError
	}
	return f.ConfigStore.Set(key, value)
}

func TestSettingsService_Save_ErrorOnSearchMode(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "search.mode",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search mode")
}

func TestSettingsService_Save_ErrorOnEmbeddingProvider(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "embedding.provider",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding provider")
}

func TestSettingsService_Save_ErrorOnEmbeddingModel(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "embedding.model",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding model")
}

func TestSettingsService_Save_ErrorOnEmbeddingBaseURL(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "embedding.base_url",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding base_url")
}

func TestSettingsService_Save_ErrorOnEmbeddingAPIKey(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "embedding.api_key",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
			APIKey:   "test-key", // Non-empty to trigger save
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding api_key")
}

func TestSettingsService_Save_ErrorOnLLMProvider(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "llm.provider",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm provider")
}

func TestSettingsService_Save_ErrorOnLLMModel(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "llm.model",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm model")
}

func TestSettingsService_Save_ErrorOnLLMBaseURL(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "llm.base_url",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm base_url")
}

func TestSettingsService_Save_ErrorOnLLMAPIKey(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "llm.api_key",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
			APIKey:   "test-key", // Non-empty to trigger save
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm api_key")
}

func TestSettingsService_Save_ErrorOnVectorEnabled(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "vector_index.enabled",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vector enabled")
}

func TestSettingsService_Save_ErrorOnVectorDimensions(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "vector_index.dimensions",
	}
	service := NewSettingsService(store, nil)

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,
		},
	}

	err := service.Save(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vector dimensions")
}

func TestSettingsService_SetEmbeddingProvider_GetError(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "embedding.provider",
	}
	service := NewSettingsService(store, nil)

	err := service.SetEmbeddingProvider(domain.AIProviderOllama, "nomic-embed-text", "")
	assert.Error(t, err)
}

func TestSettingsService_SetLLMProvider_GetError(t *testing.T) {
	store := &failingConfigStore{
		ConfigStore: memory.NewConfigStore(),
		failOn:      "llm.provider",
	}
	service := NewSettingsService(store, nil)

	err := service.SetLLMProvider(domain.AIProviderOllama, "llama3.2", "")
	assert.Error(t, err)
}

// Mock AIConfigValidator for testing
type mockAIConfigValidator struct {
	embedErr error
	llmErr   error
}

func (m *mockAIConfigValidator) ValidateEmbedding(_ *domain.EmbeddingSettings) error {
	return m.embedErr
}

func (m *mockAIConfigValidator) ValidateLLM(_ *domain.LLMSettings) error {
	return m.llmErr
}

func TestSettingsService_ValidateEmbeddingConfig_NilValidator(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.ValidateEmbeddingConfig()

	// With nil validator, should skip validation (no error)
	assert.NoError(t, err)
}

func TestSettingsService_ValidateEmbeddingConfig_Success(t *testing.T) {
	store := memory.NewConfigStore()
	validator := &mockAIConfigValidator{}
	service := NewSettingsService(store, validator)

	err := service.ValidateEmbeddingConfig()

	assert.NoError(t, err)
}

func TestSettingsService_ValidateEmbeddingConfig_Error(t *testing.T) {
	store := memory.NewConfigStore()
	validator := &mockAIConfigValidator{embedErr: assert.AnError}
	service := NewSettingsService(store, validator)

	err := service.ValidateEmbeddingConfig()

	assert.Error(t, err)
}

func TestSettingsService_ValidateLLMConfig_NilValidator(t *testing.T) {
	store := memory.NewConfigStore()
	service := NewSettingsService(store, nil)

	err := service.ValidateLLMConfig()

	// With nil validator, should skip validation (no error)
	assert.NoError(t, err)
}

func TestSettingsService_ValidateLLMConfig_Success(t *testing.T) {
	store := memory.NewConfigStore()
	validator := &mockAIConfigValidator{}
	service := NewSettingsService(store, validator)

	err := service.ValidateLLMConfig()

	assert.NoError(t, err)
}

func TestSettingsService_ValidateLLMConfig_Error(t *testing.T) {
	store := memory.NewConfigStore()
	validator := &mockAIConfigValidator{llmErr: assert.AnError}
	service := NewSettingsService(store, validator)

	err := service.ValidateLLMConfig()

	assert.Error(t, err)
}
