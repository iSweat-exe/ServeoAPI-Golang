package firewall

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
)

// RegisterRoutes configure les endpoints de gestion du firewall (UFW) de l'hôte.
func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	cfg *config.Config,
) {
	h := &Handler{Service: &Service{Config: cfg}}

	registerRoute := func(methodPath, perm string, handler http.HandlerFunc) {
		mux.Handle(methodPath, authMiddleware(middleware.RequirePermission(perm, handler)))
	}

	// Nécessite la permission 'firewall.read' pour consulter l'état et les règles
	registerRoute("GET /v2/firewall/status", "firewall.read", h.GetStatus)

	// Nécessite la permission 'firewall.write' pour toute modification
	registerRoute("POST /v2/firewall/rules", "firewall.write", h.AddRule)
	registerRoute("DELETE /v2/firewall/rules/{number}", "firewall.write", h.DeleteRule)
	registerRoute("POST /v2/firewall/enable", "firewall.write", h.EnableFirewall)
	registerRoute("POST /v2/firewall/disable", "firewall.write", h.DisableFirewall)
}
