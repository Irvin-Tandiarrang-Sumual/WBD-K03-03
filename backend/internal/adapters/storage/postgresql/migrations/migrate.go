package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"gorm.io/gorm"
)

func MigrateSystemDatabase(db *gorm.DB) error {
	// Drop erroneous/legacy unique constraints first before auto migrating
	_ = db.Exec("ALTER TABLE databases DROP CONSTRAINT IF EXISTS uni_databases_port; DROP INDEX IF EXISTS uni_databases_port;").Error

	if err := db.AutoMigrate(
		&model.Database{},
		&model.Group{},
		&model.GroupDatabase{},
		&model.User{},
		&model.Session{},
		&model.Migration{},
	); err != nil {
		return err
	}

	for _, legacyName := range []string{"uni_migrations_database_id", "uni_migrations_version", "uni_databases_port"} {
		if db.Migrator().HasConstraint(&model.Database{}, legacyName) {
			_ = db.Migrator().DropConstraint(&model.Database{}, legacyName)
		}
		if db.Migrator().HasIndex(&model.Database{}, legacyName) {
			_ = db.Migrator().DropIndex(&model.Database{}, legacyName)
		}
		if db.Migrator().HasConstraint(&model.Migration{}, legacyName) {
			_ = db.Migrator().DropConstraint(&model.Migration{}, legacyName)
		}
		if db.Migrator().HasIndex(&model.Migration{}, legacyName) {
			_ = db.Migrator().DropIndex(&model.Migration{}, legacyName)
		}
	}
	return nil
}

func Migrate(db *gorm.DB) error {
	return MigrateSystemDatabase(db)
}
