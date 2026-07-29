package files

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
	"github.com/docker/docker/client"
	"gorm.io/gorm"
)

// RegisterRoutes configure les endpoints de l'API pour le module gestionnaire de fichiers
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB, cfg *config.Config, dockerCli *client.Client) {
	h := &Handler{DB: db, Config: cfg, DockerCli: dockerCli}
	// Nécessite la permission 'files.read'
	mux.Handle("GET /v2/files/{server}/list", authMiddleware(middleware.RequirePermission("files.read", http.HandlerFunc(h.ListFiles))))
	mux.Handle("GET /v2/files/{server}/read", authMiddleware(middleware.RequirePermission("files.read", http.HandlerFunc(h.ReadFile))))

	// Nécessite la permission 'files.write'
	mux.Handle("POST /v2/files/{server}/write", authMiddleware(middleware.RequirePermission("files.write", http.HandlerFunc(h.WriteFile))))
	mux.Handle("POST /v2/files/{server}/upload", authMiddleware(middleware.RequirePermission("files.write", http.HandlerFunc(h.UploadFile))))
	mux.Handle("DELETE /v2/files/{server}/delete", authMiddleware(middleware.RequirePermission("files.write", http.HandlerFunc(h.DeleteFile))))
}
