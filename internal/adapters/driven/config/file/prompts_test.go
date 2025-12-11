package file

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestPromptStore_ImplementsInterface(t *testing.T) {
	var _ driven.PromptStore = (*PromptStore)(nil)
}

func TestNewPromptStore_WithCustomDir(t *testing.T) {
	dir := t.TempDir()

	store, err := NewPromptStore(dir)

	require.NoError(t, err)
	assert.Equal(t, dir, store.Dir())
}

func TestNewPromptStore_DefaultDir(t *testing.T) {
	// Skip if we can't determine home dir
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	store, err := NewPromptStore("")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".sercha", "prompts"), store.Dir())
}

func TestPromptStore_Load_CreatesDefaultFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	// Load triggers lazy init
	_, err = store.Load(driven.PromptQueryRewrite)
	require.NoError(t, err)

	// Check files were created
	files := []string{
		"query_rewrite.txt",
		"summarise.txt",
		"chat_system.txt",
		"README.md",
	}
	for _, f := range files {
		path := filepath.Join(dir, f)
		_, err := os.Stat(path)
		assert.NoError(t, err, "expected file %s to exist", f)
	}
}

func TestPromptStore_Load_ReturnsDefaultContent(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	prompt, err := store.Load(driven.PromptQueryRewrite)

	require.NoError(t, err)
	assert.Contains(t, prompt, "Rewrite this search query")
	assert.Contains(t, prompt, "%s") // Format placeholder
}

func TestPromptStore_Load_ReturnsCustomContent(t *testing.T) {
	dir := t.TempDir()

	// Create custom prompt before store init
	customContent := "My custom prompt: %s"
	err := os.WriteFile(
		filepath.Join(dir, "query_rewrite.txt"),
		[]byte(customContent),
		0600,
	)
	require.NoError(t, err)

	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	prompt, err := store.Load(driven.PromptQueryRewrite)

	require.NoError(t, err)
	assert.Equal(t, customContent, prompt)
}

func TestPromptStore_Load_FallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	// Delete the file after init creates it
	_, _ = store.Load(driven.PromptQueryRewrite) // Trigger init
	os.Remove(filepath.Join(dir, "query_rewrite.txt"))
	store.Reload() // Clear cache

	// Should fall back to embedded default
	prompt, err := store.Load(driven.PromptQueryRewrite)

	require.NoError(t, err)
	assert.Contains(t, prompt, "Rewrite this search query")
}

func TestPromptStore_Load_UnknownPrompt(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	_, err = store.Load("nonexistent_prompt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_prompt")
}

func TestPromptStore_Load_CachesResults(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	// First load
	prompt1, err := store.Load(driven.PromptQueryRewrite)
	require.NoError(t, err)

	// Modify file on disk
	err = os.WriteFile(
		filepath.Join(dir, "query_rewrite.txt"),
		[]byte("modified content"),
		0600,
	)
	require.NoError(t, err)

	// Second load should return cached value
	prompt2, err := store.Load(driven.PromptQueryRewrite)
	require.NoError(t, err)

	assert.Equal(t, prompt1, prompt2)
}

func TestPromptStore_Reload_ClearsCache(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	// First load
	_, err = store.Load(driven.PromptQueryRewrite)
	require.NoError(t, err)

	// Modify file on disk
	modifiedContent := "modified content: %s"
	err = os.WriteFile(
		filepath.Join(dir, "query_rewrite.txt"),
		[]byte(modifiedContent),
		0600,
	)
	require.NoError(t, err)

	// Reload cache
	store.Reload()

	// Should return new content
	prompt, err := store.Load(driven.PromptQueryRewrite)
	require.NoError(t, err)

	assert.Equal(t, modifiedContent, prompt)
}

func TestPromptStore_Load_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines)
	prompts := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			prompt, err := store.Load(driven.PromptQueryRewrite)
			if err != nil {
				errors <- err
				return
			}
			prompts <- prompt
		}()
	}

	wg.Wait()
	close(errors)
	close(prompts)

	// Check no errors
	for err := range errors {
		t.Errorf("unexpected error: %v", err)
	}

	// Check all prompts are identical
	var first string
	for prompt := range prompts {
		if first == "" {
			first = prompt
		} else {
			assert.Equal(t, first, prompt)
		}
	}
}

func TestPromptStore_DoesNotOverwriteExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create custom prompt before store creation
	customContent := "pre-existing custom prompt"
	err := os.WriteFile(
		filepath.Join(dir, "query_rewrite.txt"),
		[]byte(customContent),
		0600,
	)
	require.NoError(t, err)

	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	// Trigger init
	_, _ = store.Load(driven.PromptSummarise)

	// Original file should be unchanged
	data, err := os.ReadFile(filepath.Join(dir, "query_rewrite.txt"))
	require.NoError(t, err)
	assert.Equal(t, customContent, string(data))
}

func TestPromptStore_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()

	// Create prompt with extra whitespace
	contentWithWhitespace := "\n\n  prompt content  \n\n"
	err := os.WriteFile(
		filepath.Join(dir, "query_rewrite.txt"),
		[]byte(contentWithWhitespace),
		0600,
	)
	require.NoError(t, err)

	store, err := NewPromptStore(dir)
	require.NoError(t, err)

	prompt, err := store.Load(driven.PromptQueryRewrite)
	require.NoError(t, err)

	assert.Equal(t, "prompt content", prompt)
}
