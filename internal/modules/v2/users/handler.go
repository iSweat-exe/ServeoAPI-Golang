package users

import (
	"encoding/json"
	"net/http"
	"strconv"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/database"
	"serveoapi/internal/modules/v2/auth"
)

type CreateUserRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Permissions    string `json:"permissions"`
	ProfilePicture string `json:"profile_picture"`
}

type UpdateUserRequest struct {
	Permissions    *string `json:"permissions"`
	ProfilePicture *string `json:"profile_picture"`
	Status         *string `json:"status"` // allow banning or forcing offline
}

type UserResponse struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Permissions    string `json:"permissions"`
	ProfilePicture string `json:"profile_picture"`
	Status         string `json:"status"`
	LastConnection *int64 `json:"last_connection"`
}

func mapToResponse(u auth.User) UserResponse {
	return UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		Permissions:    u.Permissions,
		ProfilePicture: u.ProfilePicture,
		Status:         u.Status,
		LastConnection: u.LastConnection,
	}
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Creates a user with hashed password and specific permissions
// @Tags         users
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateUserRequest true "User details"
// @Success      201  {object}  UserResponse
// @Failure      400  {string}  string
// @Router       /v2/users/ [post]
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	newUser := auth.User{
		Username:       req.Username,
		Permissions:    req.Permissions,
		ProfilePicture: req.ProfilePicture,
		Status:         "offline",
	}

	if err := newUser.HashPassword(req.Password); err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		http.Error(w, "Could not create user (username might exist)", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mapToResponse(newUser))
}

// GetUsers godoc
// @Summary      List users
// @Description  Get a list of all users
// @Tags         users
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {array}   UserResponse
// @Router       /v2/users/ [get]
func GetUsers(w http.ResponseWriter, r *http.Request) {
	var users []auth.User
	if err := database.DB.Find(&users).Error; err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	var res []UserResponse
	for _, u := range users {
		res = append(res, mapToResponse(u))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Modify user permissions or profile
// @Tags         users
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Param        body body UpdateUserRequest true "Update details"
// @Success      200  {object}  UserResponse
// @Router       /v2/users/{id} [patch]
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var user auth.User
	if err := database.DB.First(&user, id).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if req.Permissions != nil {
		user.Permissions = *req.Permissions
	}
	if req.ProfilePicture != nil {
		user.ProfilePicture = *req.ProfilePicture
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	database.DB.Save(&user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapToResponse(user))
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Permanently delete a user account
// @Tags         users
// @Security     ApiKeyAuth
// @Param        id   path      int  true  "User ID"
// @Success      204
// @Failure      400  {string}  string "Cannot delete yourself"
// @Router       /v2/users/{id} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Prevent deleting yourself
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj != nil {
		userID := userIDObj.(uint)
		idStr := strconv.FormatUint(uint64(userID), 10)
		if id == idStr {
			http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
			return
		}
	}

	database.DB.Delete(&auth.User{}, id)
	w.WriteHeader(http.StatusNoContent)
}
