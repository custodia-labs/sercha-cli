package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.StartPageToken)
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.StartPageToken = "abc123pagetoken"

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	// Should be valid base64
	assert.NotContains(t, encoded, " ")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.StartPageToken = "my-page-token-12345"

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.StartPageToken, decoded.StartPageToken)
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.StartPageToken)
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
	// {"v":999,"start_page_token":"abc"}
	futureVersionBase64 := "eyJ2Ijo5OTksInN0YXJ0X3BhZ2VfdG9rZW4iOiJhYmMifQ=="

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name           string
		startPageToken string
		want           bool
	}{
		{
			name:           "empty when startPageToken is empty",
			startPageToken: "",
			want:           true,
		},
		{
			name:           "not empty when startPageToken is set",
			startPageToken: "some-token",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := NewCursor()
			cursor.StartPageToken = tt.startPageToken

			assert.Equal(t, tt.want, cursor.IsEmpty())
		})
	}
}

func TestCursor_RoundTrip(t *testing.T) {
	// Test multiple round trips preserve data
	original := NewCursor()
	original.StartPageToken = "very-long-page-token-1234567890abcdef"

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.StartPageToken, decoded.StartPageToken)
		original = decoded
	}
}
