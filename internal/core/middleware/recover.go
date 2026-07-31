package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"serveoapi/internal/core/response"
)

// Recover intercepte les paniques d'un handler pour éviter l'arrêt du processus et
// renvoie une erreur 500 au client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			// La connexion est déjà reprise par le client (WebSocket, hijack) : rien à écrire.
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			slog.Error("Panique interceptée dans un handler HTTP",
				slog.Any("panic", rec),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("stack", string(debug.Stack())),
			)

			response.SendError(w, http.StatusInternalServerError, "Internal server error")
		}()

		next.ServeHTTP(w, r)
	})
}
