package metrics

import (
	"net/http"

	"serveoapi/internal/core/middleware"
)

// RegisterRoutes enregistre les routes d'historique des métriques
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	h := &Handler{}

	registerRoute := func(methodPath, perm string, handler http.HandlerFunc) {
		mux.Handle(methodPath, authMiddleware(middleware.RequirePermission(perm, handler)))
	}

	registerRoute("GET /v2/metrics/history/containers/{id}", "metrics.read", h.GetContainerMetricsHistory)
	registerRoute("GET /v2/metrics/history/system", "metrics.read", h.GetSystemMetricsHistory)
}
