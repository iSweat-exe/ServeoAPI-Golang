package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
	"serveoapi/internal/core/validation"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

var JwtSecretKey = []byte(getEnvOrDefault("JWT_SECRET", "serveo_super_secret_key_change_me"))

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
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
func (h *Handler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validation.Validator.Struct(req); err != nil {
		response.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	var user User
	// Trouver l'utilisateur par son nom d'utilisateur
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.SendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Vérifier le hash du mot de passe
	if err := user.CheckPassword(req.Password); err != nil {
		response.SendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Mettre à jour LastConnection et Status
	now := time.Now().Unix()
	user.LastConnection = &now
	user.Status = "online"
	h.DB.Save(&user)

	// Générer le JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":           user.ID,
		"username":      user.Username,
		"permissions":   user.Permissions,
		"token_version": user.TokenVersion,                     // Add token_version to claims
		"exp":           time.Now().Add(time.Hour * 24).Unix(), // Expires in 24 hours
	})

	tokenString, err := token.SignedString(JwtSecretKey)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Could not generate token")
		return
	}

	response.SendJSON(w, http.StatusOK, LoginResponse{
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
func (h *Handler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Extraire userID du contexte (injecté par le middleware JWT)
	userID, ok := contextkeys.GetUserID(r.Context())
	if !ok {
		response.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var user User
	if err := h.DB.First(&user, userID).Error; err == nil {
		user.Status = "offline"
		// Optionnel : invalider le token actuel lors du logout manuel :
		// user.TokenVersion++
		h.DB.Save(&user)
	}

	w.WriteHeader(http.StatusNoContent)
}
