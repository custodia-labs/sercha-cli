package postprocessors

import (
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// BuilderFunc creates a PostProcessor from generic config.
// Config is a map of processor-specific settings parsed from user config.
type BuilderFunc func(cfg map[string]any) (driven.PostProcessor, error)

// Registry maps processor names to their builders.
// It allows dynamic construction of processors from configuration.
type Registry struct {
	builders map[string]BuilderFunc
}

// NewRegistry creates a new processor registry.
func NewRegistry() *Registry {
	return &Registry{
		builders: make(map[string]BuilderFunc),
	}
}

// Register adds a processor builder to the registry.
// Name should be unique and match the processor's Name() return value.
func (r *Registry) Register(name string, builder BuilderFunc) {
	r.builders[name] = builder
}

// Build creates a processor by name with the given config.
// Returns error if the processor name is not registered.
func (r *Registry) Build(name string, cfg map[string]any) (driven.PostProcessor, error) {
	builder, ok := r.builders[name]
	if !ok {
		return nil, fmt.Errorf("unknown processor: %s", name)
	}
	return builder(cfg)
}

// Has returns true if a processor with the given name is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.builders[name]
	return ok
}

// Names returns all registered processor names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.builders))
	for name := range r.builders {
		names = append(names, name)
	}
	return names
}
