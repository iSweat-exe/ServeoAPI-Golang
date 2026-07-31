package router

import (
	"net/http"

	"github.com/docker/docker/client"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "serveoapi/docs" // Documentation générée par Swagger
	"serveoapi/internal/core/config"
	"serveoapi/internal/core/database"
	"serveoapi/internal/core/middleware"
	"serveoapi/internal/modules/v2/apikeys"
	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/modules/v2/backups"
	"serveoapi/internal/modules/v2/docker"
	"serveoapi/internal/modules/v2/files"
	"serveoapi/internal/modules/v2/health"
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

	authMiddleware := middleware.JWTAuth

	// Go 1.22+ ServeMux gère automatiquement le trailing slash comme une correspondance de préfixe.
	// La documentation n'est exposée publiquement qu'en dehors de la production.
	var swagger http.Handler = httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"))
	llmDocs := http.StripPrefix("/LLMs/", http.FileServer(http.Dir("LLMs")))
	if cfg.IsProduction() {
		swagger = authMiddleware(swagger)
		llmDocs = authMiddleware(llmDocs)
	}
	mux.Handle("GET /swagger/", swagger)
	mux.Handle("GET /LLMs/", llmDocs)

	auth.RegisterRoutes(mux, authMiddleware, database.DB)

	metadata.RegisterRoutes(mux, authMiddleware, database.DB)
	system.RegisterRoutes(mux, authMiddleware, database.DB)
	docker.RegisterRoutes(mux, authMiddleware, database.DB, cfg, dockerCli)
	files.RegisterRoutes(mux, authMiddleware, database.DB, cfg, dockerCli)
	users.RegisterRoutes(mux, authMiddleware, database.DB)
	templates.RegisterRoutes(mux, authMiddleware, database.DB, cfg, dockerCli)
	ovh.RegisterRoutes(mux, authMiddleware, database.DB, cfg)
	mcp.RegisterRoutes(mux, authMiddleware, database.DB, cfg)
	apikeys.RegisterRoutes(mux, authMiddleware, database.DB)
	metrics.RegisterRoutes(mux, authMiddleware)
	backups.RegisterRoutes(mux, authMiddleware, cfg)
	health.RegisterRoutes(mux, database.DB, dockerCli)

	// Métriques Prometheus sécurisées avec JWT auth pour éviter les fuites
	mux.Handle("GET /prometheus", authMiddleware(promhttp.Handler()))

	handler := middleware.RateLimit(cfg)(mux)
	handler = middleware.CORS(cfg)(handler)
	handler = middleware.Metrics(handler)
	handler = middleware.Logger(handler)
	return middleware.Recover(handler)
}

// prometheusDocs est une fonction factice utilisée uniquement pour générer la documentation Swagger de la route /prometheus
// @Summary      Prometheus Metrics
// @Description  Exposes Prometheus metrics for the API (requires JWT auth)
// @Tags         system
// @Produce      text/plain
// @Security     ApiKeyAuth
// @Success      200  {string}  string "Prometheus metrics"
// @Router       /prometheus [get]
//
//nolint:unused
func prometheusDocs() {}
