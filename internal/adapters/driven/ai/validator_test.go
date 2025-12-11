package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNewConfigValidator(t *testing.T) {
	validator := NewConfigValidator()

	require.NotNil(t, validator)
}

func TestConfigValidator_ImplementsInterface(t *testing.T) {
	var _ driven.AIConfigValidator = (*ConfigValidator)(nil)
}

func TestConfigValidator_ValidateEmbedding_NilConfig(t *testing.T) {
	validator := NewConfigValidator()

	err := validator.ValidateEmbedding(nil)

	// nil config returns nil (graceful handling - nothing to validate)
	assert.NoError(t, err)
}

func TestConfigValidator_ValidateEmbedding_UnconfiguredProvider(t *testing.T) {
	validator := NewConfigValidator()
	config := &domain.EmbeddingSettings{
		Provider: "",
		Model:    "test-model",
	}

	err := validator.ValidateEmbedding(config)

	// Unconfigured provider returns nil (nothing to validate)
	assert.NoError(t, err)
}

func TestConfigValidator_ValidateLLM_NilConfig(t *testing.T) {
	validator := NewConfigValidator()

	err := validator.ValidateLLM(nil)

	// nil config returns nil (graceful handling - nothing to validate)
	assert.NoError(t, err)
}

func TestConfigValidator_ValidateLLM_UnconfiguredProvider(t *testing.T) {
	validator := NewConfigValidator()
	config := &domain.LLMSettings{
		Provider: "",
		Model:    "test-model",
	}

	err := validator.ValidateLLM(config)

	// Unconfigured provider returns nil (nothing to validate)
	assert.NoError(t, err)
}
