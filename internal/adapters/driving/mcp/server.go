package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is the MCP server version.
const Version = "0.1.0"

// Server is the MCP server for Sercha.
type Server struct {
	ports  *Ports
	server *mcp.Server
}

// NewServer creates a new MCP server with the given ports.
func NewServer(ports *Ports) (*Server, error) {
	if err := ports.Validate(); err != nil {
		return nil, fmt.Errorf("validating ports: %w", err)
	}

	impl := &mcp.Implementation{
		Name:    "sercha",
		Version: Version,
	}

	s := &Server{
		ports:  ports,
		server: mcp.NewServer(impl, nil),
	}

	s.registerTools()
	s.registerResources()

	return s, nil
}

// Run starts the MCP server over stdio.
// It blocks until the context is cancelled or an error occurs.
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

// RunHTTP starts the MCP server over HTTP on the specified address.
// It blocks until the context is cancelled or an error occurs.
func (s *Server) RunHTTP(ctx context.Context, addr string) error {
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.server
	}, nil)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown when context is cancelled
	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background()) //nolint:errcheck
	}()

	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
