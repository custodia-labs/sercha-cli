package calendar

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("calendar: invalid cursor format")

// Cursor tracks Google Calendar sync state using per-calendar syncTokens.
type Cursor struct {
	// Version is the cursor format version for future compatibility.
	Version int `json:"v"`
	// SyncTokens maps calendar ID to its syncToken.
	// Each calendar has its own sync token for incremental sync.
	SyncTokens map[string]string `json:"sync_tokens"`
}

// NewCursor creates a new empty cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Version:    CursorVersion,
		SyncTokens: make(map[string]string),
	}
}

// Encode serialises the cursor to a base64 string for storage.
func (c *Cursor) Encode() string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeCursor deserializes a cursor from a base64 string.
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return NewCursor(), nil
	}

	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}

	// Version check for future migrations
	if cursor.Version > CursorVersion {
		return nil, ErrInvalidCursor
	}

	// Ensure map is initialised
	if cursor.SyncTokens == nil {
		cursor.SyncTokens = make(map[string]string)
	}

	return &cursor, nil
}

// IsEmpty returns true if the cursor has no sync state.
func (c *Cursor) IsEmpty() bool {
	return len(c.SyncTokens) == 0
}

// GetSyncToken returns the sync token for a calendar.
func (c *Cursor) GetSyncToken(calendarID string) string {
	return c.SyncTokens[calendarID]
}

// SetSyncToken sets the sync token for a calendar.
func (c *Cursor) SetSyncToken(calendarID, token string) {
	c.SyncTokens[calendarID] = token
}
