package templates

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/response"

	"gorm.io/gorm"
)

type Handler struct {
	DB     *gorm.DB
	Config *config.Config
}

// ensureTemplatesDir vérifie si le dossier existe, et sinon, le crée et charge les modèles par défaut.
func (h *Handler) ensureTemplatesDir() error {
	if _, err := os.Stat(h.Config.TemplatesPath); os.IsNotExist(err) {
		if err := os.MkdirAll(h.Config.TemplatesPath, 0o755); err != nil {
			return err
		}
		// Écrire quelques valeurs par défaut (Minecraft, Rust, Python, etc.)
		writeDefaultTemplates(h.Config.TemplatesPath)
	}
	return nil
}

// GetTemplates godoc
// @Summary      List Templates
// @Description  Returns a list of all available application and game templates
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   TemplateInfo
// @Router       /v2/templates/ [get]
func (h *Handler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureTemplatesDir(); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to load templates directory")
		return
	}

	files, err := os.ReadDir(h.Config.TemplatesPath)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var templates []TemplateInfo
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(h.Config.TemplatesPath, f.Name()))
			if err != nil {
				continue
			}
			var tpl TemplateInfo
			if err := json.Unmarshal(data, &tpl); err == nil {
				// Utiliser le nom de fichier sans l'extension comme ID s'il n'est pas fourni
				if tpl.ID == "" {
					tpl.ID = strings.TrimSuffix(f.Name(), ".json")
				}
				templates = append(templates, tpl)
			}
		}
	}

	response.SendJSON(w, http.StatusOK, templates)
}

