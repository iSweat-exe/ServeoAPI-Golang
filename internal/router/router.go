package router

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "serveoapi/docs" // Swagger generated docs
	"serveoapi/internal/core/database"
	"serveoapi/internal/core/middleware"
	"serveoapi/internal/modules/v2/apikeys"
	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/modules/v2/docker"
	"serveoapi/internal/modules/v2/files"
	"serveoapi/internal/modules/v2/mcp"
	"serveoapi/internal/modules/v2/metadata"
	"serveoapi/internal/modules/v2/ovh"
	"serveoapi/internal/modules/v2/system"
	"serveoapi/internal/modules/v2/templates"
	"serveoapi/internal/modules/v2/users"
)

func New() http.Handler {
	mux := http.NewServeMux()

	// Swagger UI (Public)
	// Go 1.22+ ServeMux handles trailing slash as a prefix match automatically
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Auth Middleware for protected routes
	authMiddleware := middleware.JWTAuth

	// Register Auth routes (Login public, Logout protected)
	auth.RegisterRoutes(mux, authMiddleware, database.DB)

	// Register Modules
	metadata.RegisterRoutes(mux, authMiddleware, database.DB)
	system.RegisterRoutes(mux, authMiddleware, database.DB)
	docker.RegisterRoutes(mux, authMiddleware, database.DB)
	files.RegisterRoutes(mux, authMiddleware, database.DB)
	users.RegisterRoutes(mux, authMiddleware, database.DB)
	templates.RegisterRoutes(mux, authMiddleware, database.DB)
	ovh.RegisterRoutes(mux, authMiddleware, database.DB)
	mcp.RegisterRoutes(mux, authMiddleware, database.DB)
	apikeys.RegisterRoutes(mux, authMiddleware, database.DB)

	// Serve AI Skills definitions (legacy path, redirects to or serves LLMs/skills.md if needed, but we expose the whole folder now)
	mux.Handle("GET /LLMs/", http.StripPrefix("/LLMs/", http.FileServer(http.Dir("LLMs"))))

	// Apply Global Middlewares (RateLimit, CORS, Logger)
	handler := middleware.RateLimit(mux)
	handler = middleware.CORS(handler)
	return middleware.Logger(handler)
}
