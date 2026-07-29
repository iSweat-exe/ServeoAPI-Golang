package router

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "serveoapi/docs" // Swagger generated docs
	"serveoapi/internal/core/middleware"
	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/modules/v2/docker"
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
	auth.RegisterRoutes(mux, authMiddleware)

	// Register Modules
	metadata.RegisterRoutes(mux, authMiddleware)
	system.RegisterRoutes(mux, authMiddleware)
	docker.RegisterRoutes(mux, authMiddleware)
	users.RegisterRoutes(mux, authMiddleware)
	templates.RegisterRoutes(mux, authMiddleware)
	ovh.RegisterRoutes(mux, authMiddleware)

	// Apply Global Middlewares (RateLimit, CORS, Logger)
	handler := middleware.RateLimit(mux)
	handler = middleware.CORS(handler)
	return middleware.Logger(handler)
}
