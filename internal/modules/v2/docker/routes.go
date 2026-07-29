package docker

import (
	"net/http"
	"serveoapi/internal/core/middleware"
	"gorm.io/gorm"
)

func RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, db *gorm.DB) {
	// Containers
	mux.Handle("GET /v2/docker/containers/", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(GetContainers))))
	mux.Handle("POST /v2/docker/containers/create", authMiddleware(middleware.RequirePermission("docker.containers.write", http.HandlerFunc(CreateContainer))))
	mux.Handle("GET /v2/docker/containers/{id}", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(InspectContainer))))
	mux.Handle("POST /v2/docker/containers/{id}/{action}", authMiddleware(middleware.RequirePermission("docker.containers.write", http.HandlerFunc(ActionContainer))))
	mux.Handle("DELETE /v2/docker/containers/{id}", authMiddleware(middleware.RequirePermission("docker.containers.delete", http.HandlerFunc(DeleteContainer))))

	// Terminal (Interactive WebSockets, Auth is handled inside the WS)
	mux.Handle("GET /v2/docker/containers/{id}/exec", http.HandlerFunc(TerminalHandler))

	// Containers Streaming
	mux.Handle("GET /v2/docker/containers/{id}/logs", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(StreamContainerLogs))))
	mux.Handle("GET /v2/docker/containers/{id}/stats", authMiddleware(middleware.RequirePermission("docker.containers.read", http.HandlerFunc(StreamContainerStats))))

	// Images
	mux.Handle("GET /v2/docker/images/", authMiddleware(middleware.RequirePermission("docker.images.read", http.HandlerFunc(GetImages))))
	mux.Handle("POST /v2/docker/images/pull", authMiddleware(middleware.RequirePermission("docker.images.write", http.HandlerFunc(PullImage))))
	mux.Handle("DELETE /v2/docker/images/{id}", authMiddleware(middleware.RequirePermission("docker.images.delete", http.HandlerFunc(DeleteImage))))

	// System
	mux.Handle("GET /v2/docker/system/info", authMiddleware(middleware.RequirePermission("docker.system.read", http.HandlerFunc(GetSystemInfo))))
	mux.Handle("POST /v2/docker/system/prune", authMiddleware(middleware.RequirePermission("docker.system.delete", http.HandlerFunc(PruneSystem))))
	mux.Handle("GET /v2/docker/system/events", authMiddleware(middleware.RequirePermission("docker.system.read", http.HandlerFunc(StreamSystemEvents))))

	// Volumes
	mux.Handle("GET /v2/docker/volumes/", authMiddleware(middleware.RequirePermission("docker.volumes.read", http.HandlerFunc(GetVolumes))))
	mux.Handle("POST /v2/docker/volumes/", authMiddleware(middleware.RequirePermission("docker.volumes.write", http.HandlerFunc(CreateVolume))))
	mux.Handle("DELETE /v2/docker/volumes/{name}", authMiddleware(middleware.RequirePermission("docker.volumes.delete", http.HandlerFunc(DeleteVolume))))

	// Networks
	mux.Handle("GET /v2/docker/networks/", authMiddleware(middleware.RequirePermission("docker.networks.read", http.HandlerFunc(GetNetworks))))
	mux.Handle("POST /v2/docker/networks/", authMiddleware(middleware.RequirePermission("docker.networks.write", http.HandlerFunc(CreateNetwork))))
	mux.Handle("DELETE /v2/docker/networks/{id}", authMiddleware(middleware.RequirePermission("docker.networks.delete", http.HandlerFunc(DeleteNetwork))))

	// Docker Compose
	mux.Handle("POST /v2/docker/compose/deploy", authMiddleware(middleware.RequirePermission("docker.compose.write", http.HandlerFunc(DeployStack))))
}
