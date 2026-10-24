package driver

import (
	"context"
	"errors"

	mysqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/mysql"
	postgresqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type DatabaseConnectionTester struct {
	postgresqlConnectionTester *postgresqldriver.ConnectionTester
	mysqlConnectionTester      *mysqldriver.ConnectionTester
}

var _ ports.DatabaseConnectionTester = (*DatabaseConnectionTester)(nil)

func NewDatabaseConnectionTester() *DatabaseConnectionTester {
	return &DatabaseConnectionTester{
		postgresqlConnectionTester: postgresqldriver.NewConnectionTester(),
		mysqlConnectionTester:      mysqldriver.NewConnectionTester(),
	}
}

func (t *DatabaseConnectionTester) Test(ctx context.Context, database domain.Database) error {
	switch database.Type {
	case "postgresql":
		return t.postgresqlConnectionTester.Test(ctx, database)
	case "mysql":
		return t.mysqlConnectionTester.Test(ctx, database)
	default:
		return errors.New("unsupported database type")
	}
}

type DatabasePoolManager struct {
	postgresqlPoolManager *postgresqldriver.PoolManager
	mysqlPoolManager      *mysqldriver.PoolManager
}

var _ ports.DatabasePoolInvalidator = (*DatabasePoolManager)(nil)

func NewDatabasePoolManager() *DatabasePoolManager {
	return &DatabasePoolManager{
		postgresqlPoolManager: postgresqldriver.NewPoolManager(),
		mysqlPoolManager:      mysqldriver.NewPoolManager(),
	}
}

func (m *DatabasePoolManager) Invalidate(databaseID uint) error {
	return errors.Join(
		m.postgresqlPoolManager.Invalidate(databaseID),
		m.mysqlPoolManager.Invalidate(databaseID),
	)
}

func (m *DatabasePoolManager) CloseAll() error {
	return errors.Join(
		m.postgresqlPoolManager.CloseAll(),
		m.mysqlPoolManager.CloseAll(),
	)
}
