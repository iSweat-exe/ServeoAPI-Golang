package router

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/docker/docker/client"

	_ "serveoapi/docs" // Documentation générée par Swagger
	"serveoapi/internal/core/config"
	"serveoapi/internal/core/database"
	"serveoapi/internal/core/middleware"
	"serveoapi/internal/modules/v2/apikeys"
	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/modules/v2/backups"
	"serveoapi/internal/modules/v2/docker"
	"serveoapi/internal/modules/v2/files"
	"serveoapi/internal/modules/v2/mcp"
	"serveoapi/internal/modules/v2/metadata"
	"serveoapi/internal/modules/v2/metrics"
	"serveoapi/internal/modules/v2/ovh"
	"serveoapi/internal/modules/v2/system"
	"serveoapi/internal/modules/v2/templates"
	"serveoapi/internal/modules/v2/users"
)

func New(cfg *config.Config, dockerCli *client.Client) http.Handler {
	mux := http.NewServeMux()

	// Go 1.22+ ServeMux gère automatiquement le trailing slash comme une correspondance de préfixe
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	authMiddleware := middleware.JWTAuth

	auth.RegisterRoutes(mux, authMiddleware, database.DB)

	metadata.RegisterRoutes(mux, authMiddleware, database.DB)
	system.RegisterRoutes(mux, authMiddleware, database.DB)
	docker.RegisterRoutes(mux, authMiddleware, database.DB, cfg, dockerCli)
	files.RegisterRoutes(mux, authMiddleware, database.DB, cfg, dockerCli)
	users.RegisterRoutes(mux, authMiddleware, database.DB)
	templates.RegisterRoutes(mux, authMiddleware, database.DB, cfg)
	ovh.RegisterRoutes(mux, authMiddleware, database.DB, cfg)
	mcp.RegisterRoutes(mux, authMiddleware, database.DB, cfg)
	apikeys.RegisterRoutes(mux, authMiddleware, database.DB)
	metrics.RegisterRoutes(mux, authMiddleware)
	backups.RegisterRoutes(mux, authMiddleware, cfg)

	mux.Handle("GET /LLMs/", http.StripPrefix("/LLMs/", http.FileServer(http.Dir("LLMs"))))

	handler := middleware.RateLimit(mux)
	handler = middleware.CORS(handler)
	return middleware.Logger(handler)
}
