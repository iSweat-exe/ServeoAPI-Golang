package files

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"

	"github.com/docker/docker/client"
	"gorm.io/gorm"
)

// RegisterRoutes configure les endpoints de l'API pour le module gestionnaire de fichiers
func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	db *gorm.DB,
	cfg *config.Config,
	dockerCli *client.Client,
) {
	h := &Handler{DB: db, Config: cfg, DockerCli: dockerCli}
	registerRoute := func(methodPath, perm string, handler http.HandlerFunc) {
		mux.Handle(methodPath, authMiddleware(middleware.RequirePermission(perm, handler)))
	}

	// Nécessite la permission 'files.read'
	registerRoute("GET /v2/files/{server}/list", "files.read", h.ListFiles)
	registerRoute("GET /v2/files/{server}/read", "files.read", h.ReadFile)

	// Nécessite la permission 'files.write'
	registerRoute("POST /v2/files/{server}/write", "files.write", h.WriteFile)
	registerRoute("POST /v2/files/{server}/upload", "files.write", h.UploadFile)
	registerRoute("POST /v2/files/{server}/mkdir", "files.write", h.CreateDirectory)
	registerRoute("POST /v2/files/{server}/create", "files.write", h.CreateFile)
	registerRoute("DELETE /v2/files/{server}/delete", "files.write", h.DeleteFile)
}
