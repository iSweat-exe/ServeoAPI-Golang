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
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := ""
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			tokenString = strings.TrimSpace(parts[1])
		} else {
			// Fallback (ex: pour Swagger)
			tokenString = strings.TrimSpace(authHeader)
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
