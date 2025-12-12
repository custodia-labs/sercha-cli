package microsoft

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserInfo_GetUserEmail(t *testing.T) {
	tests := []struct {
		name     string
		userInfo UserInfo
		expected string
	}{
		{
			name: "mail is set",
			userInfo: UserInfo{
				Mail:              "user@example.com",
				UserPrincipalName: "user@tenant.onmicrosoft.com",
			},
			expected: "user@example.com",
		},
		{
			name: "mail is empty, fallback to UPN",
			userInfo: UserInfo{
				Mail:              "",
				UserPrincipalName: "user@tenant.onmicrosoft.com",
			},
			expected: "user@tenant.onmicrosoft.com",
		},
		{
			name: "both empty",
			userInfo: UserInfo{
				Mail:              "",
				UserPrincipalName: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.userInfo.GetUserEmail()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGraphBaseURL(t *testing.T) {
	assert.Equal(t, "https://graph.microsoft.com/v1.0", graphBaseURL)
}
