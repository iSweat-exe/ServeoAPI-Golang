package backups

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, cfg *config.Config) {
	h := &Handler{Config: cfg}

	mux.Handle("GET /v2/backups/{server}", authMiddleware(middleware.RequirePermission("backups.read", http.HandlerFunc(h.ListBackups))))
	mux.Handle("POST /v2/backups/{server}", authMiddleware(middleware.RequirePermission("backups.write", http.HandlerFunc(h.CreateBackup))))
	mux.Handle("POST /v2/backups/{server}/restore", authMiddleware(middleware.RequirePermission("backups.write", http.HandlerFunc(h.RestoreBackup))))
}
