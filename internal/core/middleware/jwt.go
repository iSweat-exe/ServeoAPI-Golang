package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/modules/v2/auth"
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

		// Verify token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return auth.JwtSecretKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Extract permissions and ID and inject into context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			permissions, _ := claims["permissions"].(string)
			userIDFloat, _ := claims["sub"].(float64) // jwt.MapClaims parses numbers as float64
			userID := uint(userIDFloat)

			ctx := context.WithValue(r.Context(), contextkeys.UserPermissionsKey, permissions)
			ctx = context.WithValue(ctx, contextkeys.UserIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		http.Error(w, "Invalid token payload", http.StatusUnauthorized)
	})
}
