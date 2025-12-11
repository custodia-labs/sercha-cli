package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.NotNil(t, cursor.SyncTokens)
	assert.Empty(t, cursor.SyncTokens)
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.SetSyncToken("primary", "token-123")
	cursor.SetSyncToken("work", "token-456")

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	// Should be valid base64
	assert.NotContains(t, encoded, " ")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.SetSyncToken("primary", "token-abc")
	original.SetSyncToken("work@example.com", "token-xyz")

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, "token-abc", decoded.GetSyncToken("primary"))
	assert.Equal(t, "token-xyz", decoded.GetSyncToken("work@example.com"))
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.NotNil(t, cursor.SyncTokens)
	assert.Empty(t, cursor.SyncTokens)
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
	// {"v":999,"sync_tokens":{"primary":"abc"}}
	futureVersionBase64 := "eyJ2Ijo5OTksInN5bmNfdG9rZW5zIjp7InByaW1hcnkiOiJhYmMifX0="

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestDecodeCursor_NilSyncTokensMap(t *testing.T) {
	// Create a cursor with null sync_tokens
	// {"v":1,"sync_tokens":null}
	nullMapBase64 := "eyJ2IjoxLCJzeW5jX3Rva2VucyI6bnVsbH0="

	cursor, err := DecodeCursor(nullMapBase64)

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.NotNil(t, cursor.SyncTokens, "SyncTokens should be initialized even if null in JSON")
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name       string
		syncTokens map[string]string
		want       bool
	}{
		{
			name:       "empty when no sync tokens",
			syncTokens: map[string]string{},
			want:       true,
		},
		{
			name: "not empty when sync tokens exist",
			syncTokens: map[string]string{
				"primary": "token-123",
			},
			want: false,
		},
		{
			name: "not empty with multiple sync tokens",
			syncTokens: map[string]string{
				"primary": "token-1",
				"work":    "token-2",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := NewCursor()
			for k, v := range tt.syncTokens {
				cursor.SetSyncToken(k, v)
			}

			assert.Equal(t, tt.want, cursor.IsEmpty())
		})
	}
}

func TestCursor_GetSyncToken(t *testing.T) {
	cursor := NewCursor()
	cursor.SetSyncToken("primary", "token-abc")
	cursor.SetSyncToken("work", "token-xyz")

	assert.Equal(t, "token-abc", cursor.GetSyncToken("primary"))
	assert.Equal(t, "token-xyz", cursor.GetSyncToken("work"))
	assert.Empty(t, cursor.GetSyncToken("nonexistent"))
}

func TestCursor_SetSyncToken(t *testing.T) {
	cursor := NewCursor()

	cursor.SetSyncToken("cal-1", "token-1")
	assert.Equal(t, "token-1", cursor.SyncTokens["cal-1"])

	// Overwrite existing
	cursor.SetSyncToken("cal-1", "token-updated")
	assert.Equal(t, "token-updated", cursor.SyncTokens["cal-1"])
}

func TestCursor_RoundTrip(t *testing.T) {
	// Test multiple round trips preserve data
	original := NewCursor()
	original.SetSyncToken("primary", "very-long-sync-token-1234567890")
	original.SetSyncToken("work@company.com", "another-token-abcdef")
	original.SetSyncToken("shared-calendar-id", "third-token")

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.SyncTokens, decoded.SyncTokens)
		original = decoded
	}
}

func TestCursor_MultipleCalendars(t *testing.T) {
	cursor := NewCursor()

	// Simulate syncing multiple calendars
	calendars := []string{
		"primary",
		"user@gmail.com",
		"work@company.com",
		"en.usa#holiday@group.v.calendar.google.com",
	}

	for i, cal := range calendars {
		cursor.SetSyncToken(cal, "token-"+string(rune('A'+i)))
	}

	// Verify all tokens are stored
	assert.Len(t, cursor.SyncTokens, 4)

	// Encode and decode
	decoded, err := DecodeCursor(cursor.Encode())
	require.NoError(t, err)

	// Verify all tokens preserved
	for i, cal := range calendars {
		assert.Equal(t, "token-"+string(rune('A'+i)), decoded.GetSyncToken(cal))
	}
}
