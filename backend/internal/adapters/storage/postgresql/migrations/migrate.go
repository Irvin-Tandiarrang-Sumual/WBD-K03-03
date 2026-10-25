package migrations

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"gorm.io/gorm"
)

func MigrateSystemDatabase(db *gorm.DB) error {
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

	// Older versions accidentally made database_id unique by itself, which
	// prevented a database from having more than one migration record.
	for _, legacyName := range []string{"uni_migrations_database_id", "uni_migrations_version"} {
		if db.Migrator().HasConstraint(&model.Migration{}, legacyName) {
			if err := db.Migrator().DropConstraint(&model.Migration{}, legacyName); err != nil {
				return err
			}
		}
		if db.Migrator().HasIndex(&model.Migration{}, legacyName) {
			if err := db.Migrator().DropIndex(&model.Migration{}, legacyName); err != nil {
				return err
			}
		}
	}
	return nil
}

func Migrate(db *gorm.DB) error {
	return MigrateSystemDatabase(db)
}
