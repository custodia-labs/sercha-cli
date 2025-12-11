package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
	Long:  `Commands for the Model Context Protocol (MCP) server integration.`,
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the Model Context Protocol server for AI assistant integration.

By default, the server communicates over stdio using JSON-RPC and can be
used with Claude Desktop and other MCP-compatible AI assistants.

Use --port to start an HTTP server instead, which enables:
  - Testing with MCP Inspector web UI
  - Remote access via HTTP

Examples:
  # Stdio mode (default, for Claude Desktop)
  sercha mcp serve

  # HTTP mode (for MCP Inspector, remote access)
  sercha mcp serve --port 8080

Claude Desktop configuration (claude_desktop_config.json):
  {
    "mcpServers": {
      "sercha": {
        "command": "/path/to/sercha",
        "args": ["mcp", "serve"]
      }
    }
  }`,
	RunE: runMCPServe,
}

func init() {
	mcpServeCmd.Flags().IntP("port", "p", 0, "HTTP port (0 = use stdio)")
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServe(cmd *cobra.Command, _ []string) error {
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return fmt.Errorf("getting port flag: %w", err)
	}

	ports := &mcp.Ports{
		Search:   searchService,
		Source:   sourceService,
		Document: documentService,
	}

	server, err := mcp.NewServer(ports)
	if err != nil {
		return err
	}

	if port > 0 {
		addr := fmt.Sprintf(":%d", port)
		fmt.Fprintf(cmd.OutOrStdout(), "MCP server listening on http://localhost%s\n", addr)
		return server.RunHTTP(cmd.Context(), addr)
	}

	return server.Run(cmd.Context())
}
