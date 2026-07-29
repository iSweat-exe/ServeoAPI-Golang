package apikeys

import (
	"time"

	"serveoapi/internal/core/database"
)

// ApiKey represents a Personal Access Token generated for AI/External use
type ApiKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`
	TokenHash string    `gorm:"uniqueIndex;not null" json:"-"`
	Prefix    string    `gorm:"not null" json:"prefix"` // Used to identify the token (e.g. first 4 chars)
	CreatedAt time.Time `json:"created_at"`
}

// MigrateDatabase applies migrations for apikeys module
func MigrateDatabase() error {
	return database.DB.AutoMigrate(&ApiKey{})
}

type CreateApiKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateApiKeyResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"` // Only shown once during creation
}

type ApiKeyResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}
