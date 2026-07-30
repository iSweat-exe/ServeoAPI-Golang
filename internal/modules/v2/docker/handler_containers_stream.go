package docker

import (
	"bufio"
	"encoding/json"
	"net/http"

	"serveoapi/internal/core/response"

	"serveoapi/internal/core/stream"

	"github.com/docker/docker/api/types/container"
)

// StreamContainerLogs godoc
// @Summary      Stream Container Logs (SSE)
// @Description  Streams logs of a container using Server-Sent Events
// @Tags         docker-containers
// @Produce      text/event-stream
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Container ID"
// @Success      200  {string}  string "Event Stream"
// @Router       /v2/docker/containers/{id}/logs [get]
func (h *Handler) StreamContainerLogs(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")
	cli := h.Service.DockerCli

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "50",
	}

	reader, err := cli.ContainerLogs(r.Context(), id, options)
	if err != nil {
		response.SendError(w, http.StatusNotFound, err.Error())
		return
	}
	defer reader.Close()

	stream.SetupSSEHeaders(w)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Enlever l'en-tête de 8 bytes des logs docker multiplexés (stdout/stderr)
		text := scanner.Text()
		if len(text) > 8 {
			text = text[8:]
		}

		// Encode to json string to escape quotes/newlines for SSE data payload safely
		jsonBytes, _ := json.Marshal(text)
		stream.SendSSEEvent(w, string(jsonBytes))
	}
	if err := scanner.Err(); err != nil {
		stream.SendSSEEvent(w, "error: "+err.Error())
	}
}

// StreamContainerStats godoc
// @Summary      Stream Container Stats (SSE)
// @Description  Streams CPU/RAM stats of a container using Server-Sent Events
// @Tags         docker-containers
// @Produce      text/event-stream
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Container ID"
// @Success      200  {string}  string "Event Stream"
// @Router       /v2/docker/containers/{id}/stats [get]
func (h *Handler) StreamContainerStats(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")
	cli := h.Service.DockerCli

	statsResponse, err := cli.ContainerStats(r.Context(), id, true)
	if err != nil {
		response.SendError(w, http.StatusNotFound, err.Error())
		return
	}
	defer statsResponse.Body.Close()

	stream.SetupSSEHeaders(w)

	scanner := bufio.NewScanner(statsResponse.Body)
	for scanner.Scan() {
		stream.SendSSEEvent(w, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		stream.SendSSEEvent(w, "error: "+err.Error())
	}
}
