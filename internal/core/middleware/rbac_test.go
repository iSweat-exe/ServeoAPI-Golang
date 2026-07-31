package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"serveoapi/internal/core/contextkeys"
)

func TestHasPermission(t *testing.T) {
	cases := []struct {
		name     string
		granted  string
		required string
		want     bool
	}{
		{"root wildcard", "*", "docker.containers.delete", true},
		{"exact match", "docker.containers.read", "docker.containers.read", true},
		{"prefix wildcard", "docker.containers.*", "docker.containers.delete", true},
		{
			"prefix wildcard does not cross scope",
			"docker.containers.*",
			"docker.images.delete",
			false,
		},
		{"prefix wildcard is not a substring match", "docker.*", "dockerhub.pull", false},
		{"comma separated list", "system.read, users.manage", "users.manage", true},
		{"missing permission", "system.read", "users.manage", false},
		{"empty permissions", "", "system.read", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasPermission(tc.granted, tc.required); got != tc.want {
				t.Fatalf(
					"HasPermission(%q, %q) = %v, want %v",
					tc.granted,
					tc.required,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestRequirePermission(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("rejects request without permissions in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v2/users", nil)

		RequirePermission("users.manage", next).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects insufficient permissions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v2/users", nil)
		req = req.WithContext(contextkeys.SetUserPermissions(req.Context(), "system.read"))

		RequirePermission("users.manage", next).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("allows sufficient permissions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v2/users", nil)
		req = req.WithContext(contextkeys.SetUserPermissions(req.Context(), "users.manage"))

		RequirePermission("users.manage", next).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
