package system

import (
	"net/http"
	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /v2/system/", authMiddleware(middleware.RequirePermission("system.read", http.HandlerFunc(GetSystem))))
}
