package server

import (
	"github.com/hwuu/quorum-cc/internal/config"
	"github.com/hwuu/quorum-cc/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set at build time via ldflags.
var Version = "dev"

// New creates a new MCP server with the quorum_review tool registered.
func New(cfg *config.Config) *server.MCPServer {
	s := server.NewMCPServer(
		"quorum-cc",
		Version,
		server.WithToolCapabilities(true),
	)

	s.AddTool(tools.NewReviewTool(), tools.HandleReview(cfg))

	return s
}

// ServeStdio starts the MCP server in stdio mode.
func ServeStdio(cfg *config.Config) error {
	s := New(cfg)
	return server.ServeStdio(s)
}
