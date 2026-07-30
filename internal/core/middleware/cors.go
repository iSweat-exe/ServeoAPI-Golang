package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORS est un middleware qui gère les requêtes Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:5173"
		}

		origin := r.Header.Get("Origin")
		if origin != "" && strings.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if allowedOrigins != "*" {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")

		// Réponse immédiate pour les requêtes preflight OPTIONS du navigateur
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
