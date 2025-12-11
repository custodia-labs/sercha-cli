// Package ollama provides an LLM service adapter using Ollama.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure LLMService implements the interface.
var _ driven.LLMService = (*LLMService)(nil)

// Default configuration values.
const (
	DefaultBaseURL    = "http://localhost:11434"
	DefaultLLMModel   = "llama3.2"
	DefaultLLMTimeout = 120 * time.Second
)

// LLMConfig holds configuration for the Ollama LLM service.
type LLMConfig struct {
	// BaseURL is the Ollama API base URL (default: http://localhost:11434).
	BaseURL string

	// Model is the LLM model to use (default: llama3.2).
	Model string

	// Timeout is the request timeout (default: 120s).
	Timeout time.Duration
}

// LLMService provides LLM operations using Ollama.
type LLMService struct {
	client      *http.Client
	baseURL     string
	model       string
	promptStore driven.PromptStore
}

// generateRequest is the Ollama /api/generate request format.
type generateRequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	Stream  bool     `json:"stream"`
	Options *options `json:"options,omitempty"`
}

// options holds generation parameters.
type options struct {
	NumPredict  int      `json:"num_predict,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// generateResponse is the Ollama /api/generate response format.
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// chatRequest is the Ollama /api/chat request format.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  *options      `json:"options,omitempty"`
}

// chatMessage is the Ollama chat message format.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the Ollama /api/chat response format.
type chatResponse struct {
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// NewLLMService creates a new Ollama LLM service.
func NewLLMService(cfg LLMConfig) *LLMService {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultLLMModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultLLMTimeout
	}

	return &LLMService{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
	}
}

// Generate produces text completion from a prompt.
func (s *LLMService) Generate(ctx context.Context, prompt string, opts driven.GenerateOptions) (string, error) {
	reqBody := generateRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: false,
	}

	if opts.MaxTokens > 0 || opts.Temperature > 0 || len(opts.StopWords) > 0 {
		reqBody.Options = &options{
			NumPredict:  opts.MaxTokens,
			Temperature: opts.Temperature,
			Stop:        opts.StopWords,
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/api/generate",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("ollama error (status %d): failed to read response", resp.StatusCode)
		}
		return "", fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return genResp.Response, nil
}

// Chat conducts a multi-turn conversation.
func (s *LLMService) Chat(ctx context.Context, messages []driven.ChatMessage, opts driven.ChatOptions) (string, error) {
	// Convert driven.ChatMessage to internal format
	chatMessages := make([]chatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = chatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	reqBody := chatRequest{
		Model:    s.model,
		Messages: chatMessages,
		Stream:   false,
	}

	if opts.MaxTokens > 0 || opts.Temperature > 0 {
		reqBody.Options = &options{
			NumPredict:  opts.MaxTokens,
			Temperature: opts.Temperature,
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/api/chat",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("ollama error (status %d): failed to read response", resp.StatusCode)
		}
		return "", fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// defaultQueryRewritePrompt is the fallback prompt when no PromptStore is configured.
const defaultQueryRewritePrompt = `Rewrite this search query to improve recall. Add synonyms and fix typos.
Return ONLY the rewritten query, nothing else.

Original: %s
Rewritten:`

// RewriteQuery expands or rewrites a search query for better recall.
func (s *LLMService) RewriteQuery(ctx context.Context, query string) (string, error) {
	promptTemplate := s.loadPrompt(driven.PromptQueryRewrite, defaultQueryRewritePrompt)
	prompt := fmt.Sprintf(promptTemplate, query)

	result, err := s.Generate(ctx, prompt, driven.GenerateOptions{
		MaxTokens:   100,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("rewrite query: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// defaultSummarisePrompt is the fallback prompt when no PromptStore is configured.
const defaultSummarisePrompt = `Summarise the following content in %d characters or less.
Be concise and capture the key points.

Content:
%s

Summary:`

// Summarise creates a summary of document content.
func (s *LLMService) Summarise(ctx context.Context, content string, maxLength int) (string, error) {
	promptTemplate := s.loadPrompt(driven.PromptSummarise, defaultSummarisePrompt)
	prompt := fmt.Sprintf(promptTemplate, maxLength, content)

	result, err := s.Generate(ctx, prompt, driven.GenerateOptions{
		MaxTokens:   maxLength / 4, // Rough estimate: 4 chars per token
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("summarise: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// loadPrompt loads a prompt from the store, falling back to the default if unavailable.
func (s *LLMService) loadPrompt(name, fallback string) string {
	if s.promptStore == nil {
		return fallback
	}
	prompt, err := s.promptStore.Load(name)
	if err != nil {
		return fallback
	}
	return prompt
}

// ModelName returns the name of the LLM model being used.
func (s *LLMService) ModelName() string {
	return s.model
}

// SetPromptStore sets the prompt store for loading customisable prompts.
// If not set, the service uses hardcoded default prompts.
func (s *LLMService) SetPromptStore(store driven.PromptStore) {
	s.promptStore = store
}

// Ping validates the service is reachable by checking the /api/tags endpoint.
// This is a lightweight check that validates connectivity without running inference.
func (s *LLMService) Ping(ctx context.Context) error {
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
func (s *LLMService) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
