package notion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.NotNil(t, cursor.PageStates)
	assert.Empty(t, cursor.PageStates)
	assert.True(t, cursor.LastSyncTime.IsZero())
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.SetPageState("page-1", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), false)
	cursor.SetPageState("db-1", time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC), true)
	cursor.SetLastSyncTime(time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC))

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	// Should be valid base64
	assert.NotContains(t, encoded, " ")
	assert.NotContains(t, encoded, "\n")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.SetPageState("page-abc", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), false)
	original.SetPageState("db-xyz", time.Date(2024, 1, 16, 14, 30, 0, 0, time.UTC), true)
	original.SetLastSyncTime(time.Date(2024, 1, 20, 16, 0, 0, 0, time.UTC))

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.LastSyncTime.Unix(), decoded.LastSyncTime.Unix())

	// Check page states
	state1 := decoded.GetPageState("page-abc")
	require.NotNil(t, state1)
	assert.Equal(t, original.GetPageState("page-abc").LastEditedTime.Unix(), state1.LastEditedTime.Unix())
	assert.False(t, state1.IsDatabase)

	state2 := decoded.GetPageState("db-xyz")
	require.NotNil(t, state2)
	assert.Equal(t, original.GetPageState("db-xyz").LastEditedTime.Unix(), state2.LastEditedTime.Unix())
	assert.True(t, state2.IsDatabase)
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.NotNil(t, cursor.PageStates)
	assert.Empty(t, cursor.PageStates)
	assert.True(t, cursor.LastSyncTime.IsZero())
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	cursor, err := DecodeCursor("not-valid-base64!!!")

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON
	// "not json" in base64
	cursor, err := DecodeCursor("bm90IGpzb24=")

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestDecodeCursor_FutureVersion(t *testing.T) {
	// Create a cursor with a future version (manually crafted)
	// {"v":999,"last_sync_time":"2024-01-20T12:00:00Z","page_states":{}}
	futureVersionBase64 := "eyJ2Ijo5OTksImxhc3Rfc3luY190aW1lIjoiMjAyNC0wMS0yMFQxMjowMDowMFoiLCJwYWdlX3N0YXRlcyI6e319"

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestDecodeCursor_NilPageStatesMap(t *testing.T) {
	// Create a cursor with null page_states
	// {"v":1,"last_sync_time":"2024-01-20T12:00:00Z","page_states":null}
	nullMapBase64 := "eyJ2IjoxLCJsYXN0X3N5bmNfdGltZSI6IjIwMjQtMDEtMjBUMTI6MDA6MDBaIiwicGFnZV9zdGF0ZXMiOm51bGx9"

	cursor, err := DecodeCursor(nullMapBase64)

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.NotNil(t, cursor.PageStates, "PageStates should be initialized even if null in JSON")
	assert.Empty(t, cursor.PageStates)
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name           string
		setupCursor    func() *Cursor
		expectedResult bool
	}{
		{
			name: "empty cursor",
			setupCursor: func() *Cursor {
				return NewCursor()
			},
			expectedResult: true,
		},
		{
			name: "cursor with page states",
			setupCursor: func() *Cursor {
				c := NewCursor()
				c.SetPageState("page-1", time.Now(), false)
				return c
			},
			expectedResult: false,
		},
		{
			name: "cursor with only last sync time",
			setupCursor: func() *Cursor {
				c := NewCursor()
				c.SetLastSyncTime(time.Now())
				return c
			},
			expectedResult: false,
		},
		{
			name: "cursor with both page states and sync time",
			setupCursor: func() *Cursor {
				c := NewCursor()
				c.SetPageState("page-1", time.Now(), false)
				c.SetLastSyncTime(time.Now())
				return c
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := tt.setupCursor()
			assert.Equal(t, tt.expectedResult, cursor.IsEmpty())
		})
	}
}

func TestCursor_SetPageState(t *testing.T) {
	cursor := NewCursor()
	editTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	cursor.SetPageState("page-1", editTime, false)

	state := cursor.GetPageState("page-1")
	require.NotNil(t, state)
	assert.Equal(t, editTime.Unix(), state.LastEditedTime.Unix())
	assert.False(t, state.IsDatabase)
}

