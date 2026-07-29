package mcp

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

// RegisterRoutes enregistre les routes MCP
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB, cfg *config.Config) {
	// Initialise le serveur MCP (singleton)
	InitServer(cfg)

	// Nous exposons Streamable HTTP sur une seule route
	// Le serveur Streamable HTTP gère à la fois les requêtes GET et POST.
	mcpHandler := middleware.RequirePermission("mcp.use", GetHandler())
	mux.Handle("/v2/mcp/", authMiddleware(mcpHandler))
	mux.Handle("/v2/mcp", authMiddleware(mcpHandler))
}
