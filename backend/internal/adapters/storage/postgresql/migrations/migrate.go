package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"gorm.io/gorm"
)

func MigrateSystemDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Database{},
		&model.Group{},
		&model.GroupDatabase{},
		&model.User{},
		&model.Session{},
		&model.Migration{},
	)
}

func Migrate(db *gorm.DB) error {
	return MigrateSystemDatabase(db)
}
