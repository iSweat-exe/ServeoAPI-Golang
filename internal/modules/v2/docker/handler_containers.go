package docker

import (
	"encoding/json"
	"net/http"
	"strings"

	"serveoapi/internal/core/response"
)

type Handler struct {
	Service *DockerService
}

// GetContainers godoc
// @Summary      List Docker Containers
// @Description  Returns a list of all docker containers
// @Tags         docker-containers
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   ContainerInfo
// @Router       /v2/docker/containers/ [get]
func (h *Handler) GetContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := h.Service.ListContainers(r.Context())
	if err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Failed to list containers: "+err.Error(),
		)
		return
	}

	response.SendJSON(w, http.StatusOK, containers)
}

// InspectContainer godoc
// @Summary      Inspect a Docker Container
// @Description  Returns detailed information about a container
// @Tags         docker-containers
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Container ID"
// @Success      200  {object}  interface{}
// @Router       /v2/docker/containers/{id} [get]
func (h *Handler) InspectContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	info, err := h.Service.InspectContainer(r.Context(), id)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.SendJSON(w, http.StatusOK, info)
}

// ActionContainer godoc
// @Summary      Perform action on a container (start, stop, restart)
// @Description  Action must be one of: start, stop, restart
// @Tags         docker-containers
// @Security     ApiKeyAuth
// @Param        id      path      string  true  "Container ID"
// @Param        action  path      string  true  "Action to perform (start, stop, restart)"
// @Success      204
// @Router       /v2/docker/containers/{id}/{action} [post]
func (h *Handler) ActionContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	action := r.PathValue("action")

	err := h.Service.ActionContainer(r.Context(), id, action)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteContainer godoc
// @Summary      Delete a Docker Container
// @Description  Removes a container. Pass ?force=true to force removal of a running container.
// @Tags         docker-containers
// @Security     ApiKeyAuth
// @Param        id      path      string  true  "Container ID"
// @Param        force   query     bool    false "Force remove"
// @Success      204
// @Router       /v2/docker/containers/{id} [delete]
func (h *Handler) DeleteContainer(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"

	err := h.Service.DeleteContainer(r.Context(), id, force)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CreateContainerRequest struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Env           []string          `json:"env"`
	Ports         map[string]string `json:"ports"`   // ex: {"8080": "80"}
	Volumes       []string          `json:"volumes"` // ex: ["myvol:/data", "/var/serveoapi/data/app:/app"]
	RestartPolicy string            `json:"restart_policy"`
}

// CreateContainer godoc
// @Summary      Create and Start a Docker Container
// @Description  Deploys a new container. Bind mounts are strictly restricted for security.
// @Tags         docker-containers
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateContainerRequest true "Container configuration"
// @Success      201  {object}  ContainerInfo
// @Failure      400,500 {string} string
// @Router       /v2/docker/containers/create [post]
func (h *Handler) CreateContainer(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req CreateContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	info, err := h.Service.CreateContainer(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "Security Error") {
			response.SendError(w, http.StatusBadRequest, err.Error())
		} else {
			response.SendError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(info)
}

type UpdateContainerRequest struct {
	Env    []string `json:"env"`
	Memory int64    `json:"memory"` // En Bytes
}

// UpdateContainer godoc
// @Summary      Update a Docker Container
// @Description  Updates a container's environment variables or RAM limit transparently (recreates it)
// @Tags         docker-containers
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Container ID"
// @Param        body body UpdateContainerRequest true "Update configuration"
// @Success      200  {object}  ContainerInfo
// @Failure      400,500 {string} string
// @Router       /v2/docker/containers/{id}/update [put]
func (h *Handler) UpdateContainer(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")
	var req UpdateContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	info, err := h.Service.UpdateContainer(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			response.SendError(w, http.StatusNotFound, "Container not found: "+err.Error())
		} else {
			response.SendError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}
