package driven

// PromptStore provides access to LLM prompt templates.
// Implementations may load prompts from files, embed them in the binary,
// or fetch them from a remote configuration service.
type PromptStore interface {
	// Load returns the prompt template for the given name.
	// Returns the prompt content and any error encountered.
	// If the prompt is not found, implementations should return a sensible default
	// or an error, depending on whether the prompt is required.
	Load(name string) (string, error)

	// Reload clears any cached prompts, forcing fresh loads on next access.
	// This is useful when prompts may have been edited on disk.
	Reload()
}

// Well-known prompt names used throughout the application.
// These constants define the contract between prompt consumers and providers.
const (
	// PromptQueryRewrite expands search queries for better recall.
	// The prompt template expects a %s placeholder for the original query.
	PromptQueryRewrite = "query_rewrite"

	// PromptSummarise creates summaries of document content.
	// The prompt template expects %d (max length) and %s (content) placeholders.
	PromptSummarise = "summarise"

	// PromptChatSystem is the system prompt for conversational search mode.
	// This prompt has no format placeholders.
	PromptChatSystem = "chat_system"
)

// PromptStoreAware is an optional interface for services that can use custom prompts.
// Services implementing this interface can have their prompt templates customised
// by injecting a PromptStore after construction.
type PromptStoreAware interface {
	// SetPromptStore sets the prompt store for loading customisable prompts.
	// If not set, the service should use hardcoded default prompts.
	SetPromptStore(store PromptStore)
}
