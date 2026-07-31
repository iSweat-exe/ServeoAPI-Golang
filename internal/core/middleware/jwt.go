package middleware

import (
	"net/http"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
	"serveoapi/internal/modules/v2/auth"
)

// JWTAuth middleware verifies the JWT token
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		tokenString := auth.BearerToken(r)
		if tokenString == "" {
			response.SendError(
				w,
				http.StatusUnauthorized,
				"Missing Authorization header or token parameter",
			)
			return
		}

		userID, permissions, err := ValidateToken(tokenString)
		if err != nil {
			response.SendError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := contextkeys.SetUserPermissions(r.Context(), permissions)
		ctx = contextkeys.SetUserID(ctx, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
