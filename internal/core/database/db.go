package database

import (
	"log/slog"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase initialise la connexion à la base de données SQLite.
func InitDatabase(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	// Enable WAL mode for concurrency
	DB.Exec("PRAGMA journal_mode=WAL;")

	slog.Info("Database connection successfully established.")
	return nil
}
