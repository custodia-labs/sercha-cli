package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigStore_Success(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewConfigStore(tmpDir)

	require.NoError(t, err)
	require.NotNil(t, store)
	assert.Equal(t, filepath.Join(tmpDir, "config.toml"), store.Path())
}

func TestNewConfigStore_DefaultDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	store, err := NewConfigStore("")

	require.NoError(t, err)
	require.NotNil(t, store)
	assert.Equal(t, filepath.Join(home, ".sercha", "config.toml"), store.Path())

	// Cleanup
	_ = os.Remove(store.Path())
}

func TestConfigStore_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Set a string value
	err = store.Set("test_key", "test_value")
	require.NoError(t, err)

	// Get it back
	val, ok := store.Get("test_key")
	assert.True(t, ok)
	assert.Equal(t, "test_value", val)
}

func TestConfigStore_GetString(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	err = store.Set("string_key", "hello world")
	require.NoError(t, err)

	val := store.GetString("string_key")
	assert.Equal(t, "hello world", val)

	// Non-existent key
	val = store.GetString("nonexistent")
	assert.Equal(t, "", val)

	// Wrong type
	err = store.Set("int_key", 42)
	require.NoError(t, err)
	val = store.GetString("int_key")
	assert.Equal(t, "", val)
}

func TestConfigStore_GetInt(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	err = store.Set("int_key", 42)
	require.NoError(t, err)

	val := store.GetInt("int_key")
	assert.Equal(t, 42, val)

	// Non-existent key
	val = store.GetInt("nonexistent")
	assert.Equal(t, 0, val)

	// Wrong type
	err = store.Set("string_key", "not an int")
	require.NoError(t, err)
	val = store.GetInt("string_key")
	assert.Equal(t, 0, val)
}

func TestConfigStore_GetBool(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	err = store.Set("bool_key", true)
	require.NoError(t, err)

	val := store.GetBool("bool_key")
	assert.True(t, val)

	err = store.Set("bool_key_false", false)
	require.NoError(t, err)

	val = store.GetBool("bool_key_false")
	assert.False(t, val)

	// Non-existent key
	val = store.GetBool("nonexistent")
	assert.False(t, val)

	// Wrong type
	err = store.Set("string_key", "true")
	require.NoError(t, err)
	val = store.GetBool("string_key")
	assert.False(t, val)
}

func TestConfigStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	val, ok := store.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and set values
	store1, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	err = store1.Set("key1", "value1")
	require.NoError(t, err)
	err = store1.Set("key2", 42)
	require.NoError(t, err)
	err = store1.Set("key3", true)
	require.NoError(t, err)

	// Create new store instance - should load from file
	store2, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "value1", store2.GetString("key1"))
	assert.Equal(t, 42, store2.GetInt("key2"))
	assert.True(t, store2.GetBool("key3"))
}

func TestConfigStore_Load_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store - no config file exists yet
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Should start empty with no error
	val, ok := store.Get("any_key")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Save(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Modify data directly and save
	err = store.Set("save_test", "saved_value")
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(filepath.Join(tmpDir, "config.toml"))
	assert.NoError(t, err)
}

func TestConfigStore_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	err = store.Set("test", "value")
	require.NoError(t, err)

	info, err := os.Stat(store.Path())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestConfigStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty config file
	err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte{}, 0600)
	require.NoError(t, err)

	// Store should handle empty file gracefully
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	val, ok := store.Get("any_key")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key" + string(rune('0'+id))
			_ = store.Set(key, id)
			_ = store.GetInt(key)
			_ = store.GetString(key)
			_ = store.GetBool(key)
			_, _ = store.Get(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConfigStore_Path(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	path := store.Path()
	assert.Equal(t, filepath.Join(tmpDir, "config.toml"), path)
}

func TestConfigStore_OverwriteValue(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	err = store.Set("key", "original")
	require.NoError(t, err)
	assert.Equal(t, "original", store.GetString("key"))

	err = store.Set("key", "updated")
	require.NoError(t, err)
	assert.Equal(t, "updated", store.GetString("key"))
}

func TestConfigStore_MultipleTypes(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Set different types
	err = store.Set("string_val", "hello")
	require.NoError(t, err)
	err = store.Set("int_val", 123)
	require.NoError(t, err)
	err = store.Set("bool_val", true)
	require.NoError(t, err)
	err = store.Set("float_val", 3.14)
	require.NoError(t, err)

	// Verify persistence across reload
	store2, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "hello", store2.GetString("string_val"))
	assert.Equal(t, 123, store2.GetInt("int_val"))
	assert.True(t, store2.GetBool("bool_val"))

	// Float should be accessible via Get
	floatVal, ok := store2.Get("float_val")
	assert.True(t, ok)
	assert.Equal(t, 3.14, floatVal)
}

// TestNewConfigStore_MkdirAllError tests error handling when directory creation fails
func TestNewConfigStore_MkdirAllError(t *testing.T) {
	// Use an invalid path that would cause MkdirAll to fail
	// On Unix systems, using a path under /dev/null should fail
	invalidPath := "/dev/null/cannot/create/dirs"

	store, err := NewConfigStore(invalidPath)

	assert.Error(t, err)
	assert.Nil(t, store)
}

