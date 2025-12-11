package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	t.Run("nil search service returns error", func(t *testing.T) {
		ports := &Ports{}
		server, err := NewServer(ports)
		require.Error(t, err)
		assert.Nil(t, server)
		assert.ErrorIs(t, err, ErrMissingSearchService)
	})

	t.Run("valid ports creates server", func(t *testing.T) {
		ports := &Ports{
			Search: &mockSearchService{},
		}
		server, err := NewServer(ports)
		require.NoError(t, err)
		assert.NotNil(t, server)
	})
}

func TestPorts_Validate(t *testing.T) {
	t.Run("nil search service returns error", func(t *testing.T) {
		ports := &Ports{}
		err := ports.Validate()
		assert.ErrorIs(t, err, ErrMissingSearchService)
	})

	t.Run("search only is valid", func(t *testing.T) {
		ports := &Ports{
			Search: &mockSearchService{},
		}
		err := ports.Validate()
		assert.NoError(t, err)
	})

	t.Run("all ports is valid", func(t *testing.T) {
		ports := &Ports{
			Search:   &mockSearchService{},
			Source:   &mockSourceService{},
			Document: &mockDocumentService{},
		}
		err := ports.Validate()
		assert.NoError(t, err)
	})
}
