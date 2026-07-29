package mcp

import (
	"context"
	"errors"
	"net/http"

	"github.com/mark3labs/mcp-go/server"

	"serveoapi/internal/core/contextkeys"
)

var (
	mcpServer  *server.MCPServer
	mcpHandler http.Handler
)

// InitServer initializes the MCP server and registers tools
func InitServer() {
	mcpServer = server.NewMCPServer("ServeoAPI-MCP", "1.0.0")

	// Register tools
	registerTools()

	// Wrap in a Streamable HTTP Server
	mcpHandler = server.NewStreamableHTTPServer(mcpServer)
}

// GetHandler returns the HTTP handler for the MCP routes
func GetHandler() http.Handler {
	return mcpHandler
}

// hasPermission is a helper to check if the current request context has the required RBAC permission.
func hasPermission(ctx context.Context, requiredPerm string) error {
	// Our middleware injects "permissions" into the request context (or "claims")
	// Let's assume we can fetch the user ID or claims from the context.
	// Since we are inside an MCP tool call, the HTTP request context is NOT directly available in the tool context by default in mcp-go unless we pass it.
	// Wait, the MCP tools signature is func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error).
	// If the StreamableHTTPServer passes the HTTP context to the tools, we can retrieve the claims.
	// We'll extract claims from context.
	permissionsObj := ctx.Value(contextkeys.UserPermissionsKey)
	if permissionsObj == nil {
		return errors.New("unauthorized: missing user permissions in context")
	}

	permissions, ok := permissionsObj.([]string)
	if !ok {
		return errors.New("internal server error: permissions type mismatch")
	}

	for _, p := range permissions {
		if p == requiredPerm || p == "*" {
			return nil
		}
	}
	return errors.New("forbidden: missing required permission '" + requiredPerm + "'")
}
