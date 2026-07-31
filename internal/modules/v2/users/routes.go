package users

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

	mux.Handle("GET /v2/users/me", authMiddleware(http.HandlerFunc(h.GetMe)))
	mux.Handle("PUT /v2/users/me/password", authMiddleware(http.HandlerFunc(h.UpdateMePassword)))

	registerRoute := func(methodPath string, handler http.HandlerFunc) {
		mux.Handle(
			methodPath,
			authMiddleware(middleware.RequirePermission("users.manage", handler)),
		)
	}

	registerRoute("POST /v2/users/", h.CreateUser)
	registerRoute("GET /v2/users/", h.GetUsers)
	registerRoute("PATCH /v2/users/{id}", h.UpdateUser)
	registerRoute("PUT /v2/users/{id}/password", h.UpdateUserPassword)
	registerRoute("DELETE /v2/users/{id}", h.DeleteUser)
}
