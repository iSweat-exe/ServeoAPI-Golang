package metadata

import (
	"net/http"
	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /v2/metadata/", authMiddleware(middleware.RequirePermission("metadata.read", http.HandlerFunc(GetMetadata))))
}
