package config

import "testing"

func TestParseOrigins(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"single", "https://a.tld", []string{"https://a.tld"}},
		{
			"list with spaces",
			"https://a.tld, https://b.tld",
			[]string{"https://a.tld", "https://b.tld"},
		},
		{"trailing slashes stripped", "https://a.tld/", []string{"https://a.tld"}},
		{"empty falls back to default", "   ", []string{DefaultAllowedOrigin}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseOrigins(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("parseOrigins(%q) = %v, want %v", tc.raw, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseOrigins(%q) = %v, want %v", tc.raw, got, tc.want)
				}
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	cfg := &Config{AllowedOrigins: []string{"https://panel.example.com"}}

	if !cfg.IsOriginAllowed("https://panel.example.com") {
		t.Fatal("exact origin should be allowed")
	}
	if cfg.IsOriginAllowed("https://panel.example.com.evil.tld") {
		t.Fatal("substring origin must be rejected")
	}
	if cfg.IsOriginAllowed("") {
		t.Fatal("empty origin must be rejected")
	}
}

func TestIsProduction(t *testing.T) {
	for _, env := range []string{"production", "PRODUCTION", "prod"} {
		if !(&Config{Env: env}).IsProduction() {
			t.Fatalf("Env=%q should be detected as production", env)
		}
	}
	for _, env := range []string{"development", "staging", ""} {
		if (&Config{Env: env}).IsProduction() {
			t.Fatalf("Env=%q should not be detected as production", env)
		}
	}
}

func TestAllowsAnyOrigin(t *testing.T) {
	if !(&Config{AllowedOrigins: []string{"https://a.tld", "*"}}).AllowsAnyOrigin() {
		t.Fatal("wildcard in list should allow any origin")
	}
	if (&Config{AllowedOrigins: []string{"https://a.tld"}}).AllowsAnyOrigin() {
		t.Fatal("no wildcard should not allow any origin")
	}
}
