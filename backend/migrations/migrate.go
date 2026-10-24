package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations/models"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.SetupJoinTable(
		&models.Group{},
		"Databases",
		&models.GroupDatabase{},
	); err != nil {
		return err
	}

	if err := db.SetupJoinTable(
		&models.Database{},
		"Groups",
		&models.GroupDatabase{},
	); err != nil {
		return err
	}

	return db.AutoMigrate(
		&models.Database{},
		&models.Group{},
		&models.GroupDatabase{},
		&models.User{},
		&models.Session{},
		&models.Migration{},
	)
}
