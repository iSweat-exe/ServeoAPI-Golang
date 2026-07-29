package users

import (
	"encoding/json"
	"net/http"
	"strconv"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
	"serveoapi/internal/core/validation"
	"serveoapi/internal/modules/v2/auth"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

type CreateUserRequest struct {
	Username       string `json:"username" validate:"required"`
	Password       string `json:"password" validate:"required,min=8"`
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
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	if err := validation.Validator.Struct(req); err != nil {
		response.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	newUser := auth.User{
		Username:       req.Username,
		Permissions:    req.Permissions,
		ProfilePicture: req.ProfilePicture,
		Status:         "offline",
	}

	if err := newUser.HashPassword(req.Password); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}

	if err := h.DB.Create(&newUser).Error; err != nil {
		response.SendError(w, http.StatusConflict, "Could not create user (username might exist)")
		return
	}

	response.SendJSON(w, http.StatusCreated, mapToResponse(newUser))
}

// GetUsers godoc
// @Summary      List users
// @Description  Get a list of all users
// @Tags         users
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {array}   UserResponse
// @Router       /v2/users/ [get]
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	var users []auth.User
	if err := h.DB.Find(&users).Error; err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error fetching users")
		return
	}

	var res []UserResponse
	for _, u := range users {
		res = append(res, mapToResponse(u))
	}

	response.SendJSON(w, http.StatusOK, res)
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
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	if err := validation.Validator.Struct(req); err != nil {
		response.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	var user auth.User
	if err := h.DB.First(&user, id).Error; err != nil {
		response.SendError(w, http.StatusNotFound, "User not found")
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

	h.DB.Save(&user)

	response.SendJSON(w, http.StatusOK, mapToResponse(user))
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
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Prevent deleting yourself
	if userID, ok := contextkeys.GetUserID(r.Context()); ok {
		idStr := strconv.FormatUint(uint64(userID), 10)
		if id == idStr {
			response.SendError(w, http.StatusBadRequest, "Cannot delete your own account")
			return
		}
	}

	h.DB.Delete(&auth.User{}, id)
	w.WriteHeader(http.StatusNoContent)
}
