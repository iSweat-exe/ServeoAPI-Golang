package templates

import (
	"net/http"

	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

// RegisterRoutes sets up the API endpoints for the templates module
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB) {
	h := &Handler{DB: db}
	// Require 'templates.read' permission
	mux.Handle("GET /v2/templates/", authMiddleware(middleware.RequirePermission("templates.read", http.HandlerFunc(h.GetTemplates))))
	mux.Handle("GET /v2/templates/{id}", authMiddleware(middleware.RequirePermission("templates.read", http.HandlerFunc(h.GetTemplate))))
}
