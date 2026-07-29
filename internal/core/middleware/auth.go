package middleware

import (
	"net/http"
	"strings"
)

func Auth(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			token := ""
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = strings.TrimSpace(parts[1])
			} else {
				// Fallback : accepte le token brut s'il n'y a pas le préfixe "Bearer " (utile pour Swagger)
				token = strings.TrimSpace(authHeader)
			}

			if token != expectedToken {
				http.Error(w, "Invalid token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
