package main

import (
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	storagepostgresql "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/migrations"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	db, err := storagepostgresql.ConnectToSystemDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to system database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to obtain underlying database handle: %v", err)
	}
	defer sqlDB.Close()

	log.Println("ensuring system database schema is up-to-date...")
	if err := migrations.MigrateSystemDatabase(db); err != nil {
		log.Fatalf("failed to execute migrations prior to seeding: %v", err)
	}

	credentialCipher, err := security.NewAESGCMCipher([]byte(cfg.Security.DatabaseCredentialKey))
	if err != nil {
		log.Fatalf("failed to initialize credential cipher: %v", err)
	}

	log.Println("seeding initial development and test data...")
	if err := migrations.SeedDevelopmentData(db, credentialCipher); err != nil {
		log.Fatalf("failed to seed development data: %v", err)
	}

	log.Println("Development data seeded successfully:")
}
