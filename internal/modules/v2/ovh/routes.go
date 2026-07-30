package ovh

import (
	"net/http"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"

	"gorm.io/gorm"
)

// RegisterRoutes configure les endpoints de l'API pour le module ovh
func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	db *gorm.DB,
	cfg *config.Config,
) {
	h := &Handler{DB: db}
	// Initialisation du client Singleton
	if err := InitClient(cfg); err != nil {
		// Ne pas crasher en cas d'erreur, le module sera simplement désactivé
		return
	}

	// Nécessite la permission 'ovh.read' pour consulter les serveurs
	mux.Handle(
		"GET /v2/ovh/me",
		authMiddleware(middleware.RequirePermission("ovh.read", http.HandlerFunc(h.GetMe))),
	)
	mux.Handle(
		"GET /v2/ovh/dedicated/server",
		authMiddleware(
			middleware.RequirePermission("ovh.read", http.HandlerFunc(h.ListDedicatedServers)),
		),
	)

	// Nécessite la permission 'ovh.write' pour les actions destructrices
	mux.Handle(
		"POST /v2/ovh/dedicated/server/{serviceName}/reboot",
		authMiddleware(
			middleware.RequirePermission("ovh.write", http.HandlerFunc(h.HardRebootServer)),
		),
	)
}
