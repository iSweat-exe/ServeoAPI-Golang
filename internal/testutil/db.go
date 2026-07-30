package testutil

import (
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory SQLite database for testing and runs AutoMigrate for the given models.
func SetupTestDB(models ...interface{}) (*gorm.DB, error) {
	dbName := uuid.New().String()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if len(models) > 0 {
		err = db.AutoMigrate(models...)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}
