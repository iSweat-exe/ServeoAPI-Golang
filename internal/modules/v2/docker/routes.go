package docker

import (
	"net/http"

	"github.com/docker/docker/client"
	"gorm.io/gorm"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	db *gorm.DB,
	cfg *config.Config,
	dockerCli *client.Client,
) {
	service := &DockerService{
		DockerCli: dockerCli,
		Config:    cfg,
	}
	h := &Handler{
		Service: service,
	}

	registerRoute := func(methodPath, perm string, handler http.HandlerFunc) {
		mux.Handle(methodPath, authMiddleware(middleware.RequirePermission(perm, handler)))
	}

	// Containers
	registerRoute("GET /v2/docker/containers/", "docker.containers.read", h.GetContainers)
	registerRoute(
		"POST /v2/docker/containers/create",
		"docker.containers.write",
		h.CreateContainer,
	)
	registerRoute("GET /v2/docker/containers/{id}", "docker.containers.read", h.InspectContainer)
	registerRoute(
		"PUT /v2/docker/containers/{id}/update",
		"docker.containers.write",
		h.UpdateContainer,
	)
	registerRoute(
		"POST /v2/docker/containers/{id}/{action}",
		"docker.containers.write",
		h.ActionContainer,
	)
	registerRoute(
		"DELETE /v2/docker/containers/{id}",
		"docker.containers.delete",
		h.DeleteContainer,
	)

	// Terminal (Interactive WebSockets, Auth is handled inside the WS)
	mux.Handle("GET /v2/docker/containers/{id}/exec", http.HandlerFunc(h.TerminalHandler))

	// Containers Streaming
	registerRoute(
		"GET /v2/docker/containers/{id}/logs",
		"docker.containers.read",
		h.StreamContainerLogs,
	)
	registerRoute(
		"GET /v2/docker/containers/{id}/stats",
		"docker.containers.read",
		h.StreamContainerStats,
	)

	// Images
	registerRoute("GET /v2/docker/images/", "docker.images.read", h.GetImages)
	registerRoute("POST /v2/docker/images/pull", "docker.images.write", h.PullImage)
	registerRoute("DELETE /v2/docker/images/{id}", "docker.images.delete", h.DeleteImage)

	// System
	registerRoute("GET /v2/docker/system/info", "docker.system.read", h.GetSystemInfo)
	registerRoute("POST /v2/docker/system/prune", "docker.system.delete", h.PruneSystem)
	registerRoute("GET /v2/docker/system/events", "docker.system.read", h.StreamSystemEvents)

	// Volumes
	registerRoute("GET /v2/docker/volumes/", "docker.volumes.read", h.GetVolumes)
	registerRoute("POST /v2/docker/volumes/", "docker.volumes.write", h.CreateVolume)
	registerRoute("DELETE /v2/docker/volumes/{name}", "docker.volumes.delete", h.DeleteVolume)

	// Networks
	registerRoute("GET /v2/docker/networks/", "docker.networks.read", h.GetNetworks)
	registerRoute("POST /v2/docker/networks/", "docker.networks.write", h.CreateNetwork)
	registerRoute("DELETE /v2/docker/networks/{id}", "docker.networks.delete", h.DeleteNetwork)

	// Docker Compose
	registerRoute("POST /v2/docker/compose/deploy", "docker.compose.write", h.DeployStack)
}
