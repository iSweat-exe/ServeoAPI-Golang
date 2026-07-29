package users

import (
	"net/http"
	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB) {
	h := &Handler{DB: db}

	// Profile (Me)
	mux.Handle("GET /v2/users/me", authMiddleware(http.HandlerFunc(h.GetMe)))
	mux.Handle("PUT /v2/users/me/password", authMiddleware(http.HandlerFunc(h.UpdateMePassword)))

	// Users Management (CRUD)
	mux.Handle("POST /v2/users/", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(h.CreateUser))))
	mux.Handle("GET /v2/users/", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(h.GetUsers))))
	mux.Handle("PATCH /v2/users/{id}", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(h.UpdateUser))))
	mux.Handle("DELETE /v2/users/{id}", authMiddleware(middleware.RequirePermission("users.manage", http.HandlerFunc(h.DeleteUser))))
}
