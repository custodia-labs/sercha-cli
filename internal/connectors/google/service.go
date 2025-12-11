package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

// UserInfo contains the user's basic profile information from Google.
type UserInfo struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// NewGmailService creates a Gmail API service using the provided TokenSource.
func NewGmailService(ctx context.Context, ts oauth2.TokenSource) (*gmail.Service, error) {
	return gmail.NewService(ctx, option.WithTokenSource(ts))
}

// NewDriveService creates a Google Drive API service using the provided TokenSource.
func NewDriveService(ctx context.Context, ts oauth2.TokenSource) (*drive.Service, error) {
	return drive.NewService(ctx, option.WithTokenSource(ts))
}

// NewCalendarService creates a Google Calendar API service using the provided TokenSource.
func NewCalendarService(ctx context.Context, ts oauth2.TokenSource) (*calendar.Service, error) {
	return calendar.NewService(ctx, option.WithTokenSource(ts))
}

// GetUserInfo fetches the user's profile information using an access token.
// Returns the user's email address which serves as the account identifier.
func GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	return &userInfo, nil
}
