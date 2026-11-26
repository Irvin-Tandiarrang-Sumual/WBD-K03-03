package driver

import (
	"context"
	"errors"
	"regexp"

	mysqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/mysql"
	postgresqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	mysqlclient "github.com/go-sql-driver/mysql"
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

func (m *DatabasePoolManager) CollectMetrics() {
	if m.postgresqlPoolManager != nil {
		active, idle, open, wait := m.postgresqlPoolManager.CollectStats()
		metrics.DBPoolConnectionsActive.WithLabelValues("postgresql").Set(float64(active))
		metrics.DBPoolConnectionsIdle.WithLabelValues("postgresql").Set(float64(idle))
		metrics.DBPoolConnectionsOpen.WithLabelValues("postgresql").Set(float64(open))
		metrics.DBPoolWaitDuration.WithLabelValues("postgresql").Set(wait)
	}
	if m.mysqlPoolManager != nil {
		active, idle, open, wait := m.mysqlPoolManager.CollectStats()
		metrics.DBPoolConnectionsActive.WithLabelValues("mysql").Set(float64(active))
		metrics.DBPoolConnectionsIdle.WithLabelValues("mysql").Set(float64(idle))
		metrics.DBPoolConnectionsOpen.WithLabelValues("mysql").Set(float64(open))
		metrics.DBPoolWaitDuration.WithLabelValues("mysql").Set(wait)
	}
}

type DatabaseDriver struct {
	poolManager        *DatabasePoolManager
	postgresqlExecutor *postgresqldriver.ExecutorRepository
	postgresqlDialect  *postgresqldriver.SQLDialect

	mysqlExecutor *mysqldriver.ExecutorRepository
	mysqlDialect  *mysqldriver.SQLDialect
}

var _ ports.DatabaseDriver = (*DatabaseDriver)(nil)

func (d *DatabaseDriver) ValidateMigrationSQL(ctx context.Context, db domain.Database, script string) error {
	if db.Type == domain.DatabaseTypePostgreSQL {
		_, err := observeSQL(ctx, db, "validate", script, func(ctx context.Context) (struct{}, int64, error) {
			return struct{}{}, 0, d.postgresqlExecutor.ValidateMigrationSQL(ctx, db, script)
		})
		return err
	}
	return nil
}

func (d *DatabaseDriver) PrepareBatch(ctx context.Context, db domain.Database, batch ports.BatchExecution) (int64, error) {
	if db.Type != domain.DatabaseTypePostgreSQL {
		return 0, domain.ErrInvalidDraftInput
	}
	return observeSQL(ctx, db, "batch.prepare", batch.Statement, func(ctx context.Context) (int64, int64, error) {
		count, err := d.postgresqlExecutor.PrepareBatch(ctx, db, batch)
		return count, count, err
	})
}

func (d *DatabaseDriver) ExecuteBatchChunk(ctx context.Context, db domain.Database, batch ports.BatchExecution, number int, revert bool) (*ports.BatchReceipt, error) {
	if db.Type != domain.DatabaseTypePostgreSQL {
		return nil, domain.ErrInvalidDraftInput
	}
	return observeSQL(ctx, db, "batch.chunk", batch.Statement, func(ctx context.Context) (*ports.BatchReceipt, int64, error) {
		receipt, err := d.postgresqlExecutor.ExecuteBatchChunk(ctx, db, batch, number, revert)
		if err != nil || receipt == nil {
			return receipt, 0, err
		}
		return receipt, receipt.Rows, nil
	}, "batch_number", number, "revert", revert)
}

func NewDatabaseDriver(poolManager *DatabasePoolManager) *DatabaseDriver {
	pgExecutor := postgresqldriver.NewExecutorRepository(poolManager.postgresqlPoolManager)
	myExecutor := mysqldriver.NewExecutorRepository(poolManager.mysqlPoolManager)

	return &DatabaseDriver{
		poolManager:        poolManager,
		postgresqlExecutor: pgExecutor,
		postgresqlDialect:  postgresqldriver.NewSQLDialect(pgExecutor),

		mysqlExecutor: myExecutor,
		mysqlDialect:  mysqldriver.NewSQLDialect(myExecutor),
	}
}

func (d *DatabaseDriver) ExecuteQuery(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	return observeSQL(ctx, database, "query", query, func(ctx context.Context) (*ports.DatabaseQueryResult, int64, error) {
		executor, err := d.executorForType(database.Type)
		if err != nil {
			return nil, 0, err
		}
		result, err := executor.ExecuteQuery(ctx, database, query)
		if err != nil || result == nil {
			return result, 0, err
		}
		if d.poolManager != nil {
			d.poolManager.CollectMetrics()
		}
		return result, int64(len(result.Rows)), nil
	})
}

func (d *DatabaseDriver) ExecuteCommand(ctx context.Context, database domain.Database, command string) (*ports.DatabaseCommandResult, error) {
	return observeSQL(ctx, database, "command", command, func(ctx context.Context) (*ports.DatabaseCommandResult, int64, error) {
		executor, err := d.executorForType(database.Type)
		if err != nil {
			return nil, 0, err
		}
		result, err := executor.ExecuteCommand(ctx, database, command)
		if err != nil || result == nil {
			return result, 0, err
		}
		if d.poolManager != nil {
			d.poolManager.CollectMetrics()
		}
		return result, result.RowsAffected, nil
	})
}

var sqlStateRegex = regexp.MustCompile(`(?i)(?:SQLSTATE|code)\s*[:=]?\s*([0-9A-Z]{5})`)

type sqlStateHolder interface {
	SQLState() string
}

type codeHolder interface {
	Code() string
}

func extractSQLState(err error) string {
	if err == nil {
		return ""
	}
	var sh sqlStateHolder
	if errors.As(err, &sh) {
		return sh.SQLState()
	}
	var myErr *mysqlclient.MySQLError
	if errors.As(err, &myErr) && myErr.SQLState != [5]byte{} {
		return string(myErr.SQLState[:])
	}
	var ch codeHolder
	if errors.As(err, &ch) {
		return ch.Code()
	}
	if match := sqlStateRegex.FindStringSubmatch(err.Error()); len(match) > 1 {
		return match[1]
	}
	return ""
}

func (d *DatabaseDriver) Explain(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	return observeSQL(ctx, database, "explain", query, func(ctx context.Context) (*ports.DatabaseQueryResult, int64, error) {
		executor, err := d.executorForType(database.Type)
		if err != nil {
			return nil, 0, err
		}
		result, err := executor.Explain(ctx, database, query)
		if err != nil || result == nil {
			return result, 0, err
		}
		if d.poolManager != nil {
			d.poolManager.CollectMetrics()
		}
		return result, int64(len(result.Rows)), nil
	})
}

func (d *DatabaseDriver) ExplainAnalyze(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	return observeSQL(ctx, database, "explain_analyze", query, func(ctx context.Context) (*ports.DatabaseQueryResult, int64, error) {
		executor, err := d.executorForType(database.Type)
		if err != nil {
			return nil, 0, err
		}
		result, err := executor.ExplainAnalyze(ctx, database, query)
		if err != nil || result == nil {
			return result, 0, err
		}
		if d.poolManager != nil {
			d.poolManager.CollectMetrics()
		}
		return result, int64(len(result.Rows)), nil
	})
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
