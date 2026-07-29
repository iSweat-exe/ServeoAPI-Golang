package middleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return h.Hijack()
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		start := time.Now()

		// Default status is 200 OK if WriteHeader is never called
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		log.Printf("--> %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		color := colorGreen
		switch {
		case rw.status >= 500:
			color = colorRed
		case rw.status >= 400:
			color = colorYellow
		case rw.status >= 300:
			color = colorCyan
		}

		log.Printf("<-- %s %s %s%d%s in %v", r.Method, r.URL.Path, color, rw.status, colorReset, duration)
	})
}
