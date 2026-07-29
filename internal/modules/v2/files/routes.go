package files

import (
	"net/http"

	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

// RegisterRoutes sets up the API endpoints for the file manager module
// RegisterRoutes sets up the API endpoints for the file manager module
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB) {
	h := &Handler{DB: db}
	// Require 'files.read' permission
	mux.Handle("GET /v2/files/{server}/list", authMiddleware(middleware.RequirePermission("files.read", http.HandlerFunc(h.ListFiles))))
	mux.Handle("GET /v2/files/{server}/read", authMiddleware(middleware.RequirePermission("files.read", http.HandlerFunc(h.ReadFile))))

	// Require 'files.write' permission
	mux.Handle("POST /v2/files/{server}/write", authMiddleware(middleware.RequirePermission("files.write", http.HandlerFunc(h.WriteFile))))
	mux.Handle("POST /v2/files/{server}/upload", authMiddleware(middleware.RequirePermission("files.write", http.HandlerFunc(h.UploadFile))))
	mux.Handle("DELETE /v2/files/{server}/delete", authMiddleware(middleware.RequirePermission("files.write", http.HandlerFunc(h.DeleteFile))))
}
