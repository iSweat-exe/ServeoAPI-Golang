package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"serveoapi/internal/core/database"
	"serveoapi/internal/modules/v2/auth"
)

// ValidateToken checks if a token is a valid PAT or JWT, and returns the UserID and Permissions.
func ValidateToken(tokenString string) (uint, string, error) {
	// Check if it's a PAT (Personal Access Token) - 64 hex characters
	if len(tokenString) == 64 && !strings.Contains(tokenString, ".") {
		hasher := sha256.New()
		hasher.Write([]byte(tokenString))
		hashHex := hex.EncodeToString(hasher.Sum(nil))

		var result struct {
			UserID      uint
			Permissions string
		}

		err := database.DB.Table("api_keys").
			Select("api_keys.user_id, users.permissions").
			Joins("JOIN users ON users.id = api_keys.user_id").
			Where("api_keys.token_hash = ?", hashHex).
			Scan(&result).Error

		if err != nil || result.UserID == 0 {
			return 0, "", errors.New("invalid or revoked API Key")
		}
		return result.UserID, result.Permissions, nil
	}

	// Verify token as JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return auth.JwtSecretKey, nil
	})

	if err != nil || !token.Valid {
		return 0, "", errors.New("invalid or expired JWT token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		permissions, _ := claims["permissions"].(string)
		userIDFloat, _ := claims["sub"].(float64)
		return uint(userIDFloat), permissions, nil
	}

	return 0, "", errors.New("invalid token payload")
}
