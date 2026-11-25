package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"gorm.io/gorm"
)

func MigrateSystemDatabase(db *gorm.DB) error {
	// databases.current_migration_id and migrations.database_id reference each other
	// create the database table before installing both constraints
	tables := db.Session(&gorm.Session{})
	tables.IgnoreRelationshipsWhenMigrating = true
	if !tables.Migrator().HasTable(&model.Database{}) {
		if err := tables.Migrator().CreateTable(&model.Database{}); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(
		&model.Database{},
		&model.User{},
		&model.Group{},
		&model.GroupDatabase{},
		&model.Session{},
		&model.Draft{},
		&model.DraftApproval{},
		&model.DraftCheckpoint{},
		&model.Migration{},
	); err != nil {
		return err
	}
	return backfillMigrationPositions(db)
}

func Migrate(db *gorm.DB) error {
	return MigrateSystemDatabase(db)
}
