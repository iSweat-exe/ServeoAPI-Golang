package auth

import (
	"net/http"
	"gorm.io/gorm"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB) {
	h := &Handler{DB: db}
	mux.Handle("POST /v2/auth/login", http.HandlerFunc(h.Login))
	mux.Handle("POST /v2/auth/logout", authMiddleware(http.HandlerFunc(h.Logout)))
}