func TestCursor_SetPageState_Database(t *testing.T) {
	cursor := NewCursor()
	editTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	cursor.SetPageState("db-1", editTime, true)

	state := cursor.GetPageState("db-1")
	require.NotNil(t, state)
	assert.Equal(t, editTime.Unix(), state.LastEditedTime.Unix())
	assert.True(t, state.IsDatabase)
}

func TestCursor_SetPageState_Overwrite(t *testing.T) {
	cursor := NewCursor()
	time1 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 20, 14, 0, 0, 0, time.UTC)

	cursor.SetPageState("page-1", time1, false)
	cursor.SetPageState("page-1", time2, true)

	state := cursor.GetPageState("page-1")
	require.NotNil(t, state)
	assert.Equal(t, time2.Unix(), state.LastEditedTime.Unix())
	assert.True(t, state.IsDatabase, "should update IsDatabase flag")
}

func TestCursor_GetPageState(t *testing.T) {
	cursor := NewCursor()
	editTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	cursor.SetPageState("page-1", editTime, false)
	cursor.SetPageState("db-1", editTime, true)

	state1 := cursor.GetPageState("page-1")
	require.NotNil(t, state1)
	assert.Equal(t, editTime.Unix(), state1.LastEditedTime.Unix())
	assert.False(t, state1.IsDatabase)

	state2 := cursor.GetPageState("db-1")
	require.NotNil(t, state2)
	assert.True(t, state2.IsDatabase)

	state3 := cursor.GetPageState("nonexistent")
	assert.Nil(t, state3)
}

func TestCursor_RemovePageState(t *testing.T) {
	cursor := NewCursor()
	editTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	cursor.SetPageState("page-1", editTime, false)
	assert.NotNil(t, cursor.GetPageState("page-1"))

	cursor.RemovePageState("page-1")
	assert.Nil(t, cursor.GetPageState("page-1"))

	// Removing non-existent page should not panic
	cursor.RemovePageState("nonexistent")
}

func TestCursor_SetLastSyncTime(t *testing.T) {
	cursor := NewCursor()
	syncTime := time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC)

	cursor.SetLastSyncTime(syncTime)

	assert.Equal(t, syncTime.Unix(), cursor.GetLastSyncTime().Unix())
}

func TestCursor_GetLastSyncTime(t *testing.T) {
	cursor := NewCursor()

	// Initially zero
	assert.True(t, cursor.GetLastSyncTime().IsZero())

	syncTime := time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC)
	cursor.SetLastSyncTime(syncTime)

	assert.Equal(t, syncTime.Unix(), cursor.GetLastSyncTime().Unix())
}

func TestCursor_GetAllPageIDs(t *testing.T) {
	cursor := NewCursor()
	editTime := time.Now()

	cursor.SetPageState("page-1", editTime, false)
	cursor.SetPageState("page-2", editTime, false)
	cursor.SetPageState("db-1", editTime, true)

	ids := cursor.GetAllPageIDs()

	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "page-1")
	assert.Contains(t, ids, "page-2")
	assert.Contains(t, ids, "db-1")
}

func TestCursor_GetAllPageIDs_Empty(t *testing.T) {
	cursor := NewCursor()

	ids := cursor.GetAllPageIDs()

	assert.NotNil(t, ids)
	assert.Empty(t, ids)
}

