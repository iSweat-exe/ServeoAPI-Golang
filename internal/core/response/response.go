package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// SendJSON sends a JSON response with the given status code and payload
func SendJSON(
	w http.ResponseWriter,
	status int,
	payload interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			slog.Error("Failed to encode JSON response", "error", err)
		}
	}
}

// ErrorResponse represents a standard error response payload.
type ErrorResponse struct {
	Error string `json:"error"`
}

// SendError sends a standardized JSON error response.
func SendError(w http.ResponseWriter, status int, message string) {
	SendJSON(w, status, ErrorResponse{Error: message})
}
