package dropbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.Cursor)
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.SetCursor("test-dropbox-cursor-abc123")

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	// Should be valid base64 (no spaces)
	assert.NotContains(t, encoded, " ")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.SetCursor("dropbox-cursor-xyz789")

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.Cursor, decoded.GetCursor())
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.Cursor)
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
	// {"v":999,"cursor":"test"}
	futureVersionBase64 := "eyJ2Ijo5OTksImN1cnNvciI6InRlc3QifQ=="

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
		want   bool
	}{
		{
			name:   "empty when no cursor",
			cursor: "",
			want:   true,
		},
		{
			name:   "not empty when cursor exists",
			cursor: "AAHiB1lWHNABAAAAAAAACcYH3ufNgRzg91vJc",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := NewCursor()
			cursor.SetCursor(tt.cursor)

			assert.Equal(t, tt.want, cursor.IsEmpty())
		})
	}
}

func TestCursor_SetGetCursor(t *testing.T) {
	cursor := NewCursor()

	cursor.SetCursor("initial-cursor-value")
	assert.Equal(t, "initial-cursor-value", cursor.GetCursor())

	// Overwrite
	cursor.SetCursor("updated-cursor-value")
	assert.Equal(t, "updated-cursor-value", cursor.GetCursor())
}

func TestCursor_RoundTrip(t *testing.T) {
	// Test multiple round trips preserve data
	original := NewCursor()
	original.SetCursor("AAHiB1lWHNABAAAAAAAACcYH3ufNgRzg91vJcDropboxLongCursorValue")

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.Cursor, decoded.Cursor)
		original = decoded
	}
}
