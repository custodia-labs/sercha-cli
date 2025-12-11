package driven

import "github.com/custodia-labs/sercha-cli/internal/core/domain"

// AIConfigValidator validates AI provider configurations.
// Implementations verify that configurations are valid by testing connectivity
// to the underlying AI services.
type AIConfigValidator interface {
	// ValidateEmbedding validates an embedding configuration by pinging the provider.
	// Returns nil if configuration is valid or not configured.
	ValidateEmbedding(config *domain.EmbeddingSettings) error

	// ValidateLLM validates an LLM configuration by pinging the provider.
	// Returns nil if configuration is valid or not configured.
	ValidateLLM(config *domain.LLMSettings) error
}
