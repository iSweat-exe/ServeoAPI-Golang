package firewall

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"serveoapi/internal/core/response"
)

type Handler struct {
	Service *Service
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUFWNotInstalled):
		response.SendError(w, http.StatusServiceUnavailable, "UFW is not installed on this server")
	case errors.Is(err, ErrProtectedRule):
		response.SendError(
			w,
			http.StatusConflict,
			"This rule protects SSH or API access and cannot be modified",
		)
	case errors.Is(err, ErrWouldLockout):
		response.SendError(
			w,
			http.StatusConflict,
			"Enabling the firewall now would block SSH or API access: "+
				"add an allow rule for SSH and for the API port first",
		)
	default:
		response.SendError(w, http.StatusInternalServerError, "UFW command failed: "+err.Error())
	}
}

// GetStatus godoc
// @Summary      Get Firewall Status
// @Description  Returns whether UFW is active and lists its current numbered rules
// @Tags         firewall
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200      {object}  StatusResponse
// @Failure      500,503  {string}  string
// @Router       /v2/firewall/status [get]
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.Service.Status(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.SendJSON(w, http.StatusOK, status)
}

// AddRule godoc
// @Summary      Add Firewall Rule
// @Description  Creates a new UFW rule (allow/deny/reject/limit) for a port or port range
// @Tags         firewall
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        body             body  AddRuleRequest  true  "Rule definition"
// @Success      201              {object}  map[string]string
// @Failure      400,409,500,503  {string}  string
// @Router       /v2/firewall/rules [post]
func (h *Handler) AddRule(w http.ResponseWriter, r *http.Request) {
	var req AddRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	output, err := h.Service.AddRule(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrProtectedRule) || errors.Is(err, ErrUFWNotInstalled) {
			handleServiceError(w, err)
			return
		}
		// Toute autre erreur d'AddRule provient de la validation des champs.
		response.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	response.SendJSON(w, http.StatusCreated, map[string]string{"output": output})
}

// DeleteRule godoc
// @Summary      Delete Firewall Rule
// @Description  Removes a UFW rule by its number (see GET /v2/firewall/status)
// @Tags         firewall
// @Produce      json
// @Security     ApiKeyAuth
// @Param        number               path  int  true  "Rule number"
// @Success      200                  {object}  map[string]string
// @Failure      400,409,500,503      {string}  string
// @Router       /v2/firewall/rules/{number} [delete]
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil || number <= 0 {
		response.SendError(w, http.StatusBadRequest, "Invalid rule number")
		return
	}

	output, err := h.Service.DeleteRule(r.Context(), number)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.SendJSON(w, http.StatusOK, map[string]string{"output": output})
}

// EnableFirewall godoc
// @Summary      Enable Firewall
// @Description  Enables UFW. Refuses if it would immediately lock out SSH or API access.
// @Tags         firewall
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200      {object}  map[string]string
// @Failure      409,500,503  {string}  string
// @Router       /v2/firewall/enable [post]
func (h *Handler) EnableFirewall(w http.ResponseWriter, r *http.Request) {
	output, err := h.Service.Enable(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.SendJSON(w, http.StatusOK, map[string]string{"output": output})
}

// DisableFirewall godoc
// @Summary      Disable Firewall
// @Description  Disables UFW, allowing all traffic
// @Tags         firewall
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200      {object}  map[string]string
// @Failure      500,503  {string}  string
// @Router       /v2/firewall/disable [post]
func (h *Handler) DisableFirewall(w http.ResponseWriter, r *http.Request) {
	output, err := h.Service.Disable(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.SendJSON(w, http.StatusOK, map[string]string{"output": output})
}
