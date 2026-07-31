package auth

import (
	"net/http"
	"strings"
)

// BearerToken extrait le jeton de l'en-tête Authorization.
// Le jeton brut sans préfixe "Bearer " est également accepté (pratique depuis Swagger).
func BearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if scheme, token, found := strings.Cut(authHeader, " "); found &&
		strings.EqualFold(scheme, "bearer") {
		return strings.TrimSpace(token)
	}

	return strings.TrimSpace(authHeader)
}
