package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Microsoft Graph API base URL.
const graphBaseURL = "https://graph.microsoft.com/v1.0"

// UserInfo contains the user's basic profile information from Microsoft Graph.
type UserInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

// GetUserInfo fetches the user's profile information using an access token.
// Returns the user's email address which serves as the account identifier.
// Falls back to userPrincipalName if mail is not set.
func GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	url := graphBaseURL + "/me?$select=id,displayName,mail,userPrincipalName"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d: %w",
			resp.StatusCode, WrapError(resp.StatusCode))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	return &userInfo, nil
}

// GetUserEmail returns the user's email address.
// Falls back to userPrincipalName if mail is not set.
func (u *UserInfo) GetUserEmail() string {
	if u.Mail != "" {
		return u.Mail
	}
	return u.UserPrincipalName
}
