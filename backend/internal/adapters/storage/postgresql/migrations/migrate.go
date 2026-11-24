package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"gorm.io/gorm"
)

func MigrateSystemDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Database{},
		&model.User{},
		&model.Group{},
		&model.GroupDatabase{},
		&model.Session{},
		&model.Draft{},
		&model.DraftApproval{},
		&model.DraftCheckpoint{},
		&model.Migration{},
	)
}

func Migrate(db *gorm.DB) error {
	return MigrateSystemDatabase(db)
}
