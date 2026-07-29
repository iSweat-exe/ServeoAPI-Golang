package docker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// GetVolumes godoc
// @Summary      List Docker Volumes
// @Description  Returns a list of all volumes
// @Tags         docker-volumes
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   VolumeInfo
// @Router       /v2/docker/volumes/ [get]
func GetVolumes(w http.ResponseWriter, r *http.Request) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cli.Close()

	volumes, err := cli.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
func DeleteVolume(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	force := r.URL.Query().Get("force") == "true"

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cli.Close()

	err = cli.VolumeRemove(context.Background(), name, force)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
// @Tags         volumes
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateVolumeRequest true "Volume details"
// @Success      201  {object}  VolumeInfo
// @Failure      400,500 {string} string
// @Router       /v2/docker/volumes/ [post]
func CreateVolume(w http.ResponseWriter, r *http.Request) {
	var req CreateVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Error connecting to docker", http.StatusInternalServerError)
		return
	}
	defer cli.Close()

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
