package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.NotNil(t, cursor.DeltaLinks)
	assert.Empty(t, cursor.DeltaLinks)
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.SetDeltaLink("cal-123", "https://graph.microsoft.com/v1.0/delta?token=abc")

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	assert.NotContains(t, encoded, " ")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.SetDeltaLink("cal-123", "https://graph.microsoft.com/v1.0/delta?token=xyz")
	original.SetDeltaLink("cal-456", "https://graph.microsoft.com/v1.0/delta?token=abc")

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.GetDeltaLink("cal-123"), decoded.GetDeltaLink("cal-123"))
	assert.Equal(t, original.GetDeltaLink("cal-456"), decoded.GetDeltaLink("cal-456"))
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Empty(t, cursor.DeltaLinks)
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	cursor, err := DecodeCursor("not-valid-base64!!!")

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON
	cursor, err := DecodeCursor("bm90IGpzb24=")

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestDecodeCursor_FutureVersion(t *testing.T) {
	// {"v":999,"delta_links":{}}
	futureVersionBase64 := "eyJ2Ijo5OTksImRlbHRhX2xpbmtzIjp7fX0="

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name       string
		deltaLinks map[string]string
		want       bool
	}{
		{
			name:       "empty when no delta links",
			deltaLinks: map[string]string{},
			want:       true,
		},
		{
			name:       "not empty when delta link exists",
			deltaLinks: map[string]string{"cal-1": "link"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := NewCursor()
			for k, v := range tt.deltaLinks {
				cursor.SetDeltaLink(k, v)
			}

			assert.Equal(t, tt.want, cursor.IsEmpty())
		})
	}
}

func TestCursor_SetGetDeltaLink(t *testing.T) {
	cursor := NewCursor()

	cursor.SetDeltaLink("cal-1", "https://link1.com")
	assert.Equal(t, "https://link1.com", cursor.GetDeltaLink("cal-1"))

	cursor.SetDeltaLink("cal-2", "https://link2.com")
	assert.Equal(t, "https://link2.com", cursor.GetDeltaLink("cal-2"))

	// Overwrite
	cursor.SetDeltaLink("cal-1", "https://updated.com")
	assert.Equal(t, "https://updated.com", cursor.GetDeltaLink("cal-1"))

	// Non-existent
	assert.Empty(t, cursor.GetDeltaLink("cal-999"))
}

func TestCursor_HasDeltaLink(t *testing.T) {
	cursor := NewCursor()
	cursor.SetDeltaLink("cal-1", "https://link.com")

	assert.True(t, cursor.HasDeltaLink("cal-1"))
	assert.False(t, cursor.HasDeltaLink("cal-2"))
}

func TestCursor_RoundTrip(t *testing.T) {
	original := NewCursor()
	original.SetDeltaLink("cal-1", "https://graph.microsoft.com/v1.0/delta?token=1")
	original.SetDeltaLink("cal-2", "https://graph.microsoft.com/v1.0/delta?token=2")

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.DeltaLinks, decoded.DeltaLinks)
		original = decoded
	}
}
