package middleware

import (
	"net/http"

	"serveoapi/internal/core/config"
)

// CORS est un middleware qui gère les requêtes Cross-Origin Resource Sharing.
// Les origines sont comparées strictement à la liste configurée via ALLOWED_ORIGINS.
func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.Header().Add("Vary", "Origin")

			origin := r.Header.Get("Origin")
			switch {
			case cfg.AllowsAnyOrigin():
				w.Header().Set("Access-Control-Allow-Origin", "*")
			case cfg.IsOriginAllowed(origin):
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().
				Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")

			// Réponse immédiate pour les requêtes preflight OPTIONS du navigateur
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
