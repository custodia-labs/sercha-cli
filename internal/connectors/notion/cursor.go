package notion

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor stores sync state for incremental sync.
// Since Notion's Search API doesn't report deletions, we track all known
// page/database IDs to detect when items are removed.
type Cursor struct {
	Version      int                  `json:"v"`
	LastSyncTime time.Time            `json:"last_sync_time"`
	PageStates   map[string]PageState `json:"page_states"`
}

// PageState tracks the state of a page or database for change detection.
type PageState struct {
	LastEditedTime time.Time `json:"last_edited"`
	IsDatabase     bool      `json:"is_db,omitempty"`
}

// NewCursor creates a new empty cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Version:    CursorVersion,
		PageStates: make(map[string]PageState),
	}
}

// Encode serialises the cursor to a base64 string.
func (c *Cursor) Encode() string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeCursor deserialises a cursor from a base64 string.
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

	if cursor.Version > CursorVersion {
		return nil, ErrInvalidCursor
	}

	// Ensure map is initialised
	if cursor.PageStates == nil {
		cursor.PageStates = make(map[string]PageState)
	}

	return &cursor, nil
}

// IsEmpty returns true if the cursor has no tracked pages.
func (c *Cursor) IsEmpty() bool {
	return len(c.PageStates) == 0 && c.LastSyncTime.IsZero()
}

// SetPageState updates or adds a page state.
func (c *Cursor) SetPageState(id string, lastEdited time.Time, isDatabase bool) {
	c.PageStates[id] = PageState{
		LastEditedTime: lastEdited,
		IsDatabase:     isDatabase,
	}
}

// GetPageState returns the state for a page, or nil if not found.
func (c *Cursor) GetPageState(id string) *PageState {
	if state, ok := c.PageStates[id]; ok {
		return &state
	}
	return nil
}

// RemovePageState removes a page from tracking.
func (c *Cursor) RemovePageState(id string) {
	delete(c.PageStates, id)
}

// SetLastSyncTime updates the last sync timestamp.
func (c *Cursor) SetLastSyncTime(t time.Time) {
	c.LastSyncTime = t
}

// GetLastSyncTime returns the last sync timestamp.
func (c *Cursor) GetLastSyncTime() time.Time {
	return c.LastSyncTime
}

// GetAllPageIDs returns all tracked page IDs.
func (c *Cursor) GetAllPageIDs() []string {
	ids := make([]string, 0, len(c.PageStates))
	for id := range c.PageStates {
		ids = append(ids, id)
	}
	return ids
}

// IsPageUpdated checks if a page has been updated since last sync.
func (c *Cursor) IsPageUpdated(id string, lastEdited time.Time) bool {
	state := c.GetPageState(id)
	if state == nil {
		// New page
		return true
	}
	return lastEdited.After(state.LastEditedTime)
}
