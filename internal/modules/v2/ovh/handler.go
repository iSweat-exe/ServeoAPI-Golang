package ovh

import (
	"net/http"
	"serveoapi/internal/core/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

// OvhMeResponse represents basic account information
type OvhMeResponse struct {
	Firstname string `json:"firstname"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// CheckConfig is a helper to verify if the module is enabled
func checkConfig(w http.ResponseWriter) bool {
	if GetClient() == nil {
		response.SendError(w, http.StatusServiceUnavailable, "OVH Module is not configured on this API")
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
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	if !checkConfig(w) {
		return
	}

	var me OvhMeResponse
	if err := GetClient().Get("/me", &me); err != nil {
		response.SendError(w, http.StatusInternalServerError, "OVH API Error: "+err.Error())
		return
	}

	response.SendJSON(w, http.StatusOK, me)
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
func (h *Handler) ListDedicatedServers(w http.ResponseWriter, r *http.Request) {
	if !checkConfig(w) {
		return
	}

	var servers []string
	if err := GetClient().Get("/dedicated/server", &servers); err != nil {
		response.SendError(w, http.StatusInternalServerError, "OVH API Error: "+err.Error())
		return
	}

	response.SendJSON(w, http.StatusOK, servers)
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
func (h *Handler) HardRebootServer(w http.ResponseWriter, r *http.Request) {
	if !checkConfig(w) {
		return
	}

	serviceName := r.PathValue("serviceName")
	if serviceName == "" {
		response.SendError(w, http.StatusBadRequest, "Missing serviceName parameter")
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
		response.SendError(w, http.StatusInternalServerError, "OVH API Error: "+err.Error())
		return
	}

	response.SendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Hard reboot initiated successfully",
		"task":    task,
	})
}
