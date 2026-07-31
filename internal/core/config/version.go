package config

// These variables will be injected by the compiler via ldflags:
// go build -ldflags "-X serveoapi/internal/core/config.AppVersion=v2.0.0 -X serveoapi/internal/core/config.CommitHash=abc1234"
var (
	AppVersion = "dev"
	CommitHash = "none"

	// UpdatePublicKey est la clé publique minisign utilisée pour vérifier les binaires
	// téléchargés par la commande "update". Sans elle, aucune mise à jour n'est appliquée.
	UpdatePublicKey = ""
)
