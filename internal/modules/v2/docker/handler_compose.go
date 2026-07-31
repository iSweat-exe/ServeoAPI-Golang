package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"serveoapi/internal/core/response"

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
func (h *Handler) DeployStack(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	if err := validateComposeVolumes(compose, allowedRoot); err != nil {
		response.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.executeDeploy(w, r, req)
}

func validateComposeVolumes(
	compose map[string]interface{},
	allowedRoot string,
) error {
	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		return nil
	}

	for srvName, srvData := range services {
		srvMap, ok := srvData.(map[string]interface{})
		if !ok {
			continue
		}

		volumes, ok := srvMap["volumes"].([]interface{})
		if !ok {
			continue
		}

		for _, vol := range volumes {
			if err := validateVolume(vol, srvName, allowedRoot); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateVolume(
	vol interface{},
	srvName, allowedRoot string,
) error {
	volStr := ""
	switch v := vol.(type) {
	case string:
		volStr = v
	case map[string]interface{}:
		if src, ok := v["source"].(string); ok {
			volStr = src + ":..."
		}
	}

	if volStr == "" {
		return nil
	}

	parts := strings.SplitN(volStr, ":", 2)
	src := parts[0]

	// Volumes nommés Docker (sans séparateur de chemin) : pas des bind mounts.
	if !strings.ContainsAny(src, `/\`) && !strings.HasPrefix(src, ".") {
		return nil
	}

	normalized := path.Clean(filepath.ToSlash(src))
	if strings.HasPrefix(src, ".") || !path.IsAbs(normalized) {
		return fmt.Errorf(
			"Security Error: Relative paths in volumes are forbidden in service '%s'",
			srvName,
		)
	}

	// Même normalisation que CreateContainer : bloque les traversées "../".
	if !isBindSourceAllowed(src, allowedRoot) {
		return fmt.Errorf(
			"Security Error: Bind mounts are restricted to %s in service '%s'",
			allowedRoot,
			srvName,
		)
	}
	return nil
}

// executeDeploy continues the logic of DeployStack
func (h *Handler) executeDeploy(
	w http.ResponseWriter,
	r *http.Request,
	req DeployStackRequest,
) {
	// 2. Écrire le YAML dans un fichier temporaire
	tmpDir := os.TempDir()
	fileName := filepath.Join(
		tmpDir,
		"serveo_stack_"+req.Name+"_"+time.Now().Format("20060102150405")+".yml",
	)

	if err := os.WriteFile(fileName, []byte(req.Content), 0o644); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to write temp compose file")
		return
	}

	// Remarque : Nous ne supprimons pas le fichier immédiatement pour permettre un éventuel 'docker compose down' ultérieur,
	// mais pour cet endpoint nous exécutons juste 'up -d' et laissons l'utilisateur gérer via CLI. Le fichier reste dans /tmp.

	// 3. Exécuter `docker compose up -d`
	cmd := exec.CommandContext(
		r.Context(),
		"docker",
		"compose",
		"-f",
		fileName,
		"-p",
		req.Name,
		"up",
		"-d",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Docker Compose failed: "+string(out),
		)
		return
	}

	response.SendJSON(w, http.StatusCreated, map[string]string{
		"message": "Stack deployed successfully",
		"output":  string(out),
	})
}
