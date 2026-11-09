package app

import (
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	driver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	storagepostgresql "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

func buildSystemDatabase(configuration *config.Config) (*gorm.DB, func(), error) {
	systemDatabase, err := storagepostgresql.ConnectToSystemDatabase(configuration.Database)
	if err != nil {
		return nil, nil, err
	}

	closeSystemDatabase := func() {
		sqlDatabase, err := systemDatabase.DB()
		if err != nil {
			log.Printf("failed to access system database for closing: %v", err)
			return
		}
		if err := sqlDatabase.Close(); err != nil {
			log.Printf("failed to close system database: %v", err)
		}
	}
	return systemDatabase, closeSystemDatabase, nil
}

type targetDatabaseInfrastructure struct {
	credentialCipher ports.DatabaseCredentialCipher
	connectionTester ports.DatabaseConnectionTester
	poolManager      *driver.DatabasePoolManager
	databaseDriver   ports.DatabaseDriver
	close            func()
}

func buildTargetDatabaseInfrastructure(configuration *config.Config) (*targetDatabaseInfrastructure, error) {
	credentialCipher, err := security.NewAESGCMCipher(
		[]byte(configuration.Security.DatabaseCredentialKey),
	)
	if err != nil {
		return nil, err
	}

	poolManager := driver.NewDatabasePoolManager()
	return &targetDatabaseInfrastructure{
		credentialCipher: credentialCipher,
		connectionTester: driver.NewDatabaseConnectionTester(),
		poolManager:      poolManager,
		databaseDriver:   driver.NewDatabaseDriver(poolManager),
		close: func() {
			if err := poolManager.CloseAll(); err != nil {
				log.Printf("failed to close target database pools: %v", err)
			}
		},
	}, nil
}
