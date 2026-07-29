package mcp

import (
	"net/http"

	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

// RegisterRoutes registers the MCP routes to the mux.
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB) {
	// Initialize the singleton MCP server
	InitServer()

	// We expose Streamable HTTP on a single route
	// Streamable HTTP server handles both GET and POST requests.
	mcpHandler := middleware.RequirePermission("mcp.use", GetHandler())
	mux.Handle("/v2/mcp/", authMiddleware(mcpHandler))
	mux.Handle("/v2/mcp", authMiddleware(mcpHandler))
}
