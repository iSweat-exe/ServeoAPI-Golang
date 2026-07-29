package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/database"
)

// TODO: In production, load this from an environment variable!
var JwtSecretKey = []byte("serveo_super_secret_key_change_me")

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// Login godoc
// @Summary      Login to the API
// @Description  Authenticates a user and returns a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200  {object}  LoginResponse
// @Failure      401  {string}  string "Invalid credentials"
// @Router       /v2/auth/login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user User
	// Find user by username
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password hash
	if err := user.CheckPassword(req.Password); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Update LastConnection & Status
	now := time.Now().Unix()
	user.LastConnection = &now
	user.Status = "online"
	database.DB.Save(&user)

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         user.ID,
		"username":    user.Username,
		"permissions": user.Permissions,
		"exp":         time.Now().Add(time.Hour * 24).Unix(), // Expires in 24 hours
	})

	tokenString, err := token.SignedString(JwtSecretKey)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token: tokenString,
	})
}

// Logout godoc
// @Summary      Logout from the API
// @Description  Sets the user status to offline
// @Tags         auth
// @Security     ApiKeyAuth
// @Success      204
// @Router       /v2/auth/logout [post]
func Logout(w http.ResponseWriter, r *http.Request) {
	// Extract userID from context (injected by JWT middleware)
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := userIDObj.(uint)

	// Set status to offline
	database.DB.Model(&User{}).Where("id = ?", userID).Update("status", "offline")

	w.WriteHeader(http.StatusNoContent)
}
