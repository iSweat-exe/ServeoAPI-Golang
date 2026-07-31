package auth

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"os"

	"serveoapi/internal/core/database"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User représente un compte utilisateur dans la DB.
type User struct {
	gorm.Model
	UUID           string `gorm:"type:char(36);uniqueIndex" json:"uuid"`
	Username       string `gorm:"uniqueIndex;not null"      json:"username"`
	Password       string `gorm:"not null"                  json:"-"`               // Le hash du mot de passe
	Permissions    string `gorm:"type:text"                 json:"permissions"`     // Permissions séparées par des virgules
	ProfilePicture string `gorm:"type:text"                 json:"profile_picture"` // URL ou chemin de l'image
	Status         string `gorm:"default:'offline'"         json:"status"`          // online, offline, away
	LastConnection *int64 `                                 json:"last_connection"` // Timestamp UNIX
	TokenVersion   int    `gorm:"default:0"                 json:"-"`               // Invalidate tokens on password change
}

// BeforeCreate will set a UUID rather than numeric ID.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.UUID == "" {
		u.UUID = uuid.New().String()
	}
	return
}

// CheckPassword vérifie si le mot de passe fourni correspond au hash.
func (u *User) CheckPassword(providedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(providedPassword))
}

// HashPassword hache le mot de passe de l'utilisateur avec Bcrypt (et un salt automatique).
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// MigrateDatabase crée les tables nécessaires et ajoute le compte admin par défaut.
func MigrateDatabase() error {
	err := database.DB.AutoMigrate(&User{})
	if err != nil {
		return err
	}

	// Création du compte par défaut si la table est vide
	var count int64
	if err := database.DB.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	slog.Info("Aucun utilisateur trouvé. Création de l'utilisateur par défaut 'admin'...")

	// Le mot de passe provient de l'environnement, sinon il est généré aléatoirement :
	// aucun identifiant par défaut connu ne doit exister.
	password, generated := os.LookupEnv("ADMIN_PASSWORD")
	if !generated || password == "" {
		password, err = generatePassword()
		if err != nil {
			return err
		}
		generated = false
	}

	defaultAdmin := User{
		Username:    "admin",
		Permissions: "*",
	}
	if err := defaultAdmin.HashPassword(password); err != nil {
		return err
	}
	if err := database.DB.Create(&defaultAdmin).Error; err != nil {
		return err
	}

	if generated {
		slog.Info("Utilisateur 'admin' créé avec le mot de passe fourni via ADMIN_PASSWORD.")
	} else {
		slog.Warn(
			"Utilisateur 'admin' créé avec un mot de passe généré. Notez-le maintenant, il ne sera plus affiché, et changez-le après la première connexion.",
			"password", password,
		)
	}

	return nil
}

// generatePassword produit un mot de passe initial aléatoire de 128 bits.
func generatePassword() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
