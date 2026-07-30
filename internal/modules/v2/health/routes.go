package health

import (
	"net/http"

	"github.com/docker/docker/client"
	"gorm.io/gorm"
)

// RegisterRoutes registers the healthcheck endpoints
func RegisterRoutes(
	mux *http.ServeMux,
	db *gorm.DB,
	dockerCli *client.Client,
) {
	h := &Handler{DB: db, DockerCli: dockerCli}

	// Healthcheck route is open (no auth)
	mux.Handle("GET /health", http.HandlerFunc(h.Check))
}
