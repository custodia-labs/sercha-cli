package memory

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigStore(t *testing.T) {
	store := NewConfigStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.values)
}

func TestConfigStore_Set_Success(t *testing.T) {
	store := NewConfigStore()

	err := store.Set("key1", "value1")
	require.NoError(t, err)

	val, ok := store.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestConfigStore_Set_Update(t *testing.T) {
	store := NewConfigStore()

	err := store.Set("key1", "original")
	require.NoError(t, err)

	err = store.Set("key1", "updated")
	require.NoError(t, err)

	val, ok := store.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "updated", val)
}

func TestConfigStore_Set_MultipleKeys(t *testing.T) {
	store := NewConfigStore()

	keys := map[string]any{
		"string_key": "string_value",
		"int_key":    42,
		"bool_key":   true,
		"float_key":  3.14,
	}

	for k, v := range keys {
		err := store.Set(k, v)
		require.NoError(t, err)
	}

	// Verify all were set
	for k, expected := range keys {
		val, ok := store.Get(k)
		assert.True(t, ok)
		assert.Equal(t, expected, val)
	}
}

func TestConfigStore_Set_NilValue(t *testing.T) {
	store := NewConfigStore()

	err := store.Set("key1", nil)
	require.NoError(t, err)

	val, ok := store.Get("key1")
	assert.True(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Set_EmptyKey(t *testing.T) {
	store := NewConfigStore()

	err := store.Set("", "value")
	require.NoError(t, err)

	val, ok := store.Get("")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestConfigStore_Set_ComplexTypes(t *testing.T) {
	store := NewConfigStore()

	// Map
	mapValue := map[string]string{"nested": "value"}
	err := store.Set("map_key", mapValue)
	require.NoError(t, err)

	// Slice
	sliceValue := []string{"a", "b", "c"}
	err = store.Set("slice_key", sliceValue)
	require.NoError(t, err)

	// Struct
	type Config struct {
		Name  string
		Value int
	}
	structValue := Config{Name: "test", Value: 123}
	err = store.Set("struct_key", structValue)
	require.NoError(t, err)

	// Verify they can be retrieved
	val1, ok1 := store.Get("map_key")
	assert.True(t, ok1)
	assert.Equal(t, mapValue, val1)

	val2, ok2 := store.Get("slice_key")
	assert.True(t, ok2)
	assert.Equal(t, sliceValue, val2)

	val3, ok3 := store.Get("struct_key")
	assert.True(t, ok3)
	assert.Equal(t, structValue, val3)
}

func TestConfigStore_Get_NotFound(t *testing.T) {
	store := NewConfigStore()

	val, ok := store.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Get_EmptyStore(t *testing.T) {
	store := NewConfigStore()

	val, ok := store.Get("any_key")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Get_AfterDelete(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", "value1")

	// Manually delete from map to simulate removal
	store.mu.Lock()
	delete(store.values, "key1")
	store.mu.Unlock()

	val, ok := store.Get("key1")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_GetString_Success(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", "string_value")

	val := store.GetString("key1")
	assert.Equal(t, "string_value", val)
}

func TestConfigStore_GetString_NotFound(t *testing.T) {
	store := NewConfigStore()

	val := store.GetString("nonexistent")
	assert.Equal(t, "", val)
}

func TestConfigStore_GetString_WrongType(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", 123) // int, not string

	val := store.GetString("key1")
	assert.Equal(t, "", val)
}

func TestConfigStore_GetString_EmptyString(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", "")

	val := store.GetString("key1")
	assert.Equal(t, "", val)
}

func TestConfigStore_GetInt_Success(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", 42)

	val := store.GetInt("key1")
	assert.Equal(t, 42, val)
}

func TestConfigStore_GetInt_NotFound(t *testing.T) {
	store := NewConfigStore()

	val := store.GetInt("nonexistent")
	assert.Equal(t, 0, val)
}

func TestConfigStore_GetInt_FromInt64(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", int64(123))

	val := store.GetInt("key1")
	assert.Equal(t, 123, val)
}

func TestConfigStore_GetInt_FromFloat64(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", float64(123.7))

	val := store.GetInt("key1")
	assert.Equal(t, 123, val)
}

func TestConfigStore_GetInt_WrongType(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", "not_a_number")

	val := store.GetInt("key1")
	assert.Equal(t, 0, val)
}

func TestConfigStore_GetInt_Zero(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", 0)

	val := store.GetInt("key1")
	assert.Equal(t, 0, val)
}

func TestConfigStore_GetBool_Success(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", true)

	val := store.GetBool("key1")
	assert.True(t, val)

	_ = store.Set("key2", false)
	val2 := store.GetBool("key2")
	assert.False(t, val2)
}

func TestConfigStore_GetBool_NotFound(t *testing.T) {
	store := NewConfigStore()

	val := store.GetBool("nonexistent")
	assert.False(t, val)
}

func TestConfigStore_GetBool_WrongType(t *testing.T) {
	store := NewConfigStore()

	_ = store.Set("key1", "true") // string, not bool

	val := store.GetBool("key1")
	assert.False(t, val)
}

func TestConfigStore_Save_NoOp(t *testing.T) {
	store := NewConfigStore()

	// Save should not error for memory store
	err := store.Save()
	assert.NoError(t, err)

	// Data should still be accessible
	_ = store.Set("key1", "value1")
	err = store.Save()
	assert.NoError(t, err)

	val := store.GetString("key1")
	assert.Equal(t, "value1", val)
}

func TestConfigStore_Load_NoOp(t *testing.T) {
	store := NewConfigStore()

	// Load should not error for memory store
	err := store.Load()
	assert.NoError(t, err)

	// Should start with empty state
	val, ok := store.Get("any_key")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestConfigStore_Path(t *testing.T) {
	store := NewConfigStore()

	path := store.Path()
	assert.Equal(t, ":memory:", path)
}

func TestConfigStore_Concurrency_SetAndGet(t *testing.T) {
	store := NewConfigStore()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent sets
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := "key-" + string(rune('A'+id))
			value := "value-" + string(rune('A'+id))
			_ = store.Set(key, value)
		}(i)
	}
	wg.Wait()

	// Concurrent gets
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := "key-" + string(rune('A'+id))
			_, _ = store.Get(key)
		}(i)
	}
	wg.Wait()

	// Verify all were set
	for i := 0; i < numGoroutines; i++ {
		key := "key-" + string(rune('A'+i))
		val, ok := store.Get(key)
		assert.True(t, ok)
		assert.NotNil(t, val)
	}
}

func TestConfigStore_Concurrency_MixedOperations(t *testing.T) {
	store := NewConfigStore()

	var wg sync.WaitGroup
	numOperations := 100

	// Pre-populate
	for i := 0; i < 10; i++ {
		_ = store.Set("key-"+string(rune('0'+i)), "value-"+string(rune('0'+i)))
	}

	// Run mixed concurrent operations
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			switch id % 5 {
			case 0: // Set
				_ = store.Set("key-concurrent-"+string(rune('A'+id%26)), id)
			case 1: // Get
				_, _ = store.Get("key-" + string(rune('0'+id%10)))
			case 2: // GetString
				_ = store.GetString("key-" + string(rune('0'+id%10)))
			case 3: // GetInt
				_ = store.GetInt("key-concurrent-" + string(rune('A'+id%26)))
			case 4: // GetBool
				_ = store.GetBool("key-" + string(rune('0'+id%10)))
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	_, _ = store.Get("key-0")
}

func TestConfigStore_Concurrency_UpdateSameKey(t *testing.T) {
	store := NewConfigStore()

	// Set initial value
	_ = store.Set("shared-key", "initial")

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent updates to the same key
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_ = store.Set("shared-key", "updated-"+string(rune('A'+id)))
		}(i)
	}
	wg.Wait()

	// Verify key exists and has some update
	val, ok := store.Get("shared-key")
	assert.True(t, ok)
	assert.NotEqual(t, "initial", val)
}

func TestConfigStore_Concurrency_ReadWriteMix(t *testing.T) {
	store := NewConfigStore()

	// Pre-populate
	for i := 0; i < 10; i++ {
		_ = store.Set("key-"+string(rune('0'+i)), i)
	}

	var wg sync.WaitGroup
	numReaders := 50
	numWriters := 25

	// Concurrent readers
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = store.Get("key-" + string(rune('0'+j)))
			}
		}(i)
	}

	// Concurrent writers
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = store.Set("key-"+string(rune('0'+j)), id*10+j)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock
	for i := 0; i < 10; i++ {
		val, ok := store.Get("key-" + string(rune('0'+i)))
		assert.True(t, ok)
		assert.NotNil(t, val)
	}
}

