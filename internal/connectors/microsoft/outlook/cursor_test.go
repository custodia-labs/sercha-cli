package outlook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.DeltaLink)
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.SetDeltaLink("https://graph.microsoft.com/v1.0/me/mailFolders/inbox/messages/delta?$skiptoken=abc123")

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	// Should be valid base64
	assert.NotContains(t, encoded, " ")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.SetDeltaLink("https://graph.microsoft.com/v1.0/delta?token=xyz")

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.DeltaLink, decoded.GetDeltaLink())
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.DeltaLink)
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
	// {"v":999,"delta_link":"test"}
	futureVersionBase64 := "eyJ2Ijo5OTksImRlbHRhX2xpbmsiOiJ0ZXN0In0="

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name      string
		deltaLink string
		want      bool
	}{
		{
			name:      "empty when no delta link",
			deltaLink: "",
			want:      true,
		},
		{
			name:      "not empty when delta link exists",
			deltaLink: "https://graph.microsoft.com/v1.0/delta?token=abc",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := NewCursor()
			cursor.SetDeltaLink(tt.deltaLink)

			assert.Equal(t, tt.want, cursor.IsEmpty())
		})
	}
}

func TestCursor_SetGetDeltaLink(t *testing.T) {
	cursor := NewCursor()

	cursor.SetDeltaLink("https://test.com/delta")
	assert.Equal(t, "https://test.com/delta", cursor.GetDeltaLink())

	// Overwrite
	cursor.SetDeltaLink("https://updated.com/delta")
	assert.Equal(t, "https://updated.com/delta", cursor.GetDeltaLink())
}

func TestCursor_RoundTrip(t *testing.T) {
	// Test multiple round trips preserve data
	original := NewCursor()
	original.SetDeltaLink("https://graph.microsoft.com/v1.0/me/mailFolders/inbox/messages/delta?$skiptoken=very-long-token-1234567890")

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.DeltaLink, decoded.DeltaLink)
		original = decoded
	}
}
