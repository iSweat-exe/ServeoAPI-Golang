package templates

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

// RegisterRoutes configure les endpoints de l'API pour le module templates
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB, cfg *config.Config) {
	h := &Handler{DB: db, Config: cfg}
	// Nécessite la permission 'templates.read'
	mux.Handle("GET /v2/templates/", authMiddleware(middleware.RequirePermission("templates.read", http.HandlerFunc(h.GetTemplates))))
	mux.Handle("GET /v2/templates/{id}", authMiddleware(middleware.RequirePermission("templates.read", http.HandlerFunc(h.GetTemplate))))
}
