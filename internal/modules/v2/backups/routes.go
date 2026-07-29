package backups

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	cfg *config.Config,
) {
	h := &Handler{Config: cfg}

	registerRoute := func(methodPath, perm string, handler http.HandlerFunc) {
		mux.Handle(methodPath, authMiddleware(middleware.RequirePermission(perm, handler)))
	}

	registerRoute("GET /v2/backups/{server}", "backups.read", h.ListBackups)
	registerRoute("POST /v2/backups/{server}", "backups.write", h.CreateBackup)
	registerRoute("POST /v2/backups/{server}/restore", "backups.write", h.RestoreBackup)
}
