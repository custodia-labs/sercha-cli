// Package ollama provides an embedding service adapter using Ollama.
package ollama

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
	DefaultBaseURL    = "http://localhost:11434"
	DefaultModel      = "nomic-embed-text"
	DefaultTimeout    = 30 * time.Second
	DefaultDimensions = 768 // nomic-embed-text default
)

// Config holds configuration for the Ollama embedding service.
type Config struct {
	// BaseURL is the Ollama API base URL (default: http://localhost:11434).
	BaseURL string

	// Model is the embedding model to use (default: nomic-embed-text).
	Model string

	// Timeout is the request timeout (default: 30s).
	Timeout time.Duration

	// Dimensions is the embedding vector size (model-dependent).
	Dimensions int
}

// EmbeddingService generates embeddings using Ollama.
type EmbeddingService struct {
	client     *http.Client
	baseURL    string
	model      string
	dimensions int
}

// embedRequest is the Ollama API request format.
type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// embedResponse is the Ollama API response format.
type embedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// NewEmbeddingService creates a new Ollama embedding service.
func NewEmbeddingService(cfg Config) *EmbeddingService {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Dimensions == 0 {
		cfg.Dimensions = DefaultDimensions
	}

	return &EmbeddingService{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
	}
}

// Embed generates a vector embedding for the given text.
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := embedRequest{
		Model:  s.model,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/api/embeddings",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ollama error (status %d): failed to read response", resp.StatusCode)
		}
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(embedResp.Embedding))
	for i, v := range embedResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts efficiently.
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Ollama doesn't have a native batch API, so we call Embed for each text.
	// Future optimization: use goroutines for parallelism.
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := s.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
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

// Ping validates the service is reachable by checking the /api/tags endpoint.
// This is a lightweight check that validates connectivity without running inference.
func (s *EmbeddingService) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/tags", http.NoBody)
	if err != nil {
		return fmt.Errorf("ollama: failed to create ping request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("ollama: API returned status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("ollama: API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Close releases resources.
func (s *EmbeddingService) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