// TestNewConfigStore_LoadCorruptedFile tests error handling when loading corrupted TOML
func TestNewConfigStore_LoadCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a corrupted TOML file
	corruptedContent := []byte("this is not valid TOML {{{[[")
	err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), corruptedContent, 0600)
	require.NoError(t, err)

	// Attempting to create ConfigStore should fail due to corrupted TOML
	store, err := NewConfigStore(tmpDir)

	assert.Error(t, err)
	assert.Nil(t, store)
}

// TestConfigStore_Save_Explicit tests the public Save method
func TestConfigStore_Save_Explicit(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Manually modify internal data
	store.mu.Lock()
	store.data["manual_key"] = "manual_value"
	store.mu.Unlock()

	// Explicitly save
	err = store.Save()
	require.NoError(t, err)

	// Reload to verify
	store2, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	val := store2.GetString("manual_key")
	assert.Equal(t, "manual_value", val)
}

// TestConfigStore_Save_WriteFileError tests error handling when WriteFile fails
func TestConfigStore_Save_WriteFileError(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Set a value first
	err = store.Set("test", "value")
	require.NoError(t, err)

	// Replace the file with a directory to cause write error
	err = os.Remove(store.Path())
	require.NoError(t, err)
	err = os.Mkdir(store.Path(), 0700)
	require.NoError(t, err)

	// Attempt to save should fail (can't write to directory)
	err = store.Set("another", "value")
	assert.Error(t, err)
}

// TestConfigStore_Load_InvalidTOML tests error handling when loading invalid TOML
func TestConfigStore_Load_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()

	// First create a valid store
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)
	err = store.Set("valid", "data")
	require.NoError(t, err)

	// Now corrupt the TOML file
	corruptedContent := []byte("invalid toml syntax ][}{")
	err = os.WriteFile(store.Path(), corruptedContent, 0600)
	require.NoError(t, err)

	// Attempt to load should fail
	err = store.Load()
	assert.Error(t, err)
}

// TestConfigStore_Load_ReadFileError tests error handling when ReadFile fails
func TestConfigStore_Load_ReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Create a file and set it to no-read permissions
	err = store.Set("test", "value")
	require.NoError(t, err)

	err = os.Chmod(store.Path(), 0000) // no permissions
	require.NoError(t, err)

	// Attempt to load should fail
	err = store.Load()
	assert.Error(t, err)
	assert.False(t, os.IsNotExist(err))

	// Restore permissions for cleanup
	_ = os.Chmod(store.Path(), 0600)
}

// TestConfigStore_SetWithUnmarshallableValue tests error handling with values that can't be marshaled
func TestConfigStore_SetWithUnmarshallableValue(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Channels cannot be marshaled to TOML
	ch := make(chan int)
	err = store.Set("channel", ch)

	assert.Error(t, err)
}

// TestConfigStore_GetInt_Int64Type tests GetInt with int64 type (from TOML)
func TestConfigStore_GetInt_Int64Type(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Manually set an int64 value (simulating TOML unmarshal)
	store.mu.Lock()
	store.data["int64_key"] = int64(9999)
	store.mu.Unlock()

	val := store.GetInt("int64_key")
	assert.Equal(t, 9999, val)
}

// TestNewConfigStore_WithNestedDirectory tests creating config in nested directories
func TestNewConfigStore_WithNestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "deep", "path")

	store, err := NewConfigStore(nestedPath)

	require.NoError(t, err)
	require.NotNil(t, store)
	assert.Equal(t, filepath.Join(nestedPath, "config.toml"), store.Path())

	// Verify directory was created
	info, err := os.Stat(nestedPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify directory permissions
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

// TestConfigStore_SaveReload_PreservesData tests that all data is preserved through save/reload cycle
func TestConfigStore_SaveReload_PreservesData(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Set various values
	testData := map[string]any{
		"key1": "string_value",
		"key2": int64(42),
		"key3": true,
		"key4": false,
		"key5": 3.14159,
	}

	for key, val := range testData {
		err = store.Set(key, val)
		require.NoError(t, err)
	}

	// Create new store instance
	store2, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Verify all data
	assert.Equal(t, "string_value", store2.GetString("key1"))
	assert.Equal(t, 42, store2.GetInt("key2"))
	assert.True(t, store2.GetBool("key3"))
	assert.False(t, store2.GetBool("key4"))
	floatVal, ok := store2.Get("key5")
	assert.True(t, ok)
	assert.InDelta(t, 3.14159, floatVal, 0.00001)
}

// TestConfigStore_Load_EmptyTOMLData tests handling of TOML file with nil data
func TestConfigStore_Load_EmptyTOMLData(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a TOML file that unmarshals to nil
	// An empty or whitespace-only file should result in nil map
	emptyContent := []byte("# Just a comment\n\n")
	err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), emptyContent, 0600)
	require.NoError(t, err)

	store, err := NewConfigStore(tmpDir)
	require.NoError(t, err)

	// Should initialize with empty map
	val, ok := store.Get("any_key")
	assert.False(t, ok)
	assert.Nil(t, val)
}
