package driving

import "github.com/custodia-labs/sercha-cli/internal/core/domain"

// SettingsService manages application settings.
type SettingsService interface {
	// Get retrieves current application settings.
	Get() (*domain.AppSettings, error)

	// Save persists application settings.
	Save(settings *domain.AppSettings) error

	// SetSearchMode updates the search mode.
	SetSearchMode(mode domain.SearchMode) error

	// SetEmbeddingProvider configures the embedding provider.
	SetEmbeddingProvider(provider domain.AIProvider, model, apiKey string) error

	// SetLLMProvider configures the LLM provider.
	SetLLMProvider(provider domain.AIProvider, model, apiKey string) error

	// Validate checks if current settings are valid for the configured mode.
	Validate() error

	// RequiresEmbedding returns true if current mode needs embedding.
	RequiresEmbedding() bool

	// RequiresLLM returns true if current mode needs LLM.
	RequiresLLM() bool

	// GetDefaults returns default settings.
	GetDefaults() domain.AppSettings

	// ValidateEmbeddingConfig validates the current embedding configuration by pinging the provider.
	ValidateEmbeddingConfig() error

	// ValidateLLMConfig validates the current LLM configuration by pinging the provider.
	ValidateLLMConfig() error
}
