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
	APIToken         string
	
	// OVH Cloud
	OvhEndpoint    string
	OvhAppKey      string
	OvhAppSecret   string
	OvhConsumerKey string
}

func Load() *Config {
	cfg := &Config{
		Env:              "development",
		Port:             "8080",
		RateLimit:        10, // 10 requests per second default
		SQLitePath:       "serveo.db",
		AllowedMountRoot: "/var/serveoapi/data/",
		TemplatesPath:    getEnv("TEMPLATES_PATH", "./data/templates"),
		APIToken:         getEnv("API_TOKEN", DefaultAPIToken), // Bounded to version.go for CI/CD injection

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

	if apiToken := os.Getenv("API_TOKEN"); apiToken != "" {
		cfg.APIToken = apiToken
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

