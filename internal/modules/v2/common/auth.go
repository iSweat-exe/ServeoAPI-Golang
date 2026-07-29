package common

import (
	"net/http"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
)

// GetUserIDOrUnauthorized extracts the user ID from the request context
// and sends a 401 Unauthorized response if not found. Returns ID and success boolean.
func GetUserIDOrUnauthorized(
	w http.ResponseWriter,
	r *http.Request,
) (uint, bool) {
	userID, ok := contextkeys.GetUserID(r.Context())
	if !ok {
		response.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return 0, false
	}
	return userID, true
}
