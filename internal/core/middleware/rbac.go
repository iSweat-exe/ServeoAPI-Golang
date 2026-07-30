package middleware

import (
	"net/http"
	"strings"

	"serveoapi/internal/core/contextkeys"
)

// RequirePermission checks if the authenticated user has the required permission
// or if they have the root permission "*".
func RequirePermission(requiredPerm string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		permissionsObj := r.Context().Value(contextkeys.UserPermissionsKey)
		if permissionsObj == nil {
			http.Error(w, "Access Denied: No permissions found", http.StatusForbidden)
			return
		}

		userPermsStr, ok := permissionsObj.(string)
		if !ok {
			http.Error(w, "Access Denied: Invalid permissions format", http.StatusForbidden)
			return
		}

		// Split comma-separated permissions
		perms := strings.Split(userPermsStr, ",")
		hasAccess := false

		for _, p := range perms {
			p = strings.TrimSpace(p)
			if p == "*" || p == requiredPerm {
				hasAccess = true
				break
			}

			// Optional: support wildcards like "docker.containers.*"
			if strings.HasSuffix(p, ".*") {
				prefix := strings.TrimSuffix(p, ".*")
				if strings.HasPrefix(requiredPerm, prefix+".") {
					hasAccess = true
					break
				}
			}
		}

		if !hasAccess {
			http.Error(
				w,
				"Access Denied: Missing permission '"+requiredPerm+"'",
				http.StatusForbidden,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
