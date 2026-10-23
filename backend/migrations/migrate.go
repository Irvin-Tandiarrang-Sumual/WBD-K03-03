package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations/models"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Database{},
		&models.Group{},
		&models.User{},
		&models.Session{},
		&models.Migration{},
	)
}
