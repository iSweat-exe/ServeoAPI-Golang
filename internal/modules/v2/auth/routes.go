package auth

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /v2/auth/login", http.HandlerFunc(Login))
	mux.Handle("POST /v2/auth/logout", authMiddleware(http.HandlerFunc(Logout)))
}
