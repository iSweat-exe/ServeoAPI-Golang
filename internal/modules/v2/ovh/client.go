package ovh

import (
	"log"

	"github.com/ovh/go-ovh/ovh"

	"serveoapi/internal/core/config"
)

var apiClient *ovh.Client

// InitClient initializes the OVH API client singleton
func InitClient() error {
	cfg := config.Load()

	// If no credentials, we skip initialization (module will be disabled)
	if cfg.OvhEndpoint == "" || cfg.OvhAppKey == "" {
		log.Println("⚠️  OVH Module disabled (missing credentials in config)")
		return nil
	}

	client, err := ovh.NewClient(
		cfg.OvhEndpoint,
		cfg.OvhAppKey,
		cfg.OvhAppSecret,
		cfg.OvhConsumerKey,
	)

	if err != nil {
		return err
	}

	apiClient = client
	log.Println("✅ OVH Module initialized successfully")
	return nil
}

// GetClient returns the initialized OVH client, or nil if not configured
func GetClient() *ovh.Client {
	return apiClient
}
