package users

import (
	"encoding/json"
	"net/http"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
	"serveoapi/internal/core/validation"
	"serveoapi/internal/modules/v2/auth"
)

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// GetMe godoc
// @Summary      Get my profile
// @Description  Get the currently authenticated user's profile
// @Tags         users
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  UserResponse
// @Router       /v2/users/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := contextkeys.GetUserID(r.Context())
	if !ok {
		response.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var user auth.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		response.SendError(w, http.StatusNotFound, "User not found")
		return
	}

	response.SendJSON(w, http.StatusOK, mapToResponse(user))
}

// UpdateMePassword godoc
// @Summary      Update my password
// @Description  Change your own password
// @Tags         users
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body UpdatePasswordRequest true "Password update details"
// @Success      204
// @Failure      400,401,500 {string} string
// @Router       /v2/users/me/password [put]
func (h *Handler) UpdateMePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := contextkeys.GetUserID(r.Context())
	if !ok {
		response.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // limite de 1 Mo

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	if err := validation.Validator.Struct(req); err != nil {
		response.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.NewPassword == req.OldPassword {
		response.SendError(w, http.StatusBadRequest, "New password must differ from old password")
		return
	}

	var user auth.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		response.SendError(w, http.StatusNotFound, "User not found")
		return
	}

	if err := user.CheckPassword(req.OldPassword); err != nil {
		response.SendError(w, http.StatusUnauthorized, "Invalid old password")
		return
	}

	if err := user.HashPassword(req.NewPassword); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Could not hash new password")
		return
	}

	// Incrémente TokenVersion pour invalider tous les tokens existants de cet utilisateur
	user.TokenVersion++

	if err := h.DB.Save(&user).Error; err != nil {
		response.SendError(w, http.StatusInternalServerError, "Could not save new password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
