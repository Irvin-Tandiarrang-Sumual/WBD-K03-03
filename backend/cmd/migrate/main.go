package main

import (
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
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

	log.Println("running system database migrations...")
	if err := migrations.MigrateSystemDatabase(db); err != nil {
		log.Fatalf("failed to execute migrations: %v", err)
	}

	log.Println("System database migration completed successfully.")
}
