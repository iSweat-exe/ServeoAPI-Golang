package users

import (
	"encoding/json"
	"net/http"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/database"
	"serveoapi/internal/modules/v2/auth"
)

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// GetMe godoc
// @Summary      Get my profile
// @Description  Get the currently authenticated user's profile
// @Tags         users
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {object}  UserResponse
// @Router       /v2/users/me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDObj.(uint)

	var user auth.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapToResponse(user))
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
func UpdateMePassword(w http.ResponseWriter, r *http.Request) {
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDObj.(uint)

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var user auth.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := user.CheckPassword(req.OldPassword); err != nil {
		http.Error(w, "Invalid old password", http.StatusUnauthorized)
		return
	}

	if err := user.HashPassword(req.NewPassword); err != nil {
		http.Error(w, "Could not hash new password", http.StatusInternalServerError)
		return
	}

	database.DB.Save(&user)
	w.WriteHeader(http.StatusNoContent)
}
