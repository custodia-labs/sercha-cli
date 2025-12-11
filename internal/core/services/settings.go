package services

import (
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ensure SettingsService implements the interface.
var _ driving.SettingsService = (*SettingsService)(nil)

// Config keys for settings storage.
//
//nolint:gosec // G101: These are config key names, not actual credentials.
const (
	keySearchMode      = "search.mode"
	keyEmbedProvider   = "embedding.provider"
	keyEmbedModel      = "embedding.model"
	keyEmbedBaseURL    = "embedding.base_url"
	keyEmbedAPIKey     = "embedding.api_key"
	keyLLMProvider     = "llm.provider"
	keyLLMModel        = "llm.model"
	keyLLMBaseURL      = "llm.base_url"
	keyLLMAPIKey       = "llm.api_key"
	keyVectorEnabled   = "vector_index.enabled"
	keyVectorDims      = "vector_index.dimensions"
	keyVectorPrecision = "vector_index.precision"
)

// SettingsService manages application settings.
type SettingsService struct {
	configStore driven.ConfigStore
	aiValidator driven.AIConfigValidator
}

// NewSettingsService creates a new settings service.
func NewSettingsService(configStore driven.ConfigStore, aiValidator driven.AIConfigValidator) *SettingsService {
	return &SettingsService{
		configStore: configStore,
		aiValidator: aiValidator,
	}
}

// Get retrieves current application settings.
func (s *SettingsService) Get() (*domain.AppSettings, error) {
	defaults := domain.DefaultAppSettings()

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: s.getSearchMode(defaults.Search.Mode),
		},
		Embedding: domain.EmbeddingSettings{
			Provider: s.getProvider(keyEmbedProvider, defaults.Embedding.Provider),
			Model:    s.getString(keyEmbedModel, defaults.Embedding.Model),
			BaseURL:  s.configStore.GetString(keyEmbedBaseURL), // No default - empty is valid for cloud providers
			APIKey:   s.configStore.GetString(keyEmbedAPIKey),
		},
		LLM: domain.LLMSettings{
			Provider: s.getProvider(keyLLMProvider, defaults.LLM.Provider),
			Model:    s.getString(keyLLMModel, defaults.LLM.Model),
			BaseURL:  s.configStore.GetString(keyLLMBaseURL), // No default - empty is valid for cloud providers
			APIKey:   s.configStore.GetString(keyLLMAPIKey),
		},
		VectorIndex: domain.VectorIndexSettings{
			Enabled:    s.getBool(keyVectorEnabled, defaults.VectorIndex.Enabled),
			Dimensions: s.getInt(keyVectorDims, defaults.VectorIndex.Dimensions),
			Precision:  s.getVectorPrecision(defaults.VectorIndex.Precision),
		},
	}

	return settings, nil
}

// Save persists application settings.
func (s *SettingsService) Save(settings *domain.AppSettings) error {
	// Save search settings
	if err := s.configStore.Set(keySearchMode, settings.Search.Mode.String()); err != nil {
		return fmt.Errorf("save search mode: %w", err)
	}

	// Save embedding settings
	if err := s.configStore.Set(keyEmbedProvider, settings.Embedding.Provider.String()); err != nil {
		return fmt.Errorf("save embedding provider: %w", err)
	}
	if err := s.configStore.Set(keyEmbedModel, settings.Embedding.Model); err != nil {
		return fmt.Errorf("save embedding model: %w", err)
	}
	if err := s.configStore.Set(keyEmbedBaseURL, settings.Embedding.BaseURL); err != nil {
		return fmt.Errorf("save embedding base_url: %w", err)
	}
	if settings.Embedding.APIKey != "" {
		if err := s.configStore.Set(keyEmbedAPIKey, settings.Embedding.APIKey); err != nil {
			return fmt.Errorf("save embedding api_key: %w", err)
		}
	}

	// Save LLM settings
	if err := s.configStore.Set(keyLLMProvider, settings.LLM.Provider.String()); err != nil {
		return fmt.Errorf("save llm provider: %w", err)
	}
	if err := s.configStore.Set(keyLLMModel, settings.LLM.Model); err != nil {
		return fmt.Errorf("save llm model: %w", err)
	}
	if err := s.configStore.Set(keyLLMBaseURL, settings.LLM.BaseURL); err != nil {
		return fmt.Errorf("save llm base_url: %w", err)
	}
	if settings.LLM.APIKey != "" {
		if err := s.configStore.Set(keyLLMAPIKey, settings.LLM.APIKey); err != nil {
			return fmt.Errorf("save llm api_key: %w", err)
		}
	}

	// Save vector index settings
	if err := s.configStore.Set(keyVectorEnabled, settings.VectorIndex.Enabled); err != nil {
		return fmt.Errorf("save vector enabled: %w", err)
	}
	if err := s.configStore.Set(keyVectorDims, settings.VectorIndex.Dimensions); err != nil {
		return fmt.Errorf("save vector dimensions: %w", err)
	}
	if err := s.configStore.Set(keyVectorPrecision, settings.VectorIndex.Precision.String()); err != nil {
		return fmt.Errorf("save vector precision: %w", err)
	}

	return nil
}

