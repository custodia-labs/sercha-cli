package memory

import (
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure ConfigStore implements the interface.
var _ driven.ConfigStore = (*ConfigStore)(nil)

// ConfigStore is an in-memory implementation of driven.ConfigStore for testing.
type ConfigStore struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewConfigStore creates a new in-memory config store.
func NewConfigStore() *ConfigStore {
	return &ConfigStore{
		values: make(map[string]any),
	}
}

// Get retrieves a configuration value by key.
func (s *ConfigStore) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.values[key]
	return val, ok
}

// GetString retrieves a string configuration value.
func (s *ConfigStore) GetString(key string) string {
	val, ok := s.Get(key)
	if !ok {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetInt retrieves an integer configuration value.
func (s *ConfigStore) GetInt(key string) int {
	val, ok := s.Get(key)
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetBool retrieves a boolean configuration value.
func (s *ConfigStore) GetBool(key string) bool {
	val, ok := s.Get(key)
	if !ok {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// GetStringSlice retrieves a string slice configuration value.
func (s *ConfigStore) GetStringSlice(key string) []string {
	val, ok := s.Get(key)
	if !ok {
		return nil
	}
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// Set stores a configuration value.
func (s *ConfigStore) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

// Save persists the current configuration (no-op for memory store).
func (s *ConfigStore) Save() error {
	return nil
}

// Load reads configuration from storage (no-op for memory store).
func (s *ConfigStore) Load() error {
	return nil
}

// Path returns the configuration file path.
func (s *ConfigStore) Path() string {
	return ":memory:"
}
