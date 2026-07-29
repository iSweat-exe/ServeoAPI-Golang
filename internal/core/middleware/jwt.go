package middleware

import (
	"context"
	"net/http"
	"strings"

	"serveoapi/internal/core/contextkeys"
)

// JWTAuth middleware verifies the JWT token
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenString := ""

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = strings.TrimSpace(parts[1])
			} else {
				// Fallback (ex: pour Swagger)
				tokenString = strings.TrimSpace(authHeader)
			}
		} else {
			// Fallback pour les IDEs et clients (comme Cursor/Antigravity) qui ne supportent pas les Headers HTTP
			tokenString = r.URL.Query().Get("token")
		}

		if tokenString == "" {
			http.Error(w, "Missing Authorization header or token parameter", http.StatusUnauthorized)
			return
		}

		userID, permissions, err := ValidateToken(tokenString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextkeys.UserPermissionsKey, permissions)
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
