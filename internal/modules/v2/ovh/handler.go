package ovh

import (
	"encoding/json"
	"net/http"
)

// OvhMeResponse represents basic account information
type OvhMeResponse struct {
	Firstname string `json:"firstname"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// CheckConfig is a helper to verify if the module is enabled
func checkConfig(w http.ResponseWriter) bool {
	if GetClient() == nil {
		http.Error(w, "OVH Module is not configured on this API", http.StatusServiceUnavailable)
		return false
	}
	return true
}

// GetMe godoc
// @Summary      Get OVH Account Info
// @Description  Test connection by fetching account information
// @Tags         ovh
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  OvhMeResponse
// @Router       /v2/ovh/me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	if !checkConfig(w) {
		return
	}

	var me OvhMeResponse
	if err := GetClient().Get("/me", &me); err != nil {
		http.Error(w, "OVH API Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(me)
}

// ListDedicatedServers godoc
// @Summary      List BareMetal Servers
// @Description  Returns a list of dedicated servers on the OVH account
// @Tags         ovh
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   string
// @Router       /v2/ovh/dedicated/server [get]
func ListDedicatedServers(w http.ResponseWriter, r *http.Request) {
	if !checkConfig(w) {
		return
	}

	var servers []string
	if err := GetClient().Get("/dedicated/server", &servers); err != nil {
		http.Error(w, "OVH API Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// HardRebootServer godoc
// @Summary      Hard Reboot Server
// @Description  Triggers a hard reboot on a specific dedicated server
// @Tags         ovh
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        serviceName   path      string  true  "Server Service Name (e.g. ns300000.ip-1-2-3.eu)"
// @Success      200  {string}  string "Reboot initiated"
// @Router       /v2/ovh/dedicated/server/{serviceName}/reboot [post]
func HardRebootServer(w http.ResponseWriter, r *http.Request) {
	if !checkConfig(w) {
		return
	}

	serviceName := r.PathValue("serviceName")
	if serviceName == "" {
		http.Error(w, "Missing serviceName parameter", http.StatusBadRequest)
		return
	}

	// OVH Reboot Endpoint
	// Requires a payload {"reason": "string"} according to OVH API docs?
	// Usually POST /dedicated/server/{serviceName}/reboot takes empty body or empty map?
	// The docs say no parameters required for basic reboot, but sometimes a struct is needed.
	// We'll pass nil for body.

	var task map[string]interface{}
	err := GetClient().Post("/dedicated/server/"+serviceName+"/reboot", nil, &task)
	if err != nil {
		http.Error(w, "OVH API Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Hard reboot initiated successfully",
		"task":    task,
	})
}
