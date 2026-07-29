package templates

import (
	"serveoapi/internal/modules/v2/docker"
)

// TemplateInfo represents a full application or game server template
type TemplateInfo struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Logo        string                       `json:"logo"`
	Category    string                        `json:"category"` // e.g., "game", "database", "app", "lang"
	Variables   []TemplateVariable            `json:"variables"` // Variables to prompt the user for
	Docker      docker.CreateContainerRequest `json:"docker"`   // The exact payload to send to /v2/docker/containers/create
}

// TemplateVariable defines a configuration field that the user can set
type TemplateVariable struct {
	Name        string `json:"name"`        // The placeholder name used in the Docker config (e.g., "SERVER_NAME")
	Label       string `json:"label"`       // The UI label (e.g., "Server Name")
	Description string `json:"description"` // UI description
	Default     string `json:"default"`     // Default value
	Required    bool   `json:"required"`    // Is it mandatory?
}