func TestConfigStore_DataIsolation(t *testing.T) {
	store := NewConfigStore()

	// Set a map value
	originalMap := map[string]string{"key": "value"}
	_ = store.Set("config", originalMap)

	// Modify the original
	originalMap["key"] = "modified"
	originalMap["new_key"] = "new_value"

	// Retrieved value should reflect changes (no deep copy in memory store)
	val, ok := store.Get("config")
	require.True(t, ok)
	retrievedMap, ok := val.(map[string]string)
	require.True(t, ok)
	// Note: Memory store doesn't deep copy, so modifications to the original affect the stored value
	// In practice, callers should not modify retrieved values
	assert.Equal(t, "modified", retrievedMap["key"])
}

func TestConfigStore_TypeAssertions(t *testing.T) {
	store := NewConfigStore()

	// Store different types
	_ = store.Set("string", "value")
	_ = store.Set("int", 42)
	_ = store.Set("int64", int64(43))
	_ = store.Set("float", 3.14)
	_ = store.Set("bool", true)
	_ = store.Set("map", map[string]int{"a": 1})
	_ = store.Set("slice", []int{1, 2, 3})

	// Test GetString
	assert.Equal(t, "value", store.GetString("string"))
	assert.Equal(t, "", store.GetString("int"))
	assert.Equal(t, "", store.GetString("bool"))

	// Test GetInt
	assert.Equal(t, 42, store.GetInt("int"))
	assert.Equal(t, 43, store.GetInt("int64"))
	assert.Equal(t, 3, store.GetInt("float"))
	assert.Equal(t, 0, store.GetInt("string"))
	assert.Equal(t, 0, store.GetInt("bool"))

	// Test GetBool
	assert.True(t, store.GetBool("bool"))
	assert.False(t, store.GetBool("int"))
	assert.False(t, store.GetBool("string"))
}

