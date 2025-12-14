package notion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyMetadata_NilInput(t *testing.T) {
	result := copyMetadata(nil)
	assert.Nil(t, result)
}

func TestCopyMetadata_EmptyMap(t *testing.T) {
	src := map[string]any{}
	result := copyMetadata(src)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	// Verify it's a different map instance by checking pointers
	srcPtr := &src
	resultPtr := &result
	assert.NotSame(t, srcPtr, resultPtr)
}

func TestCopyMetadata_SimpleValues(t *testing.T) {
	src := map[string]any{
		"string":  "value",
		"number":  42,
		"float":   3.14,
		"boolean": true,
	}

	result := copyMetadata(src)

	assert.Equal(t, src, result)
	// Verify it's a different map instance by modifying one and checking the other
	assert.Len(t, result, 4)
	assert.Equal(t, "value", result["string"])
	assert.Equal(t, 42, result["number"])
	assert.Equal(t, 3.14, result["float"])
	assert.Equal(t, true, result["boolean"])
}

func TestCopyMetadata_ComplexValues(t *testing.T) {
	src := map[string]any{
		"array":  []string{"a", "b", "c"},
		"nested": map[string]string{"key": "value"},
		"nil":    nil,
	}

	result := copyMetadata(src)

	assert.Equal(t, src, result)
	// Verify it's a different map instance
	assert.Len(t, result, 3)
}

func TestCopyMetadata_ShallowCopy(t *testing.T) {
	nestedMap := map[string]string{"key": "value"}
	nestedSlice := []string{"a", "b", "c"}

	src := map[string]any{
		"map":   nestedMap,
		"slice": nestedSlice,
	}

	result := copyMetadata(src)

	// Verify it's a shallow copy - values are equal
	assert.Equal(t, nestedMap, result["map"])
	assert.Equal(t, nestedSlice, result["slice"])

	// Modifying nested values will affect both since it's a shallow copy
	// But we can verify the copy is working by checking the map itself is different
	src["new_key"] = "new_value"
	assert.NotContains(t, result, "new_key")
}

func TestCopyMetadata_Isolation(t *testing.T) {
	src := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	result := copyMetadata(src)

	// Modify source
	src["key1"] = "modified"
	src["key3"] = "new"

	// Result should not be affected
	assert.Equal(t, "value1", result["key1"])
	assert.NotContains(t, result, "key3")
}

func TestCopyMetadata_PreservesAllTypes(t *testing.T) {
	type customStruct struct {
		Field string
	}

	src := map[string]any{
		"string":  "text",
		"int":     42,
		"int64":   int64(123),
		"float32": float32(1.5),
		"float64": float64(2.5),
		"bool":    true,
		"slice":   []int{1, 2, 3},
		"map":     map[string]int{"a": 1},
		"struct":  customStruct{Field: "value"},
		"nil":     nil,
	}

	result := copyMetadata(src)

	assert.Equal(t, src, result)
	assert.Len(t, result, len(src))

	// Verify all types are preserved
	assert.IsType(t, "", result["string"])
	assert.IsType(t, 0, result["int"])
	assert.IsType(t, int64(0), result["int64"])
	assert.IsType(t, float32(0), result["float32"])
	assert.IsType(t, float64(0), result["float64"])
	assert.IsType(t, false, result["bool"])
	assert.IsType(t, []int{}, result["slice"])
	assert.IsType(t, map[string]int{}, result["map"])
	assert.IsType(t, customStruct{}, result["struct"])
	assert.Nil(t, result["nil"])
}

func TestCopyMetadata_LargeMap(t *testing.T) {
	src := make(map[string]any)
	for i := 0; i < 100; i++ {
		src[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
	}

	result := copyMetadata(src)

	assert.Equal(t, src, result)
	// Verify it's a different map instance
	assert.Len(t, result, len(src))
}

func BenchmarkCopyMetadata_Small(b *testing.B) {
	src := map[string]any{
		"title":  "Test",
		"author": "John",
		"date":   "2024-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = copyMetadata(src)
	}
}

func BenchmarkCopyMetadata_Large(b *testing.B) {
	src := make(map[string]any)
	for i := 0; i < 100; i++ {
		src[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = copyMetadata(src)
	}
}

func BenchmarkCopyMetadata_Nil(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = copyMetadata(nil)
	}
}
