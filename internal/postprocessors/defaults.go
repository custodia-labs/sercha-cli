package postprocessors

import (
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/postprocessors/chunker"
)

// RegisterDefaults registers all built-in processors with the registry.
// Call this during application initialisation to enable standard processors.
func RegisterDefaults(r *Registry) {
	r.Register("chunker", buildChunker)
}

// buildChunker creates a chunker processor from generic config.
// Supported config keys:
//   - chunk_size (int): Characters per chunk (default: 1000)
//   - overlap (int): Overlapping characters between chunks (default: 200)
func buildChunker(cfg map[string]any) (driven.PostProcessor, error) {
	var opts []chunker.Option

	if cfg != nil {
		if size := getIntFromConfig(cfg, "chunk_size"); size > 0 {
			opts = append(opts, chunker.WithChunkSize(size))
		}
		if overlap := getIntFromConfig(cfg, "overlap"); overlap >= 0 {
			opts = append(opts, chunker.WithOverlap(overlap))
		}
	}

	return chunker.New(opts...), nil
}

// getIntFromConfig safely extracts an int from generic config map.
// Handles int, int64, and float64 types that may come from TOML/JSON parsing.
func getIntFromConfig(cfg map[string]any, key string) int {
	val, ok := cfg[key]
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
