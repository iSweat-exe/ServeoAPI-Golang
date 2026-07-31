package docker

import (
	"encoding/json"
	"net/http"

	"serveoapi/internal/core/response"

	"serveoapi/internal/core/stream"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// GetSystemInfo godoc
// @Summary      Get Docker System Info
// @Description  Returns system-wide information
// @Tags         docker-system
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  SystemInfo
// @Router       /v2/docker/system/info [get]
func (h *Handler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	cli := h.Service.DockerCli

	info, err := cli.Info(r.Context())
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := SystemInfo{
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersStopped: info.ContainersStopped,
		Images:            info.Images,
		MemTotal:          info.MemTotal,
		NCPU:              info.NCPU,
		ServerVersion:     info.ServerVersion,
	}

	response.SendJSON(w, http.StatusOK, resp)
}

// PruneSystem godoc
// @Summary      Prune Docker System
// @Description  Removes unused data
// @Tags         docker-system
// @Security     ApiKeyAuth
// @Success      204
// @Router       /v2/docker/system/prune [post]
func (h *Handler) PruneSystem(
	w http.ResponseWriter,
	r *http.Request,
) {
	cli := h.Service.DockerCli

	// Le nettoyage complet (images, conteneurs, volumes, réseaux) nécessite des appels séparés dans le SDK
	// Ceci agit comme un nettoyage basique des conteneurs pour le moment
	_, err := cli.ContainersPrune(r.Context(), filters.NewArgs())
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StreamSystemEvents godoc
// @Summary      Stream Docker Events (SSE)
// @Description  Streams real-time events from the server
// @Tags         docker-system
// @Produce      text/event-stream
// @Security     ApiKeyAuth
// @Success      200  {string}  string "Event Stream"
// @Router       /v2/docker/system/events [get]
func (h *Handler) StreamSystemEvents(
	w http.ResponseWriter,
	r *http.Request,
) {
	cli := h.Service.DockerCli

	msgs, errs := cli.Events(r.Context(), events.ListOptions{})

	stream.SetupSSEHeaders(w)

	for {
		select {
		case err := <-errs:
			stream.SendSSEEvent(w, "error: "+err.Error())
			return
		case msg := <-msgs:
			jsonBytes, _ := json.Marshal(msg)
			stream.SendSSEEvent(w, string(jsonBytes))
		case <-r.Context().Done():
			return
		}
	}
}
