package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"serveoapi/internal/core/config"
)

func corsHandler(origins []string) http.Handler {
	cfg := &config.Config{AllowedOrigins: origins}
	return CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestCORSAllowedOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v2/system/", nil)
	req.Header.Set("Origin", "https://panel.example.com")

	corsHandler([]string{"https://panel.example.com"}).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://panel.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the exact origin", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want %q", got, "Origin")
	}
}

// L'ancienne implémentation utilisait strings.Contains : une origine attaquante
// contenant une origine autorisée en sous-chaîne était acceptée.
func TestCORSRejectsSubstringOrigin(t *testing.T) {
	for _, origin := range []string{
		"https://panel.example.com.evil.tld",
		"https://evil.tld?x=https://panel.example.com",
		"example.com",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v2/system/", nil)
		req.Header.Set("Origin", origin)

		corsHandler([]string{"https://panel.example.com"}).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("origin %q was allowed (header = %q)", origin, got)
		}
	}
}

func TestCORSPreflightShortCircuits(t *testing.T) {
	called := false
	cfg := &config.Config{AllowedOrigins: []string{"https://panel.example.com"}}
	handler := CORS(cfg)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/v2/system/", nil)
	req.Header.Set("Origin", "https://panel.example.com")
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("preflight request reached the downstream handler")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCORSWildcard(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v2/system/", nil)
	req.Header.Set("Origin", "https://anything.tld")

	corsHandler([]string{"*"}).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}
