package ovh

import (
	"net/http"

	"serveoapi/internal/core/middleware"
)

// RegisterRoutes sets up the API endpoints for the ovh module
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	// Initialize Client Singleton
	if err := InitClient(); err != nil {
		// Log the error but don't crash, the module will just be disabled
		return
	}

	// Require 'ovh.read' permission for viewing servers
	mux.Handle("GET /v2/ovh/me", authMiddleware(middleware.RequirePermission("ovh.read", http.HandlerFunc(GetMe))))
	mux.Handle("GET /v2/ovh/dedicated/server", authMiddleware(middleware.RequirePermission("ovh.read", http.HandlerFunc(ListDedicatedServers))))

	// Require 'ovh.write' permission for destructive actions
	mux.Handle("POST /v2/ovh/dedicated/server/{serviceName}/reboot", authMiddleware(middleware.RequirePermission("ovh.write", http.HandlerFunc(HardRebootServer))))
}
