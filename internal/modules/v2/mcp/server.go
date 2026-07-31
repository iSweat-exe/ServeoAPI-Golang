package mcp

import (
	"context"
	"errors"
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/middleware"

	"github.com/mark3labs/mcp-go/server"
)

var (
	mcpServer  *server.MCPServer
	mcpHandler http.Handler
)

// InitServer initialise le serveur MCP et enregistre les outils
func InitServer(cfg *config.Config) {
	mcpServer = server.NewMCPServer("ServeoAPI-MCP", "1.0.0")

	// Enregistrer les outils
	registerTools(cfg)

	// Envelopper dans un serveur HTTP Streamable
	mcpHandler = server.NewStreamableHTTPServer(mcpServer)
}

// GetHandler retourne le handler HTTP pour les routes MCP
func GetHandler() http.Handler {
	return mcpHandler
}

// hasPermission vérifie si le contexte de la requête contient la permission RBAC requise.
// Le middleware JWT injecte les permissions sous forme de chaîne séparée par des virgules.
func hasPermission(ctx context.Context, requiredPerm string) error {
	permissions, ok := contextkeys.GetUserPermissions(ctx)
	if !ok {
		return errors.New("unauthorized: missing user permissions in context")
	}

	if !middleware.HasPermission(permissions, requiredPerm) {
		return errors.New("forbidden: missing required permission '" + requiredPerm + "'")
	}

	return nil
}
