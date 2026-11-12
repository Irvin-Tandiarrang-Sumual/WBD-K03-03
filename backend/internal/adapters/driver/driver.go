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

type DatabaseDriver struct {
	postgresqlExecutor *postgresqldriver.ExecutorRepository
	postgresqlDialect  *postgresqldriver.SQLDialect

	mysqlExecutor *mysqldriver.ExecutorRepository
	mysqlDialect  *mysqldriver.SQLDialect
}

var _ ports.DatabaseDriver = (*DatabaseDriver)(nil)

func NewDatabaseDriver(poolManager *DatabasePoolManager) *DatabaseDriver {
	pgExecutor := postgresqldriver.NewExecutorRepository(poolManager.postgresqlPoolManager)
	myExecutor := mysqldriver.NewExecutorRepository(poolManager.mysqlPoolManager)

	return &DatabaseDriver{
		postgresqlExecutor: pgExecutor,
		postgresqlDialect:  postgresqldriver.NewSQLDialect(pgExecutor),

		mysqlExecutor: myExecutor,
		mysqlDialect:  mysqldriver.NewSQLDialect(myExecutor),
	}
}

func (d *DatabaseDriver) ExecuteQuery(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	executor, err := d.executorForType(database.Type)
	if err != nil {
		return nil, err
	}
	return executor.ExecuteQuery(ctx, database, query)
}

func (d *DatabaseDriver) ExecuteCommand(ctx context.Context, database domain.Database, command string) (*ports.DatabaseCommandResult, error) {
	executor, err := d.executorForType(database.Type)
	if err != nil {
		return nil, err
	}
	return executor.ExecuteCommand(ctx, database, command)
}

func (d *DatabaseDriver) Explain(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	executor, err := d.executorForType(database.Type)
	if err != nil {
		return nil, err
	}
	return executor.Explain(ctx, database, query)
}

func (d *DatabaseDriver) ExplainAnalyze(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	executor, err := d.executorForType(database.Type)
	if err != nil {
		return nil, err
	}
	return executor.ExplainAnalyze(ctx, database, query)
}

func (d *DatabaseDriver) GetDatabaseMetadata(ctx context.Context, database domain.Database) (*ports.DatabaseMetadata, error) {
	inspector, err := d.inspectorForType(database.Type)
	if err != nil {
		return nil, err
	}
	return inspector.GetDatabaseMetadata(ctx, database)
}

func (d *DatabaseDriver) GenerateSQLDown(ctx context.Context, database domain.Database, sqlUp string) (string, error) {
	dialect, err := d.dialectForType(database.Type)
	if err != nil {
		return "", err
	}
	return dialect.GenerateSQLDown(ctx, database, sqlUp)
}

func (d *DatabaseDriver) executorForType(databaseType string) (ports.DatabaseExecutorRepository, error) {
	switch databaseType {
	case "mysql":
		return d.mysqlExecutor, nil
	case "postgresql":
		return d.postgresqlExecutor, nil
	default:
		return nil, errors.New("unsupported database type")
	}
}

func (d *DatabaseDriver) inspectorForType(databaseType string) (ports.SchemaInspectorRepository, error) {
	switch databaseType {
	case "mysql":
		return d.mysqlExecutor, nil
	case "postgresql":
		return d.postgresqlExecutor, nil
	default:
		return nil, errors.New("unsupported database type")
	}
}

func (d *DatabaseDriver) dialectForType(databaseType string) (ports.SQLDialect, error) {
	switch databaseType {
	case "mysql":
		return d.mysqlDialect, nil
	case "postgresql":
		return d.postgresqlDialect, nil
	default:
		return nil, errors.New("unsupported database type")
	}
}
