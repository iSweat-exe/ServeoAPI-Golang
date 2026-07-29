package users

import (
	"net/http"
	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	// Profile (Me)
	mux.Handle("GET /v2/users/me", authMiddleware(http.HandlerFunc(GetMe)))
	mux.Handle("PUT /v2/users/me/password", authMiddleware(http.HandlerFunc(UpdateMePassword)))

	// Users Management (CRUD)
	mux.Handle("POST /v2/users/", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(CreateUser))))
	mux.Handle("GET /v2/users/", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(GetUsers))))
	mux.Handle("PATCH /v2/users/{id}", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(UpdateUser))))
	mux.Handle("DELETE /v2/users/{id}", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(DeleteUser))))
}
