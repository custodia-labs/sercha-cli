package ai

import (
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure ConfigValidator implements the interface.
var _ driven.AIConfigValidator = (*ConfigValidator)(nil)

// ConfigValidator validates AI provider configurations.
type ConfigValidator struct{}

// NewConfigValidator creates a new AI config validator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateEmbedding validates an embedding configuration by pinging the provider.
func (v *ConfigValidator) ValidateEmbedding(config *domain.EmbeddingSettings) error {
	return ValidateEmbeddingConfig(config)
}

// ValidateLLM validates an LLM configuration by pinging the provider.
func (v *ConfigValidator) ValidateLLM(config *domain.LLMSettings) error {
	return ValidateLLMConfig(config)
}
