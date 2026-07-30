package testutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	"serveoapi/internal/core/contextkeys"
)

// NewAuthenticatedRequest creates a new HTTP request with an authenticated user context.
func NewAuthenticatedRequest(
	method, url string,
	body io.Reader,
	userID uint,
	permissions []string,
) *http.Request {
	req := httptest.NewRequest(method, url, body)

	ctx := context.WithValue(req.Context(), contextkeys.UserIDKey, userID)
	if permissions != nil {
		ctx = context.WithValue(ctx, contextkeys.UserPermissionsKey, permissions)
	}

	return req.WithContext(ctx)
}
