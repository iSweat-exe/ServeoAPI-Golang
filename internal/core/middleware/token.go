package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"serveoapi/internal/core/database"
	"serveoapi/internal/modules/v2/auth"

	"github.com/golang-jwt/jwt/v5"
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

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", errors.New("invalid token payload")
	}

	userID, ok := claimAsUint(claims["sub"])
	if !ok || userID == 0 {
		return 0, "", errors.New("invalid token subject")
	}

	tokenVersion, ok := claimAsUint(claims["token_version"])
	if !ok {
		return 0, "", errors.New("invalid token version")
	}

	// Verify TokenVersion against database
	var user auth.User
	if err := database.DB.Select("token_version, permissions").First(&user, userID).Error; err != nil {
		return 0, "", errors.New("user not found")
	}

	if int(tokenVersion) != user.TokenVersion {
		return 0, "", errors.New("token has been revoked")
	}

	// Les permissions font foi depuis la DB : un changement de droits s'applique
	// immédiatement même si le JWT n'a pas encore été réémis.
	return userID, user.Permissions, nil
}

// claimAsUint convertit un claim JWT numérique (float64 / json.Number / string)
// sans perte d'intégrité pour les identifiants entiers.
func claimAsUint(value interface{}) (uint, bool) {
	switch v := value.(type) {
	case float64:
		if v < 0 || v != math.Trunc(v) {
			return 0, false
		}
		return uint(v), true
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case string:
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return uint(n), true
	default:
		return 0, false
	}
}
