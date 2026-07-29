package stream

import (
	"fmt"
	"net/http"
)

// SetupSSEHeaders configure la réponse HTTP pour le Server-Sent Events (SSE)
func SetupSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Allow CORS if needed, or rely on global middleware
}

// SendSSEEvent envoie un évènement formaté SSE au client et flush le buffer
func SendSSEEvent(w http.ResponseWriter, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
