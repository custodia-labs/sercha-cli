// Package openai provides an embedding service adapter using OpenAI API.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure EmbeddingService implements the interface.
var _ driven.EmbeddingService = (*EmbeddingService)(nil)

// Default configuration values.
const (
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "text-embedding-3-small"
	DefaultTimeout = 60 * time.Second
)

// Model dimensions for OpenAI embedding models.
var modelDimensions = map[string]int{
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,
}

// Config holds configuration for the OpenAI embedding service.
type Config struct {
	// APIKey is the OpenAI API key (required).
	APIKey string

	// BaseURL is the API base URL (default: https://api.openai.com/v1).
	// Can be changed for Azure OpenAI or compatible APIs.
	BaseURL string

	// Model is the embedding model to use (default: text-embedding-3-small).
	Model string

	// Timeout is the request timeout (default: 60s).
	Timeout time.Duration

	// Dimensions overrides the default dimension for the model.
	// Only applicable to text-embedding-3-* models.
	Dimensions int
}

// EmbeddingService generates embeddings using OpenAI API.
type EmbeddingService struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	model      string
	dimensions int
}

// embeddingRequest is the OpenAI API request format.
type embeddingRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

// embeddingResponse is the OpenAI API response format.
type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NewEmbeddingService creates a new OpenAI embedding service.
func NewEmbeddingService(cfg Config) (*EmbeddingService, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	// Determine dimensions
	dimensions := cfg.Dimensions
	if dimensions == 0 {
		var ok bool
		dimensions, ok = modelDimensions[cfg.Model]
		if !ok {
			dimensions = 1536 // Default fallback
		}
	}

	return &EmbeddingService{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		dimensions: dimensions,
	}, nil
}

// Embed generates a vector embedding for the given text.
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := s.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("openai: no embedding returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts efficiently.
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := embeddingRequest{
		Model: s.model,
		Input: texts,
	}

	// Only include dimensions for text-embedding-3-* models
	if s.model == "text-embedding-3-small" || s.model == "text-embedding-3-large" {
		if s.dimensions > 0 {
			reqBody.Dimensions = s.dimensions
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/embeddings",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var embedResp embeddingResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if embedResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", embedResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(body))
	}

	// Convert float64 to float32 and order by index
	embeddings := make([][]float32, len(texts))
	for _, data := range embedResp.Data {
		embedding := make([]float32, len(data.Embedding))
		for i, v := range data.Embedding {
			embedding[i] = float32(v)
		}
		embeddings[data.Index] = embedding
	}

	return embeddings, nil
}

// Dimensions returns the embedding vector size.
func (s *EmbeddingService) Dimensions() int {
	return s.dimensions
}

// ModelName returns the name of the embedding model being used.
func (s *EmbeddingService) ModelName() string {
	return s.model
}

// Ping validates the service is reachable by checking the /models endpoint.
// This is a lightweight check that validates the API key without running inference.
func (s *EmbeddingService) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/models", http.NoBody)
	if err != nil {
		return fmt.Errorf("openai: failed to create ping request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("openai: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("openai: API returned status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("openai: API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Close releases resources.
func (s *EmbeddingService) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
