package integration_test

import (
	"os"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("PLATFORM_DATABASE_URL")

	if dsn == "" {
		t.Skip(
			"PLATFORM_DATABASE_URL is not set; skipping PostgreSQL integration test",
		)
	}

	db, err := gorm.Open(
		postgres.Open(dsn),
		&gorm.Config{
			TranslateError: true,
		},
	)

	if err != nil {
		t.Fatalf(
			"open test database: %v",
			err,
		)
	}

	if err := migrations.Migrate(db); err != nil {
		t.Fatalf(
			"migrate test database: %v",
			err,
		)
	}

	resetTestDB(t, db)

	t.Cleanup(func() {
		resetTestDB(t, db)
	})

	return db
}

func resetTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	err := db.Exec(`
		TRUNCATE TABLE
			sessions,
			user_groups,
			group_databases,
			migrations,
			users,
			groups,
			databases
		CASCADE
	`).Error

	if err != nil {
		t.Fatalf(
			"reset test database: %v",
			err,
		)
	}
}

func seedTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := migrations.Seed(db); err != nil {
		t.Fatalf(
			"seed test database: %v",
			err,
		)
	}
}