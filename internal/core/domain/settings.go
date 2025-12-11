package domain

const unknownDescription = "Unknown"

// SearchMode defines how search operations combine different retrieval methods.
type SearchMode string

// Available search modes.
const (
	// SearchModeTextOnly uses only keyword/full-text search.
	SearchModeTextOnly SearchMode = "text_only"

	// SearchModeHybrid combines text and semantic (vector) search.
	SearchModeHybrid SearchMode = "hybrid"

	// SearchModeLLMAssisted uses text search with LLM query expansion.
	SearchModeLLMAssisted SearchMode = "llm_assisted"

	// SearchModeFull combines text, semantic, and LLM query expansion.
	SearchModeFull SearchMode = "full"
)

// IsValid returns true if the search mode is recognised.
func (m SearchMode) IsValid() bool {
	switch m {
	case SearchModeTextOnly, SearchModeHybrid, SearchModeLLMAssisted, SearchModeFull:
		return true
	default:
		return false
	}
}

// RequiresEmbedding returns true if this mode needs an embedding provider.
func (m SearchMode) RequiresEmbedding() bool {
	return m == SearchModeHybrid || m == SearchModeFull
}

// RequiresLLM returns true if this mode needs an LLM provider.
func (m SearchMode) RequiresLLM() bool {
	return m == SearchModeLLMAssisted || m == SearchModeFull
}

// String returns the string representation.
func (m SearchMode) String() string {
	return string(m)
}

// Description returns a human-readable description of the mode.
func (m SearchMode) Description() string {
	switch m {
	case SearchModeTextOnly:
		return "Text Only (keyword search)"
	case SearchModeHybrid:
		return "Hybrid (text + semantic search)"
	case SearchModeLLMAssisted:
		return "LLM Assisted (text + query expansion)"
	case SearchModeFull:
		return "Full (text + semantic + LLM)"
	default:
		return unknownDescription
	}
}

// AIProvider identifies an AI service provider for embeddings or LLM.
type AIProvider string

// Available AI providers.
const (
	// AIProviderOllama is local Ollama instance.
	AIProviderOllama AIProvider = "ollama"

	// AIProviderOpenAI is OpenAI cloud API.
	AIProviderOpenAI AIProvider = "openai"

	// AIProviderAnthropic is Anthropic cloud API.
	AIProviderAnthropic AIProvider = "anthropic"
)

// IsValid returns true if the AI provider is recognised.
func (p AIProvider) IsValid() bool {
	switch p {
	case AIProviderOllama, AIProviderOpenAI, AIProviderAnthropic:
		return true
	default:
		return false
	}
}

// RequiresAPIKey returns true if this provider needs an API key.
func (p AIProvider) RequiresAPIKey() bool {
	return p == AIProviderOpenAI || p == AIProviderAnthropic
}

// IsLocal returns true if this provider runs locally.
func (p AIProvider) IsLocal() bool {
	return p == AIProviderOllama
}

// String returns the string representation.
func (p AIProvider) String() string {
	return string(p)
}

// Description returns a human-readable description of the provider.
func (p AIProvider) Description() string {
	switch p {
	case AIProviderOllama:
		return "Ollama (local)"
	case AIProviderOpenAI:
		return "OpenAI (cloud)"
	case AIProviderAnthropic:
		return "Anthropic (cloud)"
	default:
		return unknownDescription
	}
}

// SearchSettings holds search behaviour configuration.
type SearchSettings struct {
	// Mode is the search retrieval mode.
	Mode SearchMode
}

// EmbeddingSettings holds embedding provider configuration.
type EmbeddingSettings struct {
	// Provider is the embedding service provider.
	Provider AIProvider

	// Model is the embedding model name.
	Model string

	// BaseURL is the API endpoint (for Ollama).
	BaseURL string

	// APIKey is the API key (for OpenAI).
	APIKey string
}

// IsConfigured returns true if the embedding provider is set up.
func (e EmbeddingSettings) IsConfigured() bool {
	if !e.Provider.IsValid() {
		return false
	}
	if e.Provider.RequiresAPIKey() && e.APIKey == "" {
		return false
	}
	return true
}

// LLMSettings holds LLM provider configuration.
type LLMSettings struct {
	// Provider is the LLM service provider.
	Provider AIProvider

	// Model is the LLM model name.
	Model string

	// BaseURL is the API endpoint (for Ollama).
	BaseURL string

	// APIKey is the API key (for OpenAI/Anthropic).
	APIKey string
}

// IsConfigured returns true if the LLM provider is set up.
func (l LLMSettings) IsConfigured() bool {
	if !l.Provider.IsValid() {
		return false
	}
	if l.Provider.RequiresAPIKey() && l.APIKey == "" {
		return false
	}
	return true
}

// VectorPrecision defines the storage precision for vector embeddings.
type VectorPrecision string

// Available vector precision options.
const (
	// VectorPrecisionFloat32 stores vectors at full 32-bit precision (no compression).
	VectorPrecisionFloat32 VectorPrecision = "float32"

	// VectorPrecisionFloat16 stores vectors at 16-bit half precision (50% storage savings).
	VectorPrecisionFloat16 VectorPrecision = "float16"

	// VectorPrecisionInt8 stores vectors at 8-bit integer precision (75% storage savings).
	VectorPrecisionInt8 VectorPrecision = "int8"
)

