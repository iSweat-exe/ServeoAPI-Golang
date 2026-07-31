package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"serveoapi/internal/core/contextkeys"
)

// NewAuthenticatedRequest creates a new HTTP request with an authenticated user context.
// Les permissions sont stockées comme en production : une chaîne séparée par des virgules.
func NewAuthenticatedRequest(
	method, url string,
	body io.Reader,
	userID uint,
	permissions []string,
) *http.Request {
	req := httptest.NewRequest(method, url, body)

	ctx := contextkeys.SetUserID(req.Context(), userID)
	if permissions != nil {
		ctx = contextkeys.SetUserPermissions(ctx, strings.Join(permissions, ","))
	}

	return req.WithContext(ctx)
}
