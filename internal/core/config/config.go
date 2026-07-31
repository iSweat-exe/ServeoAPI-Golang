package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env              string
	Port             string
	RateLimit        int
	SQLitePath       string
	AllowedMountRoot string
	TemplatesPath    string
	AllowedOrigins   []string
	// TrustProxy autorise la lecture de X-Forwarded-For pour identifier le client.
	// À n'activer que derrière un reverse proxy de confiance.
	TrustProxy bool

	// OVH Cloud
	OvhEndpoint    string
	OvhAppKey      string
	OvhAppSecret   string
	OvhConsumerKey string
}

// DefaultAllowedOrigin est l'origine du serveur de développement Vite.
const DefaultAllowedOrigin = "http://localhost:5173"

func Load() *Config {
	cfg := &Config{
		Env:              "development",
		Port:             "8080",
		RateLimit:        10, // 10 requests per second default
		SQLitePath:       "serveo.db",
		AllowedMountRoot: "/var/serveoapi/data/",
		TemplatesPath:    getEnv("TEMPLATES_PATH", "./data/templates"),
		AllowedOrigins:   parseOrigins(getEnv("ALLOWED_ORIGINS", DefaultAllowedOrigin)),
		TrustProxy:       isTruthy(getEnv("TRUST_PROXY", "")),

		OvhEndpoint:    getEnv("OVH_ENDPOINT", ""), // e.g. "ovh-eu"
		OvhAppKey:      getEnv("OVH_APP_KEY", ""),
		OvhAppSecret:   getEnv("OVH_APP_SECRET", ""),
		OvhConsumerKey: getEnv("OVH_CONSUMER_KEY", ""),
	}

	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	if env := os.Getenv("ENV"); env != "" {
		cfg.Env = env
	}

	if rl := os.Getenv("API_RATE_LIMIT"); rl != "" {
		if val, err := strconv.Atoi(rl); err == nil {
			cfg.RateLimit = val
		}
	}

	if amr := os.Getenv("ALLOWED_MOUNT_ROOT"); amr != "" {
		// Ensure trailing slash for safety during prefix checks
		if !strings.HasSuffix(amr, "/") {
			amr += "/"
		}
		cfg.AllowedMountRoot = amr
	}

	if sqlitePath := os.Getenv("SQLITE_PATH"); sqlitePath != "" {
		cfg.SQLitePath = sqlitePath
	}

	return cfg
}

// IsProduction indique si l'API tourne dans un environnement de production.
func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.Env)
	return env == "production" || env == "prod"
}

// IsOriginAllowed effectue une comparaison stricte de l'origine avec la liste autorisée.
func (c *Config) IsOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// AllowsAnyOrigin indique que la configuration autorise explicitement toutes les origines.
func (c *Config) AllowsAnyOrigin() bool {
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" {
			return true
		}
	}
	return false
}

func isTruthy(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, strings.TrimSuffix(trimmed, "/"))
		}
	}
	if len(origins) == 0 {
		origins = append(origins, DefaultAllowedOrigin)
	}
	return origins
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
