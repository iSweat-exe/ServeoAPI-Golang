package apikeys

import (
	"net/http"
)

// RegisterRoutes registers the API Keys routes to the mux.
func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /v2/apikeys/create", authMiddleware(http.HandlerFunc(CreateApiKey)))
	mux.Handle("GET /v2/apikeys/", authMiddleware(http.HandlerFunc(ListApiKeys)))
	mux.Handle("DELETE /v2/apikeys/{id}", authMiddleware(http.HandlerFunc(RevokeApiKey)))
}
