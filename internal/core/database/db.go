package database

import (
	"fmt"
	"log/slog"
	"time"

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

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB handle: %w", err)
	}

	// SQLite ne supporte qu'un seul writer : un pool trop large provoque
	// des "database is locked" plutôt que du parallélisme réel.
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(time.Hour)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA foreign_keys=ON;",
	} {
		if err := DB.Exec(pragma).Error; err != nil {
			return fmt.Errorf("apply %q: %w", pragma, err)
		}
	}

	slog.Info("Database connection successfully established.")
	return nil
}
