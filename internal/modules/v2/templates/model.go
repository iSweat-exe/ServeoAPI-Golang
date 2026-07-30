package templates

import (
	"serveoapi/internal/modules/v2/docker"
)

// TemplateInfo représente un modèle complet d'application ou de serveur de jeu
type TemplateInfo struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Logo        string                        `json:"logo"`
	Category    string                        `json:"category"`  // ex: "game", "database", "app", "lang"
	Variables   []TemplateVariable            `json:"variables"` // Variables à demander à l'utilisateur
	Docker      docker.CreateContainerRequest `json:"docker"`    // Le payload exact à envoyer à /v2/docker/containers/create
}

// TemplateVariable définit un champ de configuration que l'utilisateur peut définir
type TemplateVariable struct {
	Name        string `json:"name"`        // Le nom du placeholder utilisé dans la config Docker (ex: "SERVER_NAME")
	Label       string `json:"label"`       // Le label pour l'UI (ex: "Server Name")
	Description string `json:"description"` // Description pour l'UI
	Default     string `json:"default"`     // Valeur par défaut
	Required    bool   `json:"required"`    // Est-ce obligatoire ?
}

// DeployTemplateRequest représente la requête du frontend pour déployer un serveur
type DeployTemplateRequest struct {
	Variables map[string]string `json:"variables"` // Valeurs des variables (ex: {"EULA": "TRUE"})
}
