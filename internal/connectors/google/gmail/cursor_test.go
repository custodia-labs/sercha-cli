package gmail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	cursor := NewCursor()

	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Equal(t, uint64(0), cursor.HistoryID)
}

func TestCursor_Encode(t *testing.T) {
	cursor := NewCursor()
	cursor.HistoryID = 12345678

	encoded := cursor.Encode()

	assert.NotEmpty(t, encoded)
	// Should be valid base64
	assert.NotContains(t, encoded, " ")
}

func TestDecodeCursor_Valid(t *testing.T) {
	original := NewCursor()
	original.HistoryID = 98765432

	encoded := original.Encode()
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.HistoryID, decoded.HistoryID)
}

func TestDecodeCursor_Empty(t *testing.T) {
	cursor, err := DecodeCursor("")

	require.NoError(t, err)
	assert.NotNil(t, cursor)
	assert.Equal(t, CursorVersion, cursor.Version)
	assert.Equal(t, uint64(0), cursor.HistoryID)
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
	// {"v":999,"history_id":123}
	futureVersionBase64 := "eyJ2Ijo5OTksImhpc3RvcnlfaWQiOjEyM30="

	cursor, err := DecodeCursor(futureVersionBase64)

	assert.Nil(t, cursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestCursor_IsEmpty(t *testing.T) {
	tests := []struct {
		name      string
		historyID uint64
		want      bool
	}{
		{
			name:      "empty when historyID is 0",
			historyID: 0,
			want:      true,
		},
		{
			name:      "not empty when historyID is set",
			historyID: 12345,
			want:      false,
		},
		{
			name:      "not empty with any non-zero value",
			historyID: 1,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := NewCursor()
			cursor.HistoryID = tt.historyID

			assert.Equal(t, tt.want, cursor.IsEmpty())
		})
	}
}

func TestCursor_RoundTrip(t *testing.T) {
	// Test multiple round trips preserve data
	original := NewCursor()
	original.HistoryID = 1234567890123

	for i := 0; i < 3; i++ {
		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, original.HistoryID, decoded.HistoryID)
		original = decoded
	}
}
