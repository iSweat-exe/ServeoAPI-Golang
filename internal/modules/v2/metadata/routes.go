package metadata

import (
	"net/http"

	"gorm.io/gorm"

	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	db *gorm.DB,
) {
	h := &Handler{DB: db}
	mux.Handle(
		"GET /v2/metadata/",
		authMiddleware(
			middleware.RequirePermission("metadata.read", http.HandlerFunc(h.GetMetadata)),
		),
	)
}
