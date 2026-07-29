package docker

import (
	"serveoapi/internal/core/response"
	"context"
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/volume"
)

// GetVolumes godoc
// @Summary      List Docker Volumes
// @Description  Returns a list of all volumes
// @Tags         docker-volumes
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   VolumeInfo
// @Router       /v2/docker/volumes/ [get]
func (h *Handler) GetVolumes(w http.ResponseWriter, r *http.Request) {
	cli := h.Service.DockerCli

	volumes, err := cli.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var resp []VolumeInfo
	for _, v := range volumes.Volumes {
		resp = append(resp, VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
		})
	}

	if resp == nil {
		resp = []VolumeInfo{}
	}

	response.SendJSON(w, http.StatusOK, resp)
}

// DeleteVolume godoc
// @Summary      Delete a Docker Volume
// @Description  Removes a volume by name
// @Tags         docker-volumes
// @Security     ApiKeyAuth
// @Param        name  path      string  true  "Volume Name"
// @Param        force query     bool    false "Force remove"
// @Success      204
// @Router       /v2/docker/volumes/{name} [delete]
func (h *Handler) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	force := r.URL.Query().Get("force") == "true"

	cli := h.Service.DockerCli

	err := cli.VolumeRemove(context.Background(), name, force)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CreateVolumeRequest struct {
	Name   string            `json:"name"`
	Driver string            `json:"driver"`
	Labels map[string]string `json:"labels"`
}

// CreateVolume godoc
// @Summary      Create a Docker Volume
// @Description  Create a new Docker volume manually
// @Tags         docker-volumes
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateVolumeRequest true "Volume details"
// @Success      201  {object}  VolumeInfo
// @Failure      400,500 {string} string
// @Router       /v2/docker/volumes/ [post]
func (h *Handler) CreateVolume(w http.ResponseWriter, r *http.Request) {
	var req CreateVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	cli := h.Service.DockerCli

	if req.Driver == "" {
		req.Driver = "local"
	}

	options := volume.CreateOptions{
		Name:   req.Name,
		Driver: req.Driver,
		Labels: req.Labels,
	}

	vol, err := cli.VolumeCreate(r.Context(), options)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(VolumeInfo{
		Name:       vol.Name,
		Driver:     vol.Driver,
		Mountpoint: vol.Mountpoint,
	})
}

