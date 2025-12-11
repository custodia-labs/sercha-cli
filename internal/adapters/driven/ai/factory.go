// Package ai provides factory functions for creating AI service adapters.
package ai

import (
	"context"
	"fmt"
	"time"

	ollamaembed "github.com/custodia-labs/sercha-cli/internal/adapters/driven/embedding/ollama"
	openaiembed "github.com/custodia-labs/sercha-cli/internal/adapters/driven/embedding/openai"
	anthropicllm "github.com/custodia-labs/sercha-cli/internal/adapters/driven/llm/anthropic"
	ollamallm "github.com/custodia-labs/sercha-cli/internal/adapters/driven/llm/ollama"
	openaillm "github.com/custodia-labs/sercha-cli/internal/adapters/driven/llm/openai"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// pingTimeout is the maximum time to wait for service connectivity validation.
const pingTimeout = 5 * time.Second

// InitResult contains the result of AI service initialisation.
type InitResult struct {
	EmbeddingService driven.EmbeddingService
	LLMService       driven.LLMService
	VectorIndex      driven.VectorIndex
	PromptStore      driven.PromptStore // User-customisable prompt templates.
	Warnings         []string           // Non-fatal issues that caused fallback.
	FellBack         bool               // True if fell back to text-only mode.
}

// Close releases all resources held by InitResult.
func (r *InitResult) Close() {
	if r.EmbeddingService != nil {
		r.EmbeddingService.Close()
	}
	if r.VectorIndex != nil {
		r.VectorIndex.Close()
	}
	if r.LLMService != nil {
		r.LLMService.Close()
	}
}

// CreateAndValidateEmbeddingService creates an embedding service and validates connectivity.
// Returns the service if successful, or an error with guidance.
func CreateAndValidateEmbeddingService(settings *domain.EmbeddingSettings) (driven.EmbeddingService, error) {
	if settings == nil || !settings.IsConfigured() {
		return nil, nil
	}

	svc, err := CreateEmbeddingService(settings)
	if err != nil {
		return nil, fmt.Errorf("%w: %w. Run 'sercha settings wizard' to fix",
			domain.ErrEmbeddingUnavailable, err)
	}

	if svc == nil {
		return nil, nil
	}

	// Validate connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := svc.Ping(ctx); err != nil {
		svc.Close()
		return nil, fmt.Errorf("%w: service unreachable (%w). Run 'sercha settings wizard' to fix",
			domain.ErrEmbeddingUnavailable, err)
	}

	return svc, nil
}

// CreateAndValidateLLMService creates an LLM service and validates connectivity.
// Returns the service if successful, or an error with guidance.
func CreateAndValidateLLMService(settings *domain.LLMSettings) (driven.LLMService, error) {
	if settings == nil || !settings.IsConfigured() {
		return nil, nil
	}

	svc, err := CreateLLMService(settings)
	if err != nil {
		return nil, fmt.Errorf("%w: %w. Run 'sercha settings wizard' to fix",
			domain.ErrLLMUnavailable, err)
	}

	if svc == nil {
		return nil, nil
	}

	// Validate connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := svc.Ping(ctx); err != nil {
		svc.Close()
		return nil, fmt.Errorf("%w: service unreachable (%w). Run 'sercha settings wizard' to fix",
			domain.ErrLLMUnavailable, err)
	}

	return svc, nil
}

// ValidateEmbeddingConfig validates an embedding configuration by creating a service and pinging it.
// This is intended for use in the settings wizard to validate credentials on configuration.
func ValidateEmbeddingConfig(settings *domain.EmbeddingSettings) error {
	if settings == nil || !settings.IsConfigured() {
		return nil
	}

	svc, err := CreateEmbeddingService(settings)
	if err != nil {
		return err
	}
	if svc == nil {
		return nil
	}
	defer svc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	return svc.Ping(ctx)
}

// ValidateLLMConfig validates an LLM configuration by creating a service and pinging it.
// This is intended for use in the settings wizard to validate credentials on configuration.
func ValidateLLMConfig(settings *domain.LLMSettings) error {
	if settings == nil || !settings.IsConfigured() {
		return nil
	}

	svc, err := CreateLLMService(settings)
	if err != nil {
		return err
	}
	if svc == nil {
		return nil
	}
	defer svc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	return svc.Ping(ctx)
}

// CreateEmbeddingService creates the appropriate embedding service based on settings.
// Returns nil if the provider is not configured.
func CreateEmbeddingService(settings *domain.EmbeddingSettings) (driven.EmbeddingService, error) {
	if settings == nil || !settings.IsConfigured() {
		return nil, nil
	}

	switch settings.Provider {
	case domain.AIProviderOllama:
		return createOllamaEmbedding(settings), nil

	case domain.AIProviderOpenAI:
		return createOpenAIEmbedding(settings)

	case domain.AIProviderAnthropic:
		// Anthropic does not support embeddings.
		return nil, fmt.Errorf("anthropic does not support embeddings, use ollama or openai")

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", settings.Provider)
	}
}

// CreateLLMService creates the appropriate LLM service based on settings.
// Returns nil if the provider is not configured.
func CreateLLMService(settings *domain.LLMSettings) (driven.LLMService, error) {
	if settings == nil || !settings.IsConfigured() {
		return nil, nil
	}

	switch settings.Provider {
	case domain.AIProviderOllama:
		return createOllamaLLM(settings), nil

	case domain.AIProviderOpenAI:
		return createOpenAILLM(settings)

	case domain.AIProviderAnthropic:
		return createAnthropicLLM(settings)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", settings.Provider)
	}
}

// createOllamaEmbedding creates an Ollama embedding service.
func createOllamaEmbedding(settings *domain.EmbeddingSettings) driven.EmbeddingService {
	dimensions := domain.EmbeddingDimensions()[settings.Model]
	if dimensions == 0 {
		dimensions = ollamaembed.DefaultDimensions
	}

	return ollamaembed.NewEmbeddingService(ollamaembed.Config{
		BaseURL:    settings.BaseURL,
		Model:      settings.Model,
		Dimensions: dimensions,
	})
}

// createOpenAIEmbedding creates an OpenAI embedding service.
func createOpenAIEmbedding(settings *domain.EmbeddingSettings) (driven.EmbeddingService, error) {
	dimensions := domain.EmbeddingDimensions()[settings.Model]

	return openaiembed.NewEmbeddingService(openaiembed.Config{
		APIKey:     settings.APIKey,
		BaseURL:    settings.BaseURL,
		Model:      settings.Model,
		Dimensions: dimensions,
	})
}

// createOllamaLLM creates an Ollama LLM service.
func createOllamaLLM(settings *domain.LLMSettings) driven.LLMService {
	return ollamallm.NewLLMService(ollamallm.LLMConfig{
		BaseURL: settings.BaseURL,
		Model:   settings.Model,
	})
}

// createOpenAILLM creates an OpenAI LLM service.
func createOpenAILLM(settings *domain.LLMSettings) (driven.LLMService, error) {
	return openaillm.NewLLMService(openaillm.LLMConfig{
		APIKey:  settings.APIKey,
		BaseURL: settings.BaseURL,
		Model:   settings.Model,
	})
}

// createAnthropicLLM creates an Anthropic LLM service.
func createAnthropicLLM(settings *domain.LLMSettings) (driven.LLMService, error) {
	return anthropicllm.NewLLMService(anthropicllm.Config{
		APIKey:  settings.APIKey,
		BaseURL: settings.BaseURL,
		Model:   settings.Model,
	})
}
