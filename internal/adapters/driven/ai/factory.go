// Package ai provides factory functions for creating AI service adapters.
package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-cli/cgo/hnsw"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/config/file"
	ollamaembed "github.com/custodia-labs/sercha-cli/internal/adapters/driven/embedding/ollama"
	openaiembed "github.com/custodia-labs/sercha-cli/internal/adapters/driven/embedding/openai"
	anthropicllm "github.com/custodia-labs/sercha-cli/internal/adapters/driven/llm/anthropic"
	ollamallm "github.com/custodia-labs/sercha-cli/internal/adapters/driven/llm/ollama"
	openaillm "github.com/custodia-labs/sercha-cli/internal/adapters/driven/llm/openai"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/logger"
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

// InitialiseServices creates AI services with auto-fallback on failure.
// If services required by settings fail, falls back to text-only mode and logs warnings.
// The caller should check result.FellBack and result.Warnings to inform the user.
func InitialiseServices(settings *domain.AppSettings, vectorPath string) (*InitResult, error) {
	logger.Section("AI Service Initialisation")
	logger.Debug("Search mode: %s", settings.Search.Mode.Description())
	logger.Debug("Vector index path: %s", vectorPath)

	result := &InitResult{}

	// Create prompt store for user-customisable prompts (non-critical if fails).
	promptStore, err := file.NewPromptStore("")
	if err == nil {
		result.PromptStore = promptStore
		logger.Debug("Prompt store: loaded")
	} else {
		logger.Debug("Prompt store: not available (%v)", err)
	}

	// Try to create embedding service if mode requires it (no validation - done in wizard).
	if settings.Search.Mode.RequiresEmbedding() {
		logger.Debug("Embedding required: yes (mode=%s)", settings.Search.Mode)
		logger.Debug("Embedding provider: %s", settings.Embedding.Provider.Description())
		logger.Debug("Embedding model: %s", settings.Embedding.Model)

		svc, err := CreateEmbeddingService(&settings.Embedding)
		if err != nil {
			logger.Warn("Embedding service failed: %v", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("Embedding: %v", err))
			result.FellBack = true
		} else if svc != nil {
			logger.Info("Embedding service: created (dimensions=%d)", svc.Dimensions())
			result.EmbeddingService = svc
		}
	} else {
		logger.Debug("Embedding required: no")
	}

	// Create vector index only if embedding service available.
	if result.EmbeddingService != nil && vectorPath != "" {
		precision := domainToHNSWPrecision(settings.VectorIndex.Precision)
		logger.Debug("Creating vector index: path=%s, dims=%d, precision=%v",
			vectorPath, result.EmbeddingService.Dimensions(), precision)

		idx, err := hnsw.New(vectorPath, result.EmbeddingService.Dimensions(), precision)
		if err != nil {
			logger.Warn("Vector index failed: %v", err)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Vector index: %v. Run 'sercha settings wizard' to fix", err))
			result.EmbeddingService.Close()
			result.EmbeddingService = nil
			result.FellBack = true
		} else {
			logger.Info("Vector index: created")
			result.VectorIndex = idx
		}
	}

	// Try to create LLM service if mode requires it.
	if settings.Search.Mode.RequiresLLM() {
		logger.Debug("LLM required: yes (mode=%s)", settings.Search.Mode)
		logger.Debug("LLM provider: %s", settings.LLM.Provider.Description())
		logger.Debug("LLM model: %s", settings.LLM.Model)
		initLLMService(result, &settings.LLM)
	} else {
		logger.Debug("LLM required: no")
	}

	if result.FellBack {
		logger.Warn("Fell back to text-only mode due to service failures")
	}

	return result, nil
}

// initLLMService creates and configures the LLM service, updating result accordingly.
func initLLMService(result *InitResult, settings *domain.LLMSettings) {
	svc, err := CreateLLMService(settings)
	if err != nil {
		logger.Warn("LLM service failed: %v", err)
		result.Warnings = append(result.Warnings, fmt.Sprintf("LLM: %v", err))
		result.FellBack = true
		return
	}
	if svc == nil {
		logger.Debug("LLM service: not configured")
		return
	}

	logger.Info("LLM service: created")
	result.LLMService = svc
	injectPromptStore(svc, result.PromptStore)
}

// injectPromptStore sets the prompt store on services that support it.
func injectPromptStore(svc driven.LLMService, store driven.PromptStore) {
	if store == nil {
		return
	}
	if aware, ok := svc.(driven.PromptStoreAware); ok {
		aware.SetPromptStore(store)
	}
}

// domainToHNSWPrecision converts domain VectorPrecision to HNSW Precision.
func domainToHNSWPrecision(p domain.VectorPrecision) hnsw.Precision {
	switch p {
	case domain.VectorPrecisionFloat32:
		return hnsw.PrecisionFloat32
	case domain.VectorPrecisionFloat16:
		return hnsw.PrecisionFloat16
	case domain.VectorPrecisionInt8:
		return hnsw.PrecisionInt8
	default:
		return hnsw.PrecisionFloat16
	}
}
