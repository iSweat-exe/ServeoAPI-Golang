package docker

import (
	"serveoapi/internal/core/response"
	"context"
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"strings"

	"serveoapi/internal/core/config"
)

// GetContainers godoc
// @Summary      List Docker Containers
// @Description  Returns a list of all docker containers
// @Tags         docker-containers
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   ContainerInfo
// @Router       /v2/docker/containers/ [get]
func GetContainers(w http.ResponseWriter, r *http.Request) {
	cli := GetClient()

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to list containers: "+err.Error())
		return
	}

	var resp []ContainerInfo
	for _, c := range containers {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}
		resp = append(resp, ContainerInfo{
			ID:     id,
			Names:  c.Names,
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
		})
	}

	if resp == nil {
		resp = []ContainerInfo{}
	}

	response.SendJSON(w, http.StatusOK, resp)
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
func InspectContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cli := GetClient()

	info, err := cli.ContainerInspect(context.Background(), id)
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
func ActionContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	action := r.PathValue("action")

	cli := GetClient()
	var err error

	ctx := context.Background()
	switch action {
	case "start":
		err = cli.ContainerStart(ctx, id, container.StartOptions{})
	case "stop":
		err = cli.ContainerStop(ctx, id, container.StopOptions{})
	case "restart":
		err = cli.ContainerRestart(ctx, id, container.StopOptions{})
	default:
		response.SendError(w, http.StatusBadRequest, "Invalid action. Use start, stop, or restart.")
		return
	}

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
func DeleteContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"

	cli := GetClient()

	err := cli.ContainerRemove(context.Background(), id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
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
	Ports         map[string]string `json:"ports"`   // e.g. {"8080": "80"}
	Volumes       []string          `json:"volumes"` // e.g. ["myvol:/data", "/var/serveoapi/data/app:/app"]
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
func CreateContainer(w http.ResponseWriter, r *http.Request) {
	var req CreateContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	cli := GetClient()

	// 1. Security check for Volumes (Bind Mounts)
	cfg := config.Load()
	allowedRoot := cfg.AllowedMountRoot
	var hostConfig container.HostConfig
	hostConfig.Binds = []string{}

	for _, v := range req.Volumes {
		parts := strings.SplitN(v, ":", 2)
		src := parts[0]

		// If it's a bind mount (contains slash), verify it starts with AllowedMountRoot
		if strings.Contains(src, "/") || strings.Contains(src, "\\") {
			// Resolve to absolute path to prevent ../ bypasses
			if !strings.HasPrefix(src, allowedRoot) {
				response.SendError(w, http.StatusBadRequest, "Security Error: Bind mounts are restricted to "+allowedRoot)
				return
			}
		}
		hostConfig.Binds = append(hostConfig.Binds, v)
	}

	// 2. Port Bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for hostPort, containerPort := range req.Ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err == nil {
			exposedPorts[port] = struct{}{}
			portBindings[port] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			}
		}
	}
	hostConfig.PortBindings = portBindings

	// 3. Restart Policy
	if req.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(req.RestartPolicy)}
	}

	// 4. Create Container
	containerConfig := &container.Config{
		Image:        req.Image,
		Env:          req.Env,
		ExposedPorts: exposedPorts,
	}

	resp, err := cli.ContainerCreate(r.Context(), containerConfig, &hostConfig, nil, nil, req.Name)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Create failed: "+err.Error())
		return
	}

	// 5. Start Container
	if err := cli.ContainerStart(r.Context(), resp.ID, container.StartOptions{}); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Start failed: "+err.Error())
		return
	}

	// Return info
	inspect, err := cli.ContainerInspect(r.Context(), resp.ID)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Inspect failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ContainerInfo{
		ID:     inspect.ID[:12],
		Names:  []string{inspect.Name},
		Image:  inspect.Config.Image,
		State:  inspect.State.Status,
		Status: inspect.State.Status, // inspect doesn't have a pre-formatted 'Up X hours' string like ContainerList
	})
}

