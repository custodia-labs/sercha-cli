package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure PromptStore implements the interface.
var _ driven.PromptStore = (*PromptStore)(nil)

// PromptStore loads LLM prompts from user-editable files on disk.
// Prompts are loaded from a configurable directory with fallback to embedded defaults.
//
// The store uses lazy initialisation - files are only created when first accessed,
// not in the constructor. This makes testing easier and avoids unexpected I/O.
type PromptStore struct {
	mu        sync.RWMutex
	promptDir string
	cache     map[string]string
	initOnce  sync.Once
	initErr   error
}

// defaultPrompts contains embedded default prompts.
// These are used when user files don't exist and as the initial content for new files.
//
//nolint:lll // Prompt content is intentionally long and should not be wrapped.
var defaultPrompts = map[string]string{
	driven.PromptQueryRewrite: `Rewrite this search query to improve recall. Add synonyms and fix typos.
Return ONLY the rewritten query, nothing else.

Original: %s
Rewritten:`,

	driven.PromptSummarise: `Summarise the following content in %d characters or less.
Be concise and capture the key points.

Content:
%s

Summary:`,

	driven.PromptChatSystem: `You are Sercha, a knowledgeable search assistant. You help users find and understand information from their indexed documents.

You have access to the following tools:
- search(query): Search the document index and return relevant results
- get_document(id): Retrieve the full content of a specific document

When answering questions:
1. Use the search tool to find relevant documents
2. Cite your sources by referencing document titles
3. If you need more context, use get_document to read the full content
4. Be concise but thorough

Current conversation context will include previous search results.`,
}

// NewPromptStore creates a new file-based prompt store.
// If promptDir is empty, defaults to ~/.sercha/prompts/.
//
// The constructor does not perform any I/O - directory creation and
// file writes happen lazily on first Load() call.
func NewPromptStore(promptDir string) (*PromptStore, error) {
	if promptDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		promptDir = filepath.Join(home, ".sercha", "prompts")
	}

	return &PromptStore{
		promptDir: promptDir,
		cache:     make(map[string]string),
	}, nil
}

// Load returns the prompt template for the given name.
// On first call, initialises the prompt directory and creates default files.
// Returns cached value if available, otherwise loads from file.
// Falls back to embedded default if file doesn't exist.
func (s *PromptStore) Load(name string) (string, error) {
	// Ensure directory and defaults exist (lazy init)
	s.initOnce.Do(s.initialise)
	if s.initErr != nil {
		// Fall back to embedded defaults if init failed
		if prompt, ok := defaultPrompts[name]; ok {
			return prompt, nil
		}
		return "", fmt.Errorf("prompt store init failed: %w", s.initErr)
	}

	// Check cache first (read lock)
	s.mu.RLock()
	if prompt, ok := s.cache[name]; ok {
		s.mu.RUnlock()
		return prompt, nil
	}
	s.mu.RUnlock()

	// Load from file (no lock held during I/O)
	prompt, err := s.loadFromFile(name)
	if err != nil {
		// Fall back to embedded default
		if defaultPrompt, ok := defaultPrompts[name]; ok {
			return defaultPrompt, nil
		}
		return "", fmt.Errorf("load prompt %q: %w", name, err)
	}

	// Cache the result (write lock)
	// Use double-check pattern to avoid overwriting concurrent loads
	s.mu.Lock()
	if _, ok := s.cache[name]; !ok {
		s.cache[name] = prompt
	} else {
		// Another goroutine loaded it first, use their value
		prompt = s.cache[name]
	}
	s.mu.Unlock()

	return prompt, nil
}

// Reload clears the prompt cache, forcing fresh loads from disk.
func (s *PromptStore) Reload() {
	s.mu.Lock()
	s.cache = make(map[string]string)
	s.mu.Unlock()
}

// Dir returns the prompt directory path.
func (s *PromptStore) Dir() string {
	return s.promptDir
}

// initialise creates the prompt directory and default files.
// Called once via sync.Once on first Load().
func (s *PromptStore) initialise() {
	// Create directory
	if err := os.MkdirAll(s.promptDir, 0700); err != nil {
		s.initErr = fmt.Errorf("create prompt directory: %w", err)
		return
	}

	// Create default prompt files (only if they don't exist)
	for name, content := range defaultPrompts {
		path := filepath.Join(s.promptDir, name+".txt")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				s.initErr = fmt.Errorf("create default prompt %q: %w", name, err)
				return
			}
		}
	}

	// Create README
	if err := s.createReadme(); err != nil {
		s.initErr = err
	}
}

// loadFromFile reads a prompt from disk.
func (s *PromptStore) loadFromFile(name string) (string, error) {
	path := filepath.Join(s.promptDir, name+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// createReadme writes a README file explaining the prompts directory.
func (s *PromptStore) createReadme() error {
	path := filepath.Join(s.promptDir, "README.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // Already exists or stat error (ignore)
	}

	content := `# Sercha Prompts

This directory contains customisable prompts used by Sercha's LLM features.

## Files

- ` + "`query_rewrite.txt`" + ` - Expands search queries for better recall
- ` + "`summarise.txt`" + ` - Summarises document content
- ` + "`chat_system.txt`" + ` - System prompt for conversational search

## Customisation

Edit any file to customise LLM behaviour. Changes take effect on the next
command or after restarting the TUI.

## Format Placeholders

Some prompts use Go fmt placeholders:
- ` + "`%s`" + ` - String (e.g., the query or content)
- ` + "`%d`" + ` - Integer (e.g., max length)

Ensure customised prompts maintain placeholders in the correct positions.
`
	return os.WriteFile(path, []byte(content), 0600)
}
