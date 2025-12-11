package file

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure ConfigStore implements the interface.
var _ driven.ConfigStore = (*ConfigStore)(nil)

// ConfigStore is a file-based implementation of driven.ConfigStore using TOML.
// Configuration is stored in a TOML file within the sercha config directory.
type ConfigStore struct {
	mu       sync.RWMutex
	filePath string
	data     map[string]any
}

// NewConfigStore creates a new TOML-based config store.
// If configDir is empty, defaults to ~/.sercha/config.toml.
func NewConfigStore(configDir string) (*ConfigStore, error) {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configDir = filepath.Join(home, ".sercha")
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, err
	}

	s := &ConfigStore{
		filePath: filepath.Join(configDir, "config.toml"),
		data:     make(map[string]any),
	}

	// Load existing data if file exists
	if err := s.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

// Get retrieves a configuration value by key.
func (s *ConfigStore) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	return val, ok
}

// GetString retrieves a string configuration value.
func (s *ConfigStore) GetString(key string) string {
	val, ok := s.Get(key)
	if !ok {
		return ""
	}

	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

// GetInt retrieves an integer configuration value.
func (s *ConfigStore) GetInt(key string) int {
	val, ok := s.Get(key)
	if !ok {
		return 0
	}

	// TOML integers are parsed as int64
	switch v := val.(type) {
	case int64:
		return int(v)
	case int:
		return v
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

	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b
}

// GetStringSlice retrieves a string slice configuration value.
func (s *ConfigStore) GetStringSlice(key string) []string {
	val, ok := s.Get(key)
	if !ok {
		return nil
	}

	// TOML arrays are parsed as []any
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

// Set stores a configuration value and persists immediately.
func (s *ConfigStore) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	return s.save()
}

// Save persists the current configuration to disk.
func (s *ConfigStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

// save writes configuration to the TOML file (caller must hold lock).
func (s *ConfigStore) save() error {
	data, err := toml.Marshal(s.data)
	if err != nil {
		return err
	}

	// Write with restricted permissions
	return os.WriteFile(s.filePath, data, 0600)
}

// Load reads configuration from the TOML file.
func (s *ConfigStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file yet - that's fine, start empty
			s.data = make(map[string]any)
			return nil
		}
		return err
	}

	var loaded map[string]any
	if err := toml.Unmarshal(data, &loaded); err != nil {
		return err
	}

	if loaded == nil {
		loaded = make(map[string]any)
	}

	// Flatten nested maps into dot-notation keys for easier access
	s.data = flattenMap(loaded, "")
	return nil
}

// FlattenMap converts nested maps to dot-notation keys.
// E.g., {"a": {"b": 1}} becomes {"a.b": 1}.
func flattenMap(m map[string]any, prefix string) map[string]any {
	result := make(map[string]any)

	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nested, ok := value.(map[string]any); ok {
			// Recursively flatten nested maps
			for k, v := range flattenMap(nested, fullKey) {
				result[k] = v
			}
		} else {
			result[fullKey] = value
		}
	}

	return result
}

// Path returns the configuration file path.
func (s *ConfigStore) Path() string {
	return s.filePath
}