// GetTemplate godoc
// @Summary      Get Template by ID
// @Description  Returns the full configuration of a specific template
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Template ID"
// @Success      200  {object}  TemplateInfo
// @Failure      404  {string}  string "Template not found"
// @Router       /v2/templates/{id} [get]
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Empêcher la traversée de chemin
	cleanPath := filepath.Clean(id + ".json")
	if strings.Contains(cleanPath, "..") || strings.Contains(cleanPath, "/") ||
		strings.Contains(cleanPath, "\\") {
		response.SendError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	fullPath := filepath.Join(h.Config.TemplatesPath, cleanPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		response.SendError(w, http.StatusNotFound, "Template not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

var defaultTemplates = map[string]string{
	"minecraft.json": `{
			"id": "minecraft",
			"name": "Minecraft Server",
			"description": "High performance Minecraft server (supports Paper, Forge, Fabric, etc.)",
			"logo": "https://upload.wikimedia.org/wikipedia/en/5/51/Minecraft_cover.png",
			"category": "game",
			"variables": [
				{"name": "EULA", "label": "Accept EULA", "description": "You must accept the EULA to run the server (TRUE or FALSE)", "default": "TRUE", "required": true},
				{"name": "SERVER_MEMORY", "label": "RAM (Memory)", "description": "Amount of RAM (e.g. 2G, 4G)", "default": "2G", "required": true},
				{"name": "MC_VERSION", "label": "Minecraft Version", "description": "e.g. LATEST, 1.20.4", "default": "LATEST", "required": true},
				{"name": "MC_TYPE", "label": "Server Type", "description": "PAPER, FORGE, FABRIC, QUILT, VANILLA", "default": "PAPER", "required": true}
			],
			"docker": {
				"image": "itzg/minecraft-server",
				"name": "minecraft-{{id}}",
				"env": [
					"EULA={{EULA}}",
					"TYPE={{MC_TYPE}}",
					"VERSION={{MC_VERSION}}",
					"MEMORY={{SERVER_MEMORY}}",
					"CREATE_CONSOLE_IN_PIPE=true"
				],
				"ports": {
					"25565": "25565"
				},
				"bind_mounts": [
					"/var/serveoapi/data/minecraft-{{id}}:/data"
				]
			}
		}`,
	"rust.json": `{
			"id": "rust",
			"name": "Rust Server",
			"description": "Official Rust Dedicated Server.",
			"logo": "https://upload.wikimedia.org/wikipedia/en/5/51/Rust_Logo.png",
			"category": "game",
			"variables": [
				{"name": "SERVER_NAME", "label": "Server Name", "description": "Public name of the server", "default": "ServeoAPI Rust Server", "required": true},
				{"name": "MAX_PLAYERS", "label": "Max Players", "description": "Maximum allowed players", "default": "50", "required": true},
				{"name": "RUST_IDENTITY", "label": "Server Identity", "description": "Save folder name", "default": "serveo", "required": true}
			],
			"docker": {
				"image": "didstopia/rust-server",
				"name": "rust-{{id}}",
				"env": [
					"RUST_SERVER_NAME={{SERVER_NAME}}",
					"RUST_SERVER_MAXPLAYERS={{MAX_PLAYERS}}",
					"RUST_SERVER_IDENTITY={{RUST_IDENTITY}}"
				],
				"ports": {
					"28015": "28015",
					"28016": "28016"
				},
				"bind_mounts": [
					"/var/serveoapi/data/rust-{{id}}:/steamcmd/rust"
				]
			}
		}`,
	"csgo.json": `{
			"id": "csgo",
			"name": "CS:GO Server",
			"description": "Counter-Strike: Global Offensive Dedicated Server.",
			"logo": "https://upload.wikimedia.org/wikipedia/en/c/ce/CS_GO_Cover_Art.png",
			"category": "game",
			"variables": [
				{"name": "GSLT_TOKEN", "label": "Steam GSLT Token", "description": "Required to list the server publicly", "default": "", "required": true},
				{"name": "TICKRATE", "label": "Tickrate", "description": "64 or 128", "default": "128", "required": true}
			],
			"docker": {
				"image": "joaopaulo/csgo-server",
				"name": "csgo-{{id}}",
				"env": [
					"SRCDS_TOKEN={{GSLT_TOKEN}}",
					"CSGO_TICKRATE={{TICKRATE}}"
				],
				"ports": {
					"27015": "27015"
				},
				"bind_mounts": [
					"/var/serveoapi/data/csgo-{{id}}:/home/steam/csgo-dedicated"
				]
			}
		}`,
	"palworld.json": `{
			"id": "palworld",
			"name": "Palworld Server",
			"description": "Palworld Dedicated Server.",
			"logo": "https://upload.wikimedia.org/wikipedia/en/3/3d/Palworld_logo.png",
			"category": "game",
			"variables": [
				{"name": "MAX_PLAYERS", "label": "Max Players", "description": "Maximum allowed players (1-32)", "default": "32", "required": true},
				{"name": "SERVER_PASSWORD", "label": "Server Password", "description": "Leave blank for public", "default": "", "required": false},
				{"name": "ADMIN_PASSWORD", "label": "Admin Password", "description": "Password for admin commands", "default": "admin", "required": true}
			],
			"docker": {
				"image": "thijsvanloef/palworld-server-docker",
				"name": "palworld-{{id}}",
				"env": [
					"PUID=1000",
					"PGID=1000",
					"PORT=8211",
					"PLAYERS={{MAX_PLAYERS}}",
					"SERVER_PASSWORD={{SERVER_PASSWORD}}",
					"ADMIN_PASSWORD={{ADMIN_PASSWORD}}",
					"MULTITHREADING=true"
				],
				"ports": {
					"8211": "8211/udp",
					"27015": "27015/udp"
				},
				"bind_mounts": [
					"/var/serveoapi/data/palworld-{{id}}:/palworld"
				]
			}
		}`,
	"nodejs.json": `{
			"id": "nodejs",
			"name": "Node.js",
			"description": "Node.js execution environment.",
			"logo": "https://upload.wikimedia.org/wikipedia/commons/d/d9/Node.js_logo.svg",
			"category": "lang",
			"variables": [
				{"name": "NODE_VERSION", "label": "Node Version", "description": "Docker tag for node (e.g., 18, 20, latest)", "default": "18", "required": true}
			],
			"docker": {
				"image": "node:{{NODE_VERSION}}",
				"name": "node-{{id}}",
				"cmd": ["tail", "-f", "/dev/null"],
				"bind_mounts": [
					"/var/serveoapi/data/node-{{id}}:/app"
				]
			}
		}`,
	"python.json": `{
			"id": "python",
			"name": "Python",
			"description": "Python execution environment.",
			"logo": "https://upload.wikimedia.org/wikipedia/commons/c/c3/Python-logo-notext.svg",
			"category": "lang",
			"variables": [
				{"name": "PYTHON_VERSION", "label": "Python Version", "description": "Docker tag for python (e.g., 3.11, 3.12)", "default": "3.11", "required": true}
			],
			"docker": {
				"image": "python:{{PYTHON_VERSION}}",
				"name": "python-{{id}}",
				"cmd": ["tail", "-f", "/dev/null"],
				"bind_mounts": [
					"/var/serveoapi/data/python-{{id}}:/app"
				]
			}
		}`,
	"rustlang.json": `{
			"id": "rustlang",
			"name": "Rust (Lang)",
			"description": "Rust programming language environment.",
			"logo": "https://upload.wikimedia.org/wikipedia/commons/d/d5/Rust_programming_language_black_logo.svg",
			"category": "lang",
			"variables": [
				{"name": "RUST_VERSION", "label": "Rust Version", "description": "Docker tag for rust (e.g., latest, 1.75)", "default": "latest", "required": true}
			],
			"docker": {
				"image": "rust:{{RUST_VERSION}}",
				"name": "rustlang-{{id}}",
				"cmd": ["tail", "-f", "/dev/null"],
				"bind_mounts": [
					"/var/serveoapi/data/rustlang-{{id}}:/app"
				]
			}
		}`,
}

// Fonction utilitaire pour écrire les modèles par défaut
func writeDefaultTemplates(path string) {
	for name, content := range defaultTemplates {
		_ = os.WriteFile(filepath.Join(path, name), []byte(content), 0o644)
	}
}
