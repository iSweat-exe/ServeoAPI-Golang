package middleware

import (
	"context"
	"net/http"
	"strings"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
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
		}

		if tokenString == "" {
			response.SendError(w, http.StatusUnauthorized, "Missing Authorization header or token parameter")
			return
		}

		userID, permissions, err := ValidateToken(tokenString)
		if err != nil {
			response.SendError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), contextkeys.UserPermissionsKey, permissions)
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
