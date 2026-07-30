package mcp

import (
	"context"
	"errors"
	"net/http"

	"serveoapi/internal/core/config"

	"github.com/mark3labs/mcp-go/server"

	"serveoapi/internal/core/contextkeys"
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
func hasPermission(ctx context.Context, requiredPerm string) error {
	// Le middleware injecte les permissions dans le contexte
	// Nous extrayons les permissions du contexte.
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
