// Package driven provides interfaces for infrastructure adapters (secondary/outbound ports).
package driven

import "context"

// LLMService provides language model operations for query and document understanding.
// This is an optional service - when nil, features degrade gracefully to keyword-only search.
//
// Implementations may include:
//   - OpenAI (GPT-4, GPT-3.5)
//   - Anthropic (Claude)
//   - Ollama (local models)
//   - LM Studio (local inference server)
type LLMService interface {
	// Generate produces text completion from a prompt.
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error)

	// Chat conducts a multi-turn conversation.
	Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (string, error)

	// RewriteQuery expands or rewrites a search query for better recall.
	// This can add synonyms, fix typos, or expand abbreviations.
	RewriteQuery(ctx context.Context, query string) (string, error)

	// Summarise creates a summary of document content.
	Summarise(ctx context.Context, content string, maxLength int) (string, error)

	// ModelName returns the name of the LLM model being used.
	ModelName() string

	// Ping validates the service is reachable by making a lightweight test request.
	// This is used at startup to verify connectivity before committing to a search mode.
	Ping(ctx context.Context) error

	// Close releases resources.
	Close() error
}

// GenerateOptions configures text generation behaviour.
type GenerateOptions struct {
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int

	// Temperature controls randomness (0.0 = deterministic, 1.0 = creative).
	Temperature float64

	// StopWords are sequences that stop generation when encountered.
	StopWords []string
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	// Role is one of "system", "user", or "assistant".
	Role string

	// Content is the message text.
	Content string
}

// ChatOptions configures chat behaviour.
type ChatOptions struct {
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int

	// Temperature controls randomness (0.0 = deterministic, 1.0 = creative).
	Temperature float64
}
