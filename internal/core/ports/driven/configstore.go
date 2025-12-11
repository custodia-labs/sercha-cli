package driven

// ConfigStore provides access to application configuration.
// Implementations handle persistence (e.g., TOML files) and type conversion.
type ConfigStore interface {
	// Get retrieves a configuration value by key.
	// Returns the value and a boolean indicating if the key exists.
	Get(key string) (any, bool)

	// GetString retrieves a string configuration value.
	// Returns empty string if key doesn't exist or isn't a string.
	GetString(key string) string

	// GetInt retrieves an integer configuration value.
	// Returns 0 if key doesn't exist or isn't an integer.
	GetInt(key string) int

	// GetBool retrieves a boolean configuration value.
	// Returns false if key doesn't exist or isn't a boolean.
	GetBool(key string) bool

	// GetStringSlice retrieves a string slice configuration value.
	// Returns nil if key doesn't exist or isn't a slice.
	GetStringSlice(key string) []string

	// Set stores a configuration value.
	// The value is persisted immediately.
	Set(key string, value any) error

	// Save persists the current configuration to storage.
	Save() error

	// Load reads configuration from storage.
	Load() error

	// Path returns the configuration file path.
	Path() string
}
