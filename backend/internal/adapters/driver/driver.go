package driver

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"time"

	mysqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/mysql"
	postgresqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
		active, idle, open, _ := m.postgresqlPoolManager.CollectStats()
		metrics.DBPoolConnectionsActive.WithLabelValues("postgresql").Set(float64(active))
		metrics.DBPoolConnectionsIdle.WithLabelValues("postgresql").Set(float64(idle))
		metrics.DBPoolConnectionsOpen.WithLabelValues("postgresql").Set(float64(open))
	}
	if m.mysqlPoolManager != nil {
		active, idle, open, _ := m.mysqlPoolManager.CollectStats()
		metrics.DBPoolConnectionsActive.WithLabelValues("mysql").Set(float64(active))
		metrics.DBPoolConnectionsIdle.WithLabelValues("mysql").Set(float64(idle))
		metrics.DBPoolConnectionsOpen.WithLabelValues("mysql").Set(float64(open))
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

func (d *DatabaseDriver) PrepareBatch(ctx context.Context, db domain.Database, batch ports.BatchExecution) (int64, error) {
	if db.Type != domain.DatabaseTypePostgreSQL {
		return 0, domain.ErrInvalidDraftInput
	}
	return d.postgresqlExecutor.PrepareBatch(ctx, db, batch)
}

func (d *DatabaseDriver) ExecuteBatchChunk(ctx context.Context, db domain.Database, batch ports.BatchExecution, number int, revert bool) (*ports.BatchReceipt, error) {
	if db.Type != domain.DatabaseTypePostgreSQL {
		return nil, domain.ErrInvalidDraftInput
	}
	return d.postgresqlExecutor.ExecuteBatchChunk(ctx, db, batch, number, revert)
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
	tr := telemetry.Tracer("db.driver")
	ctx, span := tr.Start(ctx, "db.query",
		trace.WithAttributes(
			attribute.String("db.system", database.Type),
			attribute.Int("db.id", int(database.ID)),
		),
	)
	defer span.End()

	slog.DebugContext(ctx, "SQL query started",
		"event", "sql.query.start",
		"database_id", database.ID,
		"database_dialect", database.Type,
		"query", query,
	)

	start := time.Now()
	executor, err := d.executorForType(database.Type)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "SQL query failed: unsupported database type",
			"event", "sql.query.failed",
			"database_id", database.ID,
			"database_dialect", database.Type,
			"duration_ms", float64(time.Since(start).Nanoseconds())/1e6,
			"error", err.Error(),
		)
		return nil, err
	}

	result, err := executor.ExecuteQuery(ctx, database, query)
	durationMs := float64(time.Since(start).Nanoseconds()) / 1e6

	if err != nil {
		sqlState := extractSQLState(err)
		span.RecordError(err)
		if sqlState != "" {
			span.SetAttributes(attribute.String("db.response.status_code", sqlState))
		}
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "SQL query failed",
			"event", "sql.query.failed",
			"database_id", database.ID,
			"database_dialect", database.Type,
			"duration_ms", durationMs,
			"query", query,
			"error", err.Error(),
			"sqlstate", sqlState,
		)
		return nil, err
	}

	rowCount := 0
	if result != nil {
		rowCount = len(result.Rows)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", rowCount))
	span.SetStatus(codes.Ok, "success")

	slog.InfoContext(ctx, "SQL query completed",
		"event", "sql.query.complete",
		"database_id", database.ID,
		"database_dialect", database.Type,
		"duration_ms", durationMs,
		"query", query,
		"rows_returned", rowCount,
	)
	if d.poolManager != nil {
		d.poolManager.CollectMetrics()
	}
	return result, nil
}

func (d *DatabaseDriver) ExecuteCommand(ctx context.Context, database domain.Database, command string) (*ports.DatabaseCommandResult, error) {
	tr := telemetry.Tracer("db.driver")
	ctx, span := tr.Start(ctx, "db.command",
		trace.WithAttributes(
			attribute.String("db.system", database.Type),
			attribute.Int("db.id", int(database.ID)),
		),
	)
	defer span.End()

	slog.DebugContext(ctx, "SQL command started",
		"event", "sql.command.start",
		"database_id", database.ID,
		"database_dialect", database.Type,
		"query", command,
	)

	start := time.Now()
	executor, err := d.executorForType(database.Type)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "SQL command failed: unsupported database type",
			"event", "sql.command.failed",
			"database_id", database.ID,
			"database_dialect", database.Type,
			"duration_ms", float64(time.Since(start).Nanoseconds())/1e6,
			"error", err.Error(),
		)
		return nil, err
	}

	result, err := executor.ExecuteCommand(ctx, database, command)
	durationMs := float64(time.Since(start).Nanoseconds()) / 1e6

	if err != nil {
		sqlState := extractSQLState(err)
		span.RecordError(err)
		if sqlState != "" {
			span.SetAttributes(attribute.String("db.response.status_code", sqlState))
		}
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "SQL command failed",
			"event", "sql.command.failed",
			"database_id", database.ID,
			"database_dialect", database.Type,
			"duration_ms", durationMs,
			"query", command,
			"error", err.Error(),
			"sqlstate", sqlState,
		)
		return nil, err
	}

	rowsAffected := int64(0)
	if result != nil {
		rowsAffected = result.RowsAffected
	}
	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	span.SetStatus(codes.Ok, "success")

	slog.InfoContext(ctx, "SQL command completed",
		"event", "sql.command.complete",
		"database_id", database.ID,
		"database_dialect", database.Type,
		"duration_ms", durationMs,
		"query", command,
		"rows_affected", rowsAffected,
	)
	if d.poolManager != nil {
		d.poolManager.CollectMetrics()
	}
	return result, nil
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
