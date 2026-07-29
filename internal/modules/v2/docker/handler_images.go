package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/image"
	"serveoapi/internal/core/stream"
)

// GetImages godoc
// @Summary      List Docker Images
// @Description  Returns a list of all docker images
// @Tags         docker-images
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   ImageInfo
// @Router       /v2/docker/images/ [get]
func GetImages(w http.ResponseWriter, r *http.Request) {
	cli := GetClient()

	images, err := cli.ImageList(context.Background(), image.ListOptions{All: false})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []ImageInfo
	for _, img := range images {
		id := img.ID
		if len(id) > 19 {
			id = id[7:19] // Remove "sha256:" and keep short hash
		}

		resp = append(resp, ImageInfo{
			ID:       id,
			RepoTags: img.RepoTags,
			Size:     img.Size,
			Created:  img.Created,
		})
	}

	if resp == nil {
		resp = []ImageInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PullImage godoc
// @Summary      Pull Docker Image (SSE)
// @Description  Pulls an image and streams progress using Server-Sent Events
// @Tags         docker-images
// @Accept       json
// @Produce      text/event-stream
// @Security     ApiKeyAuth
// @Param        body body PullImageRequest true "Image to pull"
// @Success      200  {string}  string "Event Stream"
// @Router       /v2/docker/images/pull [post]
func PullImage(w http.ResponseWriter, r *http.Request) {
	var req PullImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Image == "" {
		http.Error(w, "Invalid payload, missing image name", http.StatusBadRequest)
		return
	}

	cli := GetClient()

	out, err := cli.ImagePull(r.Context(), req.Image, image.PullOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer out.Close()

	stream.SetupSSEHeaders(w)

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		stream.SendSSEEvent(w, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		stream.SendSSEEvent(w, "error: "+err.Error())
	}
	stream.SendSSEEvent(w, `{"status": "Pull complete"}`)
}

// DeleteImage godoc
// @Summary      Delete a Docker Image
// @Description  Removes an image
// @Tags         docker-images
// @Security     ApiKeyAuth
// @Param        id      path      string  true  "Image ID or Tag"
// @Param        force   query     bool    false "Force remove"
// @Success      204
// @Router       /v2/docker/images/{id} [delete]
func DeleteImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"

	cli := GetClient()

	_, err := cli.ImageRemove(context.Background(), id, image.RemoveOptions{
		Force: force,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
