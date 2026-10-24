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

type DatabaseExecutor struct {
	postgresqlExecutor *postgresqldriver.DatabaseExecutor
	mysqlExecutor      *mysqldriver.DatabaseExecutor
}

var _ ports.DatabaseExecutor = (*DatabaseExecutor)(nil)

func NewDatabaseExecutor(poolManager *DatabasePoolManager) *DatabaseExecutor {
	return &DatabaseExecutor{
		postgresqlExecutor: postgresqldriver.NewDatabaseExecutor(poolManager.postgresqlPoolManager),
		mysqlExecutor:      mysqldriver.NewDatabaseExecutor(poolManager.mysqlPoolManager),
	}
}

func (e *DatabaseExecutor) ExecuteQuery(ctx context.Context, database domain.Database, query string, args ...any) (*ports.DatabaseQueryResult, error) {
	executor, err := databaseExecutorForType(e, database.Type)
	if err != nil {
		return nil, err
	}
	return executor.ExecuteQuery(ctx, database, query, args...)
}

func (e *DatabaseExecutor) ExecuteCommand(ctx context.Context, database domain.Database, command string, args ...any) (*ports.DatabaseCommandResult, error) {
	executor, err := databaseExecutorForType(e, database.Type)
	if err != nil {
		return nil, err
	}
	return executor.ExecuteCommand(ctx, database, command, args...)
}

func (e *DatabaseExecutor) ExplainQuery(ctx context.Context, database domain.Database, query string, analyze bool, args ...any) (*ports.DatabaseQueryResult, error) {
	executor, err := databaseExecutorForType(e, database.Type)
	if err != nil {
		return nil, err
	}
	return executor.ExplainQuery(ctx, database, query, analyze, args...)
}

func (e *DatabaseExecutor) GetDatabaseMetadata(ctx context.Context, database domain.Database) (*ports.DatabaseMetadata, error) {
	executor, err := databaseExecutorForType(e, database.Type)
	if err != nil {
		return nil, err
	}
	return executor.GetDatabaseMetadata(ctx, database)
}

func databaseExecutorForType(executor *DatabaseExecutor, databaseType string) (ports.DatabaseExecutor, error) {
	switch databaseType {
	case "mysql":
		return executor.mysqlExecutor, nil
	case "postgresql":
		return executor.postgresqlExecutor, nil
	default:
		return nil, errors.New("unsupported database type")
	}
}