func TestCursor_IsPageUpdated(t *testing.T) {
	cursor := NewCursor()
	oldTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 1, 20, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		setupCursor    func()
		pageID         string
		lastEdited     time.Time
		expectedResult bool
	}{
		{
			name:           "new page",
			setupCursor:    func() {},
			pageID:         "new-page",
			lastEdited:     newTime,
			expectedResult: true,
		},
		{
			name: "page updated",
			setupCursor: func() {
				cursor.SetPageState("updated-page", oldTime, false)
			},
			pageID:         "updated-page",
			lastEdited:     newTime,
			expectedResult: true,
		},
		{
			name: "page not updated",
			setupCursor: func() {
				cursor.SetPageState("same-page", newTime, false)
			},
			pageID:         "same-page",
			lastEdited:     newTime,
			expectedResult: false,
		},
		{
			name: "page edited earlier",
			setupCursor: func() {
				cursor.SetPageState("old-edit", newTime, false)
			},
			pageID:         "old-edit",
			lastEdited:     oldTime,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor = NewCursor() // Reset cursor for each test
			tt.setupCursor()
			result := cursor.IsPageUpdated(tt.pageID, tt.lastEdited)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestCursor_RoundTrip(t *testing.T) {
	// Test multiple round trips preserve data
	original := NewCursor()
	original.SetPageState("page-1", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), false)
	original.SetPageState("page-2", time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC), false)
	original.SetPageState("db-1", time.Date(2024, 1, 17, 12, 0, 0, 0, time.UTC), true)
	original.SetLastSyncTime(time.Date(2024, 1, 20, 15, 0, 0, 0, time.UTC))

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)

		assert.Equal(t, original.Version, decoded.Version)
		assert.Equal(t, original.LastSyncTime.Unix(), decoded.LastSyncTime.Unix())
		assert.Len(t, decoded.PageStates, len(original.PageStates))

		for id, state := range original.PageStates {
			decodedState := decoded.GetPageState(id)
			require.NotNil(t, decodedState)
			assert.Equal(t, state.LastEditedTime.Unix(), decodedState.LastEditedTime.Unix())
			assert.Equal(t, state.IsDatabase, decodedState.IsDatabase)
		}

		original = decoded
	}
}

func TestCursor_MultiplePages(t *testing.T) {
	cursor := NewCursor()

	// Simulate tracking many pages
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		pageID := "page-" + string(rune('a'+i))
		editTime := baseTime.Add(time.Duration(i) * time.Hour)
		isDB := i%10 == 0
		cursor.SetPageState(pageID, editTime, isDB)
	}

	// Verify all pages are stored
	assert.Len(t, cursor.PageStates, 100)

	// Encode and decode
	decoded, err := DecodeCursor(cursor.Encode())
	require.NoError(t, err)

	// Verify all pages preserved
	assert.Len(t, decoded.PageStates, 100)

	// Verify specific pages
	for i := 0; i < 100; i++ {
		pageID := "page-" + string(rune('a'+i))
		state := decoded.GetPageState(pageID)
		require.NotNil(t, state)

		expectedTime := baseTime.Add(time.Duration(i) * time.Hour)
		assert.Equal(t, expectedTime.Unix(), state.LastEditedTime.Unix())
		assert.Equal(t, i%10 == 0, state.IsDatabase)
	}
}

func TestCursor_ConcurrentOperations(t *testing.T) {
	// Test that cursor operations are safe for basic usage patterns
	cursor := NewCursor()

	// Set up initial state
	cursor.SetPageState("page-1", time.Now(), false)
	cursor.SetPageState("page-2", time.Now(), true)
	cursor.SetLastSyncTime(time.Now())

	// Perform multiple operations
	cursor.SetPageState("page-3", time.Now(), false)
	ids := cursor.GetAllPageIDs()
	cursor.RemovePageState("page-1")
	encoded := cursor.Encode()

	// Verify state is consistent
	assert.Len(t, ids, 3)
	assert.NotEmpty(t, encoded)
	assert.Nil(t, cursor.GetPageState("page-1"))
	assert.NotNil(t, cursor.GetPageState("page-2"))
	assert.NotNil(t, cursor.GetPageState("page-3"))
}

func TestCursor_ZeroTimestamps(t *testing.T) {
	cursor := NewCursor()
	zeroTime := time.Time{}

	cursor.SetPageState("page-zero", zeroTime, false)
	cursor.SetLastSyncTime(zeroTime)

	state := cursor.GetPageState("page-zero")
	require.NotNil(t, state)
	assert.True(t, state.LastEditedTime.IsZero())
	assert.True(t, cursor.GetLastSyncTime().IsZero())

	// Encode and decode
	decoded, err := DecodeCursor(cursor.Encode())
	require.NoError(t, err)

	decodedState := decoded.GetPageState("page-zero")
	require.NotNil(t, decodedState)
	assert.True(t, decodedState.LastEditedTime.IsZero())
	assert.True(t, decoded.GetLastSyncTime().IsZero())
}

func TestCursor_VersionBackwardCompatibility(t *testing.T) {
	// Test that cursor with current version can be decoded
	cursor := NewCursor()
	cursor.SetPageState("page-1", time.Now(), false)

	encoded := cursor.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, CursorVersion, decoded.Version)
}
