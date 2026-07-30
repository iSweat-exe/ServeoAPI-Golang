package apikeys

import (
	"net/http"

	"gorm.io/gorm"
)

// RegisterRoutes registers the API Keys routes to the mux.
func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	db *gorm.DB,
) {
	h := &Handler{DB: db}
	mux.Handle("POST /v2/apikeys/create", authMiddleware(http.HandlerFunc(h.CreateApiKey)))
	mux.Handle("GET /v2/apikeys/", authMiddleware(http.HandlerFunc(h.ListApiKeys)))
	mux.Handle("DELETE /v2/apikeys/{id}", authMiddleware(http.HandlerFunc(h.RevokeApiKey)))
}
