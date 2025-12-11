// Package logger provides verbose logging for the Sercha CLI.
// When verbose mode is enabled via the --verbose flag, debug messages
// are printed to stderr to help users understand the search pipeline.
package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	mu      sync.RWMutex
	verbose bool
	output  io.Writer = os.Stderr
)

// SetVerbose enables or disables verbose logging.
func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
}

// IsVerbose returns true if verbose mode is enabled.
func IsVerbose() bool {
	mu.RLock()
	defer mu.RUnlock()
	return verbose
}

// SetOutput sets the output writer for verbose logs.
// Defaults to os.Stderr. Useful for testing.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
}

// Debug prints a message if verbose mode is enabled.
func Debug(format string, args ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if verbose {
		fmt.Fprintf(output, "[DEBUG] "+format+"\n", args...)
	}
}

// Section prints a section header if verbose mode is enabled.
func Section(name string) {
	mu.RLock()
	defer mu.RUnlock()
	if verbose {
		fmt.Fprintf(output, "\n=== %s ===\n", name)
	}
}

// Info prints an informational message if verbose mode is enabled.
func Info(format string, args ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if verbose {
		fmt.Fprintf(output, "[INFO] "+format+"\n", args...)
	}
}

// Warn prints a warning message if verbose mode is enabled.
func Warn(format string, args ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if verbose {
		fmt.Fprintf(output, "[WARN] "+format+"\n", args...)
	}
}