func TestConfigStore_MultipleInstances(t *testing.T) {
	store1 := NewConfigStore()
	store2 := NewConfigStore()

	_ = store1.Set("key1", "value1")
	_ = store2.Set("key2", "value2")

	// Each store should be independent
	val1, ok1 := store1.Get("key1")
	assert.True(t, ok1)
	assert.Equal(t, "value1", val1)

	_, ok2 := store1.Get("key2")
	assert.False(t, ok2)

	val3, ok3 := store2.Get("key2")
	assert.True(t, ok3)
	assert.Equal(t, "value2", val3)

	_, ok4 := store2.Get("key1")
	assert.False(t, ok4)
}

func TestConfigStore_InterfaceCompliance(t *testing.T) {
	store := NewConfigStore()

	// Verify all interface methods work
	// Set
	err := store.Set("test-key", "test-value")
	assert.NoError(t, err)

	// Get
	val, ok := store.Get("test-key")
	assert.True(t, ok)
	assert.Equal(t, "test-value", val)

	// GetString
	strVal := store.GetString("test-key")
	assert.Equal(t, "test-value", strVal)

	// GetInt
	_ = store.Set("int-key", 42)
	intVal := store.GetInt("int-key")
	assert.Equal(t, 42, intVal)

	// GetBool
	_ = store.Set("bool-key", true)
	boolVal := store.GetBool("bool-key")
	assert.True(t, boolVal)

	// Save
	err = store.Save()
	assert.NoError(t, err)

	// Load
	err = store.Load()
	assert.NoError(t, err)

	// Path
	path := store.Path()
	assert.Equal(t, ":memory:", path)
}

func TestConfigStore_LargeDataset(t *testing.T) {
	store := NewConfigStore()

	// Set 1000 keys
	for i := 0; i < 1000; i++ {
		key := "key-" + string(rune(i))
		value := "value-" + string(rune(i))
		err := store.Set(key, value)
		require.NoError(t, err)
	}

	// Verify all can be retrieved
	for i := 0; i < 1000; i++ {
		key := "key-" + string(rune(i))
		val, ok := store.Get(key)
		assert.True(t, ok)
		assert.NotNil(t, val)
	}
}

func TestConfigStore_SpecialCharactersInKeys(t *testing.T) {
	store := NewConfigStore()

	specialKeys := []string{
		"key with spaces",
		"key.with.dots",
		"key/with/slashes",
		"key-with-dashes",
		"key_with_underscores",
		"key:with:colons",
		"key@with@at",
		"key#with#hash",
		"",
	}

	for _, key := range specialKeys {
		err := store.Set(key, "value")
		require.NoError(t, err)

		val, ok := store.Get(key)
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	}
}

func TestConfigStore_NilAndZeroValues(t *testing.T) {
	store := NewConfigStore()

	// Test nil
	_ = store.Set("nil-key", nil)
	val1, ok1 := store.Get("nil-key")
	assert.True(t, ok1)
	assert.Nil(t, val1)

	// Test zero int
	_ = store.Set("zero-int", 0)
	val2 := store.GetInt("zero-int")
	assert.Equal(t, 0, val2)

	// Test false bool
	_ = store.Set("false-bool", false)
	val3 := store.GetBool("false-bool")
	assert.False(t, val3)

	// Test empty string
	_ = store.Set("empty-string", "")
	val4 := store.GetString("empty-string")
	assert.Equal(t, "", val4)
}
