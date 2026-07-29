package apikeys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"serveoapi/internal/core/contextkeys"
	"serveoapi/internal/core/database"
)

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
func CreateApiKey(w http.ResponseWriter, r *http.Request) {
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDObj.(uint)

	var req CreateApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokenString, err := generateSecureToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
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

	if err := database.DB.Create(&apiKey).Error; err != nil {
		http.Error(w, "Failed to save API key", http.StatusInternalServerError)
		return
	}

	resp := CreateApiKeyResponse{
		ID:    apiKey.ID,
		Name:  apiKey.Name,
		Token: tokenString, // Send ONLY once
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ListApiKeys godoc
// @Summary      List API Keys
// @Description  Get all API Keys for the authenticated user
// @Tags         apikeys
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   ApiKeyResponse
// @Router       /v2/apikeys/ [get]
func ListApiKeys(w http.ResponseWriter, r *http.Request) {
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDObj.(uint)

	var keys []ApiKey
	if err := database.DB.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		http.Error(w, "Failed to fetch keys", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
func RevokeApiKey(w http.ResponseWriter, r *http.Request) {
	userIDObj := r.Context().Value(contextkeys.UserIDKey)
	if userIDObj == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDObj.(uint)

	keyID := r.PathValue("id")
	if keyID == "" {
		http.Error(w, "Key ID required", http.StatusBadRequest)
		return
	}

	res := database.DB.Where("id = ? AND user_id = ?", keyID, userID).Delete(&ApiKey{})
	if res.Error != nil {
		http.Error(w, "Failed to delete key", http.StatusInternalServerError)
		return
	}
	if res.RowsAffected == 0 {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "API Key revoked successfully"})
}
