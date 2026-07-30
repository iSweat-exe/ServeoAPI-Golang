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
	// Désactiver le buffering NGINX/Proxy pour le SSE
	w.Header().Set("X-Accel-Buffering", "no")
}

// SendSSEEvent envoie un évènement formaté SSE au client et flush le buffer
func SendSSEEvent(
	w http.ResponseWriter,
	data string,
) {
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
