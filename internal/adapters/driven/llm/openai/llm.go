// Package openai provides an LLM service adapter using OpenAI API.
package openai

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
	DefaultBaseURL    = "https://api.openai.com/v1"
	DefaultLLMModel   = "gpt-4o-mini"
	DefaultLLMTimeout = 120 * time.Second
)

// LLMConfig holds configuration for the OpenAI LLM service.
type LLMConfig struct {
	// APIKey is the OpenAI API key (required).
	APIKey string

	// BaseURL is the API base URL (default: https://api.openai.com/v1).
	// Can be changed for Azure OpenAI or compatible APIs.
	BaseURL string

	// Model is the LLM model to use (default: gpt-4o-mini).
	Model string

	// Timeout is the request timeout (default: 120s).
	Timeout time.Duration
}

// LLMService provides LLM operations using OpenAI API.
type LLMService struct {
	client      *http.Client
	baseURL     string
	apiKey      string
	model       string
	promptStore driven.PromptStore
}

// chatCompletionRequest is the OpenAI /chat/completions request format.
type chatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []chatCompletionMsg `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
}

// chatCompletionMsg is the OpenAI chat message format.
type chatCompletionMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse is the OpenAI /chat/completions response format.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NewLLMService creates a new OpenAI LLM service.
func NewLLMService(cfg LLMConfig) (*LLMService, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: API key is required")
	}
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
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
	}, nil
}

// Generate produces text completion from a prompt.
func (s *LLMService) Generate(ctx context.Context, prompt string, opts driven.GenerateOptions) (string, error) {
	messages := []driven.ChatMessage{
		{Role: "user", Content: prompt},
	}
	chatOpts := driven.ChatOptions{
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
	}
	return s.chatCompletion(ctx, messages, chatOpts, opts.StopWords)
}

// Chat conducts a multi-turn conversation.
func (s *LLMService) Chat(ctx context.Context, messages []driven.ChatMessage, opts driven.ChatOptions) (string, error) {
	return s.chatCompletion(ctx, messages, opts, nil)
}

// chatCompletion is the internal implementation for both Generate and Chat.
func (s *LLMService) chatCompletion(
	ctx context.Context,
	messages []driven.ChatMessage,
	opts driven.ChatOptions,
	stopWords []string,
) (string, error) {
	// Convert driven.ChatMessage to internal format
	chatMessages := make([]chatCompletionMsg, len(messages))
	for i, msg := range messages {
		chatMessages[i] = chatCompletionMsg{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	reqBody := chatCompletionRequest{
		Model:    s.model,
		Messages: chatMessages,
	}

	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = opts.MaxTokens
	}
	if opts.Temperature > 0 {
		reqBody.Temperature = opts.Temperature
	}
	if len(stopWords) > 0 {
		reqBody.Stop = stopWords
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/chat/completions",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("openai error: %s", chatResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(body))
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openai: no response choices returned")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// defaultQueryRewritePrompt is the fallback prompt when no PromptStore is configured.
const defaultQueryRewritePrompt = `Rewrite this search query to improve recall. Add synonyms and fix typos.
Return ONLY the rewritten query, nothing else.

Original: %s
Rewritten:`

// defaultSummarisePrompt is the fallback prompt when no PromptStore is configured.
const defaultSummarisePrompt = `Summarise the following content in %d characters or less.
Be concise and capture the key points.

Content:
%s

Summary:`

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

// Ping validates the service is reachable by checking the /models endpoint.
// This is a lightweight check that validates the API key without running inference.
func (s *LLMService) Ping(ctx context.Context) error {
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
func (s *LLMService) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
