package docker

import (
	"serveoapi/internal/core/response"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type DeployStackRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"` // Raw YAML content
}

// DeployStack godoc
// @Summary      Deploy a Docker Compose Stack
// @Description  Deploys a stack from a docker-compose.yml file. Strict security checks applied to volumes.
// @Tags         docker-compose
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body DeployStackRequest true "Stack details"
// @Success      201  {string}  string "Stack deployed successfully"
// @Failure      400,500 {string} string
// @Router       /v2/docker/compose/deploy [post]
func (h *Handler) DeployStack(w http.ResponseWriter, r *http.Request) {
	var req DeployStackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	if req.Name == "" || req.Content == "" {
		response.SendError(w, http.StatusBadRequest, "Name and Content are required")
		return
	}

	allowedRoot := h.Service.Config.AllowedMountRoot

	// 1. Vérification de sécurité: Analyser le YAML et inspecter les volumes
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.Content), &compose); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid YAML format: "+err.Error())
		return
	}

	if services, ok := compose["services"].(map[string]interface{}); ok {
		for srvName, srvData := range services {
			if srvMap, ok := srvData.(map[string]interface{}); ok {
				if volumes, ok := srvMap["volumes"].([]interface{}); ok {
					for _, vol := range volumes {
						volStr := ""
						switch v := vol.(type) {
						case string:
							volStr = v
						case map[string]interface{}:
							if src, ok := v["source"].(string); ok {
								volStr = src + ":..."
							}
						}

						if volStr != "" {
							parts := strings.SplitN(volStr, ":", 2)
							src := parts[0]

							// S'il s'agit d'un bind mount ou d'un chemin relatif
							if strings.Contains(src, "/") || strings.Contains(src, "\\") || strings.HasPrefix(src, ".") {
								// Refuser totalement les chemins relatifs par sécurité
								if strings.HasPrefix(src, ".") {
									response.SendError(w, http.StatusBadRequest, "Security Error: Relative paths in volumes are forbidden in service '"+srvName+"'")
									return
								}
								// Vérifier le chemin absolu
								if !strings.HasPrefix(src, allowedRoot) {
									response.SendError(w, http.StatusBadRequest, "Security Error: Bind mounts are restricted to "+allowedRoot+" in service '"+srvName+"'")
									return
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Écrire le YAML dans un fichier temporaire
	tmpDir := os.TempDir()
	fileName := filepath.Join(tmpDir, "serveo_stack_"+req.Name+"_"+time.Now().Format("20060102150405")+".yml")

	if err := os.WriteFile(fileName, []byte(req.Content), 0644); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to write temp compose file")
		return
	}

	// Remarque : Nous ne supprimons pas le fichier immédiatement pour permettre un éventuel 'docker compose down' ultérieur,
	// mais pour cet endpoint nous exécutons juste 'up -d' et laissons l'utilisateur gérer via CLI. Le fichier reste dans /tmp.

	// 3. Exécuter `docker compose up -d`
	cmd := exec.CommandContext(r.Context(), "docker", "compose", "-f", fileName, "-p", req.Name, "up", "-d")
	out, err := cmd.CombinedOutput()
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Docker Compose failed: "+string(out))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Stack deployed successfully",
		"output":  string(out),
	})
}

