package system

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
		"GET /v2/system/",
		authMiddleware(middleware.RequirePermission("system.read", http.HandlerFunc(h.GetSystem))),
	)
}
