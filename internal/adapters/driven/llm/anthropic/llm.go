// Package anthropic provides an LLM service adapter using Anthropic API.
package anthropic

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
	DefaultBaseURL = "https://api.anthropic.com"
	DefaultModel   = "claude-3-5-sonnet-latest"
	DefaultTimeout = 120 * time.Second

	// AnthropicVersion is the required API version header.
	anthropicVersion = "2023-06-01"
)

// Config holds configuration for the Anthropic LLM service.
type Config struct {
	// APIKey is the Anthropic API key (required).
	APIKey string

	// BaseURL is the API base URL (default: https://api.anthropic.com).
	BaseURL string

	// Model is the LLM model to use (default: claude-3-5-sonnet-latest).
	Model string

	// Timeout is the request timeout (default: 120s).
	Timeout time.Duration
}

// LLMService provides LLM operations using Anthropic API.
type LLMService struct {
	client      *http.Client
	baseURL     string
	apiKey      string
	model       string
	promptStore driven.PromptStore
}

// messagesRequest is the Anthropic /v1/messages request format.
type messagesRequest struct {
	Model       string            `json:"model"`
	Messages    []messagesMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens"`
	System      string            `json:"system,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	StopSeqs    []string          `json:"stop_sequences,omitempty"`
}

// messagesMessage is the Anthropic message format.
type messagesMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesResponse is the Anthropic /v1/messages response format.
type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewLLMService creates a new Anthropic LLM service.
func NewLLMService(cfg Config) (*LLMService, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic: API key is required")
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
	return s.sendMessages(ctx, "", messages, chatOpts, opts.StopWords)
}

// Chat conducts a multi-turn conversation.
func (s *LLMService) Chat(ctx context.Context, messages []driven.ChatMessage, opts driven.ChatOptions) (string, error) {
	// Extract system message if present
	var systemPrompt string
	var chatMessages []driven.ChatMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			chatMessages = append(chatMessages, msg)
		}
	}

	return s.sendMessages(ctx, systemPrompt, chatMessages, opts, nil)
}

// sendMessages is the internal implementation for both Generate and Chat.
func (s *LLMService) sendMessages(
	ctx context.Context,
	systemPrompt string,
	messages []driven.ChatMessage,
	opts driven.ChatOptions,
	stopWords []string,
) (string, error) {
	// Convert driven.ChatMessage to internal format
	apiMessages := make([]messagesMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = messagesMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Anthropic requires max_tokens to be set
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024 // Default
	}

	reqBody := messagesRequest{
		Model:     s.model,
		Messages:  apiMessages,
		MaxTokens: maxTokens,
		System:    systemPrompt,
	}

	if opts.Temperature > 0 {
		reqBody.Temperature = opts.Temperature
	}
	if len(stopWords) > 0 {
		reqBody.StopSeqs = stopWords
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/v1/messages",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var msgResp messagesResponse
	if err := json.Unmarshal(body, &msgResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if msgResp.Error != nil {
		return "", fmt.Errorf("anthropic error: %s", msgResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(body))
	}

	if len(msgResp.Content) == 0 {
		return "", fmt.Errorf("anthropic: no response content returned")
	}

	// Concatenate all text content blocks
	var result strings.Builder
	for _, block := range msgResp.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}

	return result.String(), nil
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

// Ping validates the service is reachable by checking the /v1/models endpoint.
// This is a lightweight check that validates the API key without running inference.
func (s *LLMService) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/v1/models", http.NoBody)
	if err != nil {
		return fmt.Errorf("anthropic: failed to create ping request: %w", err)
	}
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("anthropic: API returned status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("anthropic: API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Close releases resources.
func (s *LLMService) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