// SetSearchMode updates the search mode.
func (s *SettingsService) SetSearchMode(mode domain.SearchMode) error {
	if !mode.IsValid() {
		return fmt.Errorf("invalid search mode: %s", mode)
	}

	settings, err := s.Get()
	if err != nil {
		return err
	}

	settings.Search.Mode = mode

	// Auto-enable vector index if semantic search is needed
	if mode.RequiresEmbedding() {
		settings.VectorIndex.Enabled = true
	}

	return s.Save(settings)
}

// SetEmbeddingProvider configures the embedding provider.
func (s *SettingsService) SetEmbeddingProvider(provider domain.AIProvider, model, apiKey string) error {
	if !provider.IsValid() {
		return fmt.Errorf("invalid embedding provider: %s", provider)
	}

	// Validate provider supports embeddings
	validProviders := domain.AllEmbeddingProviders()
	valid := false
	for _, p := range validProviders {
		if p == provider {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("provider %s does not support embeddings", provider)
	}

	// Validate API key if required
	if provider.RequiresAPIKey() && apiKey == "" {
		return fmt.Errorf("API key required for %s", provider)
	}

	settings, err := s.Get()
	if err != nil {
		return err
	}

	settings.Embedding.Provider = provider

	// Set model - use provided or default
	if model != "" {
		settings.Embedding.Model = model
	} else {
		defaults := domain.DefaultEmbeddingModels()
		if defaultModel, ok := defaults[provider]; ok {
			settings.Embedding.Model = defaultModel
		}
	}

	// Set base URL based on provider type
	if provider.IsLocal() {
		// Local providers need a base URL
		if settings.Embedding.BaseURL == "" {
			settings.Embedding.BaseURL = "http://localhost:11434"
		}
	} else {
		// Cloud providers don't need a custom base URL
		settings.Embedding.BaseURL = ""
	}

	// Set API key
	settings.Embedding.APIKey = apiKey

	// Update vector dimensions based on model
	dims := domain.EmbeddingDimensions()
	if d, ok := dims[settings.Embedding.Model]; ok {
		settings.VectorIndex.Dimensions = d
	}

	return s.Save(settings)
}

// SetLLMProvider configures the LLM provider.
func (s *SettingsService) SetLLMProvider(provider domain.AIProvider, model, apiKey string) error {
	if !provider.IsValid() {
		return fmt.Errorf("invalid LLM provider: %s", provider)
	}

	// Validate API key if required
	if provider.RequiresAPIKey() && apiKey == "" {
		return fmt.Errorf("API key required for %s", provider)
	}

	settings, err := s.Get()
	if err != nil {
		return err
	}

	settings.LLM.Provider = provider

	// Set model - use provided or default
	if model != "" {
		settings.LLM.Model = model
	} else {
		defaults := domain.DefaultLLMModels()
		if defaultModel, ok := defaults[provider]; ok {
			settings.LLM.Model = defaultModel
		}
	}

	// Set base URL based on provider type
	if provider.IsLocal() {
		// Local providers need a base URL
		if settings.LLM.BaseURL == "" {
			settings.LLM.BaseURL = "http://localhost:11434"
		}
	} else {
		// Cloud providers don't need a custom base URL
		settings.LLM.BaseURL = ""
	}

	// Set API key
	settings.LLM.APIKey = apiKey

	return s.Save(settings)
}

// Validate checks if current settings are valid for the configured mode.
func (s *SettingsService) Validate() error {
	settings, err := s.Get()
	if err != nil {
		return err
	}

	// Validate search mode
	if !settings.Search.Mode.IsValid() {
		return fmt.Errorf("invalid search mode: %s", settings.Search.Mode)
	}

	// Check embedding configuration if required
	if settings.Search.Mode.RequiresEmbedding() {
		if !settings.Embedding.IsConfigured() {
			return fmt.Errorf(
				"search mode %q requires embedding provider to be configured",
				settings.Search.Mode.Description(),
			)
		}
	}

	// Check LLM configuration if required
	if settings.Search.Mode.RequiresLLM() {
		if !settings.LLM.IsConfigured() {
			return fmt.Errorf(
				"search mode %q requires LLM provider to be configured",
				settings.Search.Mode.Description(),
			)
		}
	}

	return nil
}

// RequiresEmbedding returns true if current mode needs embedding.
func (s *SettingsService) RequiresEmbedding() bool {
	settings, err := s.Get()
	if err != nil {
		return false
	}
	return settings.Search.Mode.RequiresEmbedding()
}

// RequiresLLM returns true if current mode needs LLM.
func (s *SettingsService) RequiresLLM() bool {
	settings, err := s.Get()
	if err != nil {
		return false
	}
	return settings.Search.Mode.RequiresLLM()
}

// GetDefaults returns default settings.
func (s *SettingsService) GetDefaults() domain.AppSettings {
	return domain.DefaultAppSettings()
}

// ValidateEmbeddingConfig validates the current embedding configuration by pinging the provider.
func (s *SettingsService) ValidateEmbeddingConfig() error {
	if s.aiValidator == nil {
		return nil
	}
	settings, err := s.Get()
	if err != nil {
		return err
	}
	return s.aiValidator.ValidateEmbedding(&settings.Embedding)
}

// ValidateLLMConfig validates the current LLM configuration by pinging the provider.
func (s *SettingsService) ValidateLLMConfig() error {
	if s.aiValidator == nil {
		return nil
	}
	settings, err := s.Get()
	if err != nil {
		return err
	}
	return s.aiValidator.ValidateLLM(&settings.LLM)
}

// Helper methods for reading config with defaults.

func (s *SettingsService) getString(key, defaultVal string) string {
	val := s.configStore.GetString(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func (s *SettingsService) getInt(key string, defaultVal int) int {
	val := s.configStore.GetInt(key)
	if val == 0 {
		return defaultVal
	}
	return val
}

func (s *SettingsService) getBool(key string, defaultVal bool) bool {
	if _, exists := s.configStore.Get(key); !exists {
		return defaultVal
	}
	return s.configStore.GetBool(key)
}

func (s *SettingsService) getSearchMode(defaultVal domain.SearchMode) domain.SearchMode {
	val := s.configStore.GetString(keySearchMode)
	if val == "" {
		return defaultVal
	}
	mode := domain.SearchMode(val)
	if !mode.IsValid() {
		return defaultVal
	}
	return mode
}

func (s *SettingsService) getProvider(key string, defaultVal domain.AIProvider) domain.AIProvider {
	val := s.configStore.GetString(key)
	if val == "" {
		return defaultVal
	}
	provider := domain.AIProvider(val)
	if !provider.IsValid() {
		return defaultVal
	}
	return provider
}

func (s *SettingsService) getVectorPrecision(defaultVal domain.VectorPrecision) domain.VectorPrecision {
	val := s.configStore.GetString(keyVectorPrecision)
	if val == "" {
		return defaultVal
	}
	precision := domain.VectorPrecision(val)
	if !precision.IsValid() {
		return defaultVal
	}
	return precision
}

// GetPipelineConfig returns the post-processor pipeline configuration.
// Returns default configuration if nothing is configured.
func (s *SettingsService) GetPipelineConfig() domain.PipelineConfig {
	defaults := domain.DefaultPipelineConfig()

	// Try to load processors list from config
	if processors := s.configStore.GetStringSlice("pipeline.processors"); len(processors) > 0 {
		defaults.Processors = processors
	}

	// Load per-processor configs
	// For each known processor, check if config exists
	for _, name := range defaults.Processors {
		prefix := "pipeline." + name + "."
		cfg := s.loadProcessorConfig(prefix)
		if len(cfg) > 0 {
			if defaults.ProcessorConfigs == nil {
				defaults.ProcessorConfigs = make(map[string]map[string]any)
			}
			// Merge with existing defaults
			existing := defaults.ProcessorConfigs[name]
			if existing == nil {
				existing = make(map[string]any)
			}
			for k, v := range cfg {
				existing[k] = v
			}
			defaults.ProcessorConfigs[name] = existing
		}
	}

	return defaults
}

// loadProcessorConfig loads config keys with a given prefix into a map.
func (s *SettingsService) loadProcessorConfig(prefix string) map[string]any {
	cfg := make(map[string]any)

	// Check common processor config keys
	knownKeys := []string{"chunk_size", "overlap", "max_length", "model"}
	for _, key := range knownKeys {
		fullKey := prefix + key
		if val, exists := s.configStore.Get(fullKey); exists {
			cfg[key] = val
		}
	}

	return cfg
}

// GetSchedulerConfig returns the scheduler configuration.
// Returns default configuration if nothing is configured.
func (s *SettingsService) GetSchedulerConfig() domain.SchedulerConfig {
	defaults := domain.DefaultSchedulerConfig()

	// Master switch
	if _, exists := s.configStore.Get("scheduler.enabled"); exists {
		defaults.Enabled = s.configStore.GetBool("scheduler.enabled")
	}

	// Per-task config
	// Map from task ID to config key (underscore version for TOML)
	taskKeys := map[string]string{
		domain.TaskIDOAuthRefresh: "oauth_refresh",
		domain.TaskIDDocumentSync: "document_sync",
	}

	for taskID, configKey := range taskKeys {
		prefix := "scheduler." + configKey + "."

		taskCfg := defaults.TaskConfigs[taskID]

		// Check enabled
		if _, exists := s.configStore.Get(prefix + "enabled"); exists {
			taskCfg.Enabled = s.configStore.GetBool(prefix + "enabled")
		}

		// Check interval (duration string like "45m", "1h")
		if interval := s.configStore.GetString(prefix + "interval"); interval != "" {
			if d, err := s.parseDuration(interval); err == nil {
				taskCfg.Interval = d
			}
		}

		defaults.TaskConfigs[taskID] = taskCfg
	}

	return defaults
}

// parseDuration parses a duration string.
func (s *SettingsService) parseDuration(str string) (time.Duration, error) {
	return time.ParseDuration(str)
}
