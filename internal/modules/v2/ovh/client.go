package ovh

import (
	"log/slog"

	"github.com/ovh/go-ovh/ovh"

	"serveoapi/internal/core/config"
)

var apiClient *ovh.Client

// InitClient initialise le client de l'API OVH (singleton)
func InitClient(cfg *config.Config) error {
	// Sans identifiants, on ignore l'initialisation (le module sera désactivé)
	if cfg.OvhEndpoint == "" || cfg.OvhAppKey == "" {
		slog.Info("OVH Module disabled (missing credentials in config)")
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
	slog.Info("OVH Module initialized successfully")
	return nil
}

// GetClient retourne le client OVH initialisé, ou nil si non configuré
func GetClient() *ovh.Client {
	return apiClient
}
