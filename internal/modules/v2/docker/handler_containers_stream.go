package docker

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"

	"serveoapi/internal/core/response"

	"serveoapi/internal/core/stream"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
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

	pr, pw := io.Pipe()
	// Fermer le lecteur débloque la goroutine de copie si l'on sort de la boucle en premier.
	defer pr.Close()

	go func() {
		defer pw.Close()
		// StdCopy nettoie le multiplex header de 8 bytes de Docker
		_, _ = stdcopy.StdCopy(pw, pw, reader)
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		// Encode to json string to escape quotes/newlines for SSE data payload safely
		jsonBytes, err := json.Marshal(scanner.Text())
		if err != nil {
			continue
		}
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
