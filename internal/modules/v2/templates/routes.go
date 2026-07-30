package templates

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
	"serveoapi/internal/modules/v2/docker"

	"github.com/docker/docker/client"
	"gorm.io/gorm"
)

// RegisterRoutes configure les endpoints de l'API pour le module templates
func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	db *gorm.DB,
	cfg *config.Config,
	dockerCli *client.Client,
) {
	dockerService := &docker.DockerService{
		DockerCli: dockerCli,
		Config:    cfg,
	}
	h := &Handler{DB: db, Config: cfg, DockerService: dockerService}
	// Nécessite la permission 'templates.read'
	mux.Handle("GET /v2/templates/", authMiddleware(
		middleware.RequirePermission("templates.read", http.HandlerFunc(h.GetTemplates))))
	mux.Handle("GET /v2/templates/{id}", authMiddleware(
		middleware.RequirePermission("templates.read", http.HandlerFunc(h.GetTemplate))))

	// Endpoint de déploiement (sécurisé, nécessite droit d'écriture sur les conteneurs)
	mux.Handle("POST /v2/templates/{id}/deploy", authMiddleware(
		middleware.RequirePermission(
			"docker.containers.write",
			http.HandlerFunc(h.DeployTemplate),
		),
	))
}
