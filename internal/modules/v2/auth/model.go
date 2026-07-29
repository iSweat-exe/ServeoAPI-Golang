package auth

import (
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"serveoapi/internal/core/database"
)

// User représente un compte utilisateur dans la DB.
type User struct {
	gorm.Model
	Username       string `gorm:"uniqueIndex;not null" json:"username"`
	Password       string `gorm:"not null" json:"-"`                // Le hash du mot de passe
	Permissions    string `gorm:"type:text" json:"permissions"`     // Permissions séparées par des virgules
	ProfilePicture string `gorm:"type:text" json:"profile_picture"` // URL ou chemin de l'image
	Status         string `gorm:"default:'offline'" json:"status"`  // online, offline, away
	LastConnection *int64 `json:"last_connection"`                  // Timestamp UNIX
	TokenVersion   int    `gorm:"default:0" json:"-"`               // Invalidate tokens on password change
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
	database.DB.Model(&User{}).Count(&count)
	if count == 0 {
		log.Println("Aucun utilisateur trouvé. Création de l'utilisateur par défaut 'admin'...")
		defaultAdmin := User{
			Username:    "admin",
			Permissions: "*",
		}
		if err := defaultAdmin.HashPassword("root"); err != nil {
			return err
		}
		database.DB.Create(&defaultAdmin)
		log.Println("Utilisateur 'admin' (password: 'root') créé avec succès.")
	}

	return nil
}
