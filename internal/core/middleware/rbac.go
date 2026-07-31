package middleware

import (
	"net/http"
	"strings"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
)

// HasPermission indique si la liste de permissions (séparées par des virgules) couvre
// la permission demandée. "*" accorde tous les droits et un suffixe ".*" agit comme
// un joker sur le préfixe (ex: "docker.containers.*").
func HasPermission(userPermissions, requiredPerm string) bool {
	for _, p := range strings.Split(userPermissions, ",") {
		p = strings.TrimSpace(p)
		if p == "*" || p == requiredPerm {
			return true
		}

		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(requiredPerm, prefix+".") {
				return true
			}
		}
	}

	return false
}

// RequirePermission checks if the authenticated user has the required permission
// or if they have the root permission "*".
func RequirePermission(requiredPerm string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		userPermsStr, ok := contextkeys.GetUserPermissions(r.Context())
		if !ok {
			response.SendError(w, http.StatusForbidden, "Access Denied: No permissions found")
			return
		}

		if !HasPermission(userPermsStr, requiredPerm) {
			response.SendError(
				w,
				http.StatusForbidden,
				"Access Denied: Missing permission '"+requiredPerm+"'",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
