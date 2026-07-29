package apikeys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/response"
	"serveoapi/internal/modules/v2/common"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

// generateSecureToken creates a random 64-character token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of the token to store in DB
func hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CreateApiKey godoc
// @Summary      Create API Key
// @Description  Generate a new Personal Access Token (PAT) for AIs
// @Tags         apikeys
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request body CreateApiKeyRequest true "Key details"
// @Success      201  {object}  CreateApiKeyResponse
// @Router       /v2/apikeys/create [post]
func (h *Handler) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := contextkeys.GetUserID(r.Context())
	if !ok {
		response.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Manual validation or validation tags if defined in model (assuming they are or we just check Name)
	if req.Name == "" {
		response.SendError(w, http.StatusBadRequest, "Name is required")
		return
	}

	tokenString, err := generateSecureToken()
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Prefix is "srv_" + first 4 chars for identification without exposing full token
	prefix := "srv_" + tokenString[:4]

	apiKey := ApiKey{
		UserID:    userID,
		Name:      req.Name,
		TokenHash: hashToken(tokenString),
		Prefix:    prefix,
	}

	if err := h.DB.Create(&apiKey).Error; err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to save API key")
		return
	}

	resp := CreateApiKeyResponse{
		ID:    apiKey.ID,
		Name:  apiKey.Name,
		Token: tokenString, // Send ONLY once
	}

	response.SendJSON(w, http.StatusCreated, resp)
}

// ListApiKeys godoc
// @Summary      List API Keys
// @Description  Get all API Keys for the authenticated user
// @Tags         apikeys
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   ApiKeyResponse
// @Router       /v2/apikeys/ [get]
func (h *Handler) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.GetUserIDOrUnauthorized(w, r)
	if !ok {
		return
	}

	var keys []ApiKey
	if err := h.DB.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to fetch keys")
		return
	}

	var resp []ApiKeyResponse
	for _, k := range keys {
		resp = append(resp, ApiKeyResponse{
			ID:        k.ID,
			Name:      k.Name,
			Prefix:    k.Prefix,
			CreatedAt: k.CreatedAt,
		})
	}

	response.SendJSON(w, http.StatusOK, resp)
}

// RevokeApiKey godoc
// @Summary      Revoke API Key
// @Description  Delete an API Key
// @Tags         apikeys
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id path int true "API Key ID"
// @Success      200  {object}  map[string]string
// @Router       /v2/apikeys/{id} [delete]
func (h *Handler) RevokeApiKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.GetUserIDOrUnauthorized(w, r)
	if !ok {
		return
	}

	keyID := r.PathValue("id")
	if keyID == "" {
		response.SendError(w, http.StatusBadRequest, "Key ID required")
		return
	}

	res := h.DB.Where("id = ? AND user_id = ?", keyID, userID).Delete(&ApiKey{})
	if res.Error != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to delete key")
		return
	}
	if res.RowsAffected == 0 {
		response.SendError(w, http.StatusNotFound, "Key not found")
		return
	}

	response.SendJSON(w, http.StatusOK, map[string]string{"message": "API Key revoked successfully"})
}
