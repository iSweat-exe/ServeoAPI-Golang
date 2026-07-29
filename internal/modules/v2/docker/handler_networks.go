package docker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/network"
)

// GetNetworks godoc
// @Summary      List Docker Networks
// @Description  Returns a list of all networks
// @Tags         docker-networks
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   NetworkInfo
// @Router       /v2/docker/networks/ [get]
func GetNetworks(w http.ResponseWriter, r *http.Request) {
	cli := GetClient()

	networks, err := cli.NetworkList(context.Background(), network.ListOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []NetworkInfo
	for _, n := range networks {
		id := n.ID
		if len(id) > 12 {
			id = id[:12]
		}
		resp = append(resp, NetworkInfo{
			ID:     id,
			Name:   n.Name,
			Driver: n.Driver,
		})
	}

	if resp == nil {
		resp = []NetworkInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DeleteNetwork godoc
// @Summary      Delete a Docker Network
// @Description  Removes a network by ID
// @Tags         docker-networks
// @Security     ApiKeyAuth
// @Param        id    path      string  true  "Network ID"
// @Success      204
// @Router       /v2/docker/networks/{id} [delete]
func DeleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cli := GetClient()

	err := cli.NetworkRemove(context.Background(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CreateNetworkRequest struct {
	Name   string            `json:"name"`
	Driver string            `json:"driver"`
	Labels map[string]string `json:"labels"`
}

// CreateNetwork godoc
// @Summary      Create a Docker Network
// @Description  Create a new Docker network manually
// @Tags         networks
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateNetworkRequest true "Network details"
// @Success      201  {object}  NetworkInfo
// @Failure      400,500 {string} string
// @Router       /v2/docker/networks/ [post]
func CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req CreateNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	cli := GetClient()

	if req.Driver == "" {
		req.Driver = "bridge"
	}

	options := network.CreateOptions{
		Driver: req.Driver,
		Labels: req.Labels,
	}

	res, err := cli.NetworkCreate(r.Context(), req.Name, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch created network to return info
	net, err := cli.NetworkInspect(r.Context(), res.ID, network.InspectOptions{})
	if err != nil {
		// Just return ID if inspect fails
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(NetworkInfo{
			ID:     res.ID,
			Name:   req.Name,
			Driver: req.Driver,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NetworkInfo{
		ID:     net.ID,
		Name:   net.Name,
		Driver: net.Driver,
	})
}
