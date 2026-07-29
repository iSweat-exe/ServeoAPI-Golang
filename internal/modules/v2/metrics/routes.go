package metrics

import (
	"net/http"

	"serveoapi/internal/core/middleware"
)

// RegisterRoutes enregistre les routes d'historique des métriques
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	h := &Handler{}

	mux.Handle("GET /v2/metrics/history/containers/{id}", authMiddleware(middleware.RequirePermission("metrics.read", http.HandlerFunc(h.GetContainerMetricsHistory))))
	mux.Handle("GET /v2/metrics/history/system", authMiddleware(middleware.RequirePermission("metrics.read", http.HandlerFunc(h.GetSystemMetricsHistory))))
}