// IsValid returns true if the precision is recognised.
func (p VectorPrecision) IsValid() bool {
	switch p {
	case VectorPrecisionFloat32, VectorPrecisionFloat16, VectorPrecisionInt8:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (p VectorPrecision) String() string {
	return string(p)
}

// Description returns a human-readable description of the precision.
func (p VectorPrecision) Description() string {
	switch p {
	case VectorPrecisionFloat32:
		return "Float32 (full precision, no compression)"
	case VectorPrecisionFloat16:
		return "Float16 (half precision, 50% savings)"
	case VectorPrecisionInt8:
		return "Int8 (8-bit quantized, 75% savings)"
	default:
		return unknownDescription
	}
}

// VectorIndexSettings holds vector index configuration.
type VectorIndexSettings struct {
	// Enabled indicates whether vector indexing is active.
	Enabled bool

	// Dimensions is the embedding vector size.
	Dimensions int

	// Precision is the storage precision for vectors.
	// Default is float16 (best balance of size vs quality).
	Precision VectorPrecision
}

// AppSettings holds all application settings.
type AppSettings struct {
	// Search holds search behaviour settings.
	Search SearchSettings

	// Embedding holds embedding provider settings.
	Embedding EmbeddingSettings

	// LLM holds LLM provider settings.
	LLM LLMSettings

	// VectorIndex holds vector index settings.
	VectorIndex VectorIndexSettings
}

// DefaultAppSettings returns settings with sensible defaults.
// AI features (Embedding, LLM) are left unconfigured by default.
// Users must explicitly configure them via settings wizard.
func DefaultAppSettings() AppSettings {
	return AppSettings{
		Search: SearchSettings{
			Mode: SearchModeTextOnly,
		},
		// Embedding is left unconfigured - user must set up via settings wizard
		Embedding: EmbeddingSettings{},
		// LLM is left unconfigured - user must set up via settings wizard
		LLM: LLMSettings{},
		VectorIndex: VectorIndexSettings{
			Enabled:    false,
			Dimensions: 768,                    // nomic-embed-text default
			Precision:  VectorPrecisionFloat16, // Best balance of size vs quality
		},
	}
}

// AllSearchModes returns all available search modes.
func AllSearchModes() []SearchMode {
	return []SearchMode{
		SearchModeTextOnly,
		SearchModeHybrid,
		SearchModeLLMAssisted,
		SearchModeFull,
	}
}

// AllEmbeddingProviders returns providers that support embeddings.
func AllEmbeddingProviders() []AIProvider {
	return []AIProvider{
		AIProviderOllama,
		AIProviderOpenAI,
	}
}

// AllLLMProviders returns providers that support LLM operations.
func AllLLMProviders() []AIProvider {
	return []AIProvider{
		AIProviderOllama,
		AIProviderOpenAI,
		AIProviderAnthropic,
	}
}

// DefaultEmbeddingModels returns default models for each embedding provider.
func DefaultEmbeddingModels() map[AIProvider]string {
	return map[AIProvider]string{
		AIProviderOllama: "nomic-embed-text",
		AIProviderOpenAI: "text-embedding-3-small",
	}
}

// DefaultLLMModels returns default models for each LLM provider.
func DefaultLLMModels() map[AIProvider]string {
	return map[AIProvider]string{
		AIProviderOllama:    "llama3.2",
		AIProviderOpenAI:    "gpt-4o-mini",
		AIProviderAnthropic: "claude-3-5-sonnet-latest",
	}
}

// EmbeddingDimensions returns the vector dimensions for known models.
func EmbeddingDimensions() map[string]int {
	return map[string]int{
		// Ollama models
		"nomic-embed-text":  768,
		"mxbai-embed-large": 1024,
		"all-minilm":        384,
		// OpenAI models
		"text-embedding-3-small": 1536,
		"text-embedding-3-large": 3072,
		"text-embedding-ada-002": 1536,
	}
}

// AllVectorPrecisions returns all available vector precision options.
func AllVectorPrecisions() []VectorPrecision {
	return []VectorPrecision{
		VectorPrecisionFloat32,
		VectorPrecisionFloat16,
		VectorPrecisionInt8,
	}
}

// PipelineConfig holds post-processor pipeline configuration.
// Uses generic map-based config for extensibility - new processors can be added
// without modifying this struct.
type PipelineConfig struct {
	// Processors is the ordered list of processor names to run.
	Processors []string

	// ProcessorConfigs holds per-processor configuration as generic maps.
	// Key is processor name, value is processor-specific config.
	ProcessorConfigs map[string]map[string]any
}

// GetProcessorConfig returns config for a specific processor, or nil if not set.
func (c *PipelineConfig) GetProcessorConfig(name string) map[string]any {
	if c.ProcessorConfigs == nil {
		return nil
	}
	return c.ProcessorConfigs[name]
}

// DefaultPipelineConfig returns the default pipeline configuration.
// Works out-of-the-box with chunker using sensible defaults.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Processors: []string{"chunker"},
		ProcessorConfigs: map[string]map[string]any{
			"chunker": {
				"chunk_size": 1000,
				"overlap":    200,
			},
		},
	}
}
