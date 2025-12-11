package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthCapability_Constants(t *testing.T) {
	assert.Equal(t, AuthCapability(0), AuthCapNone)
	assert.Equal(t, AuthCapability(1), AuthCapPAT)
	assert.Equal(t, AuthCapability(2), AuthCapOAuth)
}

func TestAuthCapability_SupportsPAT(t *testing.T) {
	tests := []struct {
		name     string
		cap      AuthCapability
		expected bool
	}{
		{"none", AuthCapNone, false},
		{"pat only", AuthCapPAT, true},
		{"oauth only", AuthCapOAuth, false},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.SupportsPAT())
		})
	}
}

func TestAuthCapability_SupportsOAuth(t *testing.T) {
	tests := []struct {
		name     string
		cap      AuthCapability
		expected bool
	}{
		{"none", AuthCapNone, false},
		{"pat only", AuthCapPAT, false},
		{"oauth only", AuthCapOAuth, true},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.SupportsOAuth())
		})
	}
}

func TestAuthCapability_SupportsMultipleMethods(t *testing.T) {
	tests := []struct {
		name     string
		cap      AuthCapability
		expected bool
	}{
		{"none", AuthCapNone, false},
		{"pat only", AuthCapPAT, false},
		{"oauth only", AuthCapOAuth, false},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.SupportsMultipleMethods())
		})
	}
}

func TestAuthCapability_RequiresAuth(t *testing.T) {
	tests := []struct {
		name     string
		cap      AuthCapability
		expected bool
	}{
		{"none", AuthCapNone, false},
		{"pat only", AuthCapPAT, true},
		{"oauth only", AuthCapOAuth, true},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.RequiresAuth())
		})
	}
}

func TestAuthCapability_SupportedMethods(t *testing.T) {
	tests := []struct {
		name     string
		cap      AuthCapability
		expected []AuthMethod
	}{
		{"none", AuthCapNone, nil},
		{"pat only", AuthCapPAT, []AuthMethod{AuthMethodPAT}},
		{"oauth only", AuthCapOAuth, []AuthMethod{AuthMethodOAuth}},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, []AuthMethod{AuthMethodPAT, AuthMethodOAuth}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methods := tt.cap.SupportedMethods()
			if tt.expected == nil {
				assert.Empty(t, methods)
			} else {
				assert.Equal(t, tt.expected, methods)
			}
		})
	}
}

func TestAuthCapability_String(t *testing.T) {
	tests := []struct {
		name     string
		cap      AuthCapability
		expected string
	}{
		{"none", AuthCapNone, "none"},
		{"pat only", AuthCapPAT, "pat"},
		{"oauth only", AuthCapOAuth, "oauth"},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, "pat,oauth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cap.String())
		})
	}
}

func TestConnectorType_RequiresAuth(t *testing.T) {
	tests := []struct {
		name       string
		capability AuthCapability
		expected   bool
	}{
		{"no auth", AuthCapNone, false},
		{"pat auth", AuthCapPAT, true},
		{"oauth auth", AuthCapOAuth, true},
		{"pat and oauth", AuthCapPAT | AuthCapOAuth, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := &ConnectorType{
				ID:             "test",
				AuthCapability: tt.capability,
			}
			assert.Equal(t, tt.expected, connector.RequiresAuth())
		})
	}
}
