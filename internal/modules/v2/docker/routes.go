package docker

import (
	"net/http"
	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
	"github.com/docker/docker/client"
	"gorm.io/gorm"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB, cfg *config.Config, dockerCli *client.Client) {
	service := &DockerService{
		DockerCli: dockerCli,
		Config:    cfg,
	}
	h := &Handler{
		Service: service,
	}

	// Containers
	mux.Handle("GET /v2/docker/containers/", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(h.GetContainers))))
	mux.Handle("POST /v2/docker/containers/create", authMiddleware(middleware.RequirePermission("docker.containers.write", http.HandlerFunc(h.CreateContainer))))
	mux.Handle("GET /v2/docker/containers/{id}", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(h.InspectContainer))))
	mux.Handle("PUT /v2/docker/containers/{id}/update", authMiddleware(middleware.RequirePermission("docker.containers.write", http.HandlerFunc(h.UpdateContainer))))
	mux.Handle("POST /v2/docker/containers/{id}/{action}", authMiddleware(middleware.RequirePermission("docker.containers.write", http.HandlerFunc(h.ActionContainer))))
	mux.Handle("DELETE /v2/docker/containers/{id}", authMiddleware(middleware.RequirePermission("docker.containers.delete", http.HandlerFunc(h.DeleteContainer))))

	// Terminal (Interactive WebSockets, Auth is handled inside the WS)
	mux.Handle("GET /v2/docker/containers/{id}/exec", http.HandlerFunc(h.TerminalHandler))

	// Containers Streaming
	mux.Handle("GET /v2/docker/containers/{id}/logs", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(h.StreamContainerLogs))))
	mux.Handle("GET /v2/docker/containers/{id}/stats", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(h.StreamContainerStats))))

	// Images
	mux.Handle("GET /v2/docker/images/", authMiddleware(middleware.RequirePermission("docker.images.read", http.HandlerFunc(h.GetImages))))
	mux.Handle("POST /v2/docker/images/pull", authMiddleware(middleware.RequirePermission("docker.images.write", http.HandlerFunc(h.PullImage))))
	mux.Handle("DELETE /v2/docker/images/{id}", authMiddleware(middleware.RequirePermission("docker.images.delete", http.HandlerFunc(h.DeleteImage))))

	// System
	mux.Handle("GET /v2/docker/system/info", authMiddleware(middleware.RequirePermission("docker.system.read", http.HandlerFunc(h.GetSystemInfo))))
	mux.Handle("POST /v2/docker/system/prune", authMiddleware(middleware.RequirePermission("docker.system.delete", http.HandlerFunc(h.PruneSystem))))
	mux.Handle("GET /v2/docker/system/events", authMiddleware(middleware.RequirePermission("docker.system.read", http.HandlerFunc(h.StreamSystemEvents))))

	// Volumes
	mux.Handle("GET /v2/docker/volumes/", authMiddleware(middleware.RequirePermission("docker.volumes.read", http.HandlerFunc(h.GetVolumes))))
	mux.Handle("POST /v2/docker/volumes/", authMiddleware(middleware.RequirePermission("docker.volumes.write", http.HandlerFunc(h.CreateVolume))))
	mux.Handle("DELETE /v2/docker/volumes/{name}", authMiddleware(middleware.RequirePermission("docker.volumes.delete", http.HandlerFunc(h.DeleteVolume))))

	// Networks
	mux.Handle("GET /v2/docker/networks/", authMiddleware(middleware.RequirePermission("docker.networks.read", http.HandlerFunc(h.GetNetworks))))
	mux.Handle("POST /v2/docker/networks/", authMiddleware(middleware.RequirePermission("docker.networks.write", http.HandlerFunc(h.CreateNetwork))))
	mux.Handle("DELETE /v2/docker/networks/{id}", authMiddleware(middleware.RequirePermission("docker.networks.delete", http.HandlerFunc(h.DeleteNetwork))))

	// Docker Compose
	mux.Handle("POST /v2/docker/compose/deploy", authMiddleware(middleware.RequirePermission("docker.compose.write", http.HandlerFunc(h.DeployStack))))
}
