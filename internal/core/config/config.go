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
	AllowedMountRoot string
	APIToken         string
}

func Load() *Config {
	cfg := &Config{
		Env:              "development",
		Port:             "8080",
		RateLimit:        10, // 10 requests per second default
		AllowedMountRoot: "/var/serveoapi/data/",
		APIToken:         "isweat_123", //! Default for dev, MUST change in prod
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
