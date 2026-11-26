package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
)

type ExecutorRepository struct {
	poolManager *PoolManager
}

var _ ports.DatabaseExecutorRepository = (*ExecutorRepository)(nil)
var _ ports.SchemaInspectorRepository = (*ExecutorRepository)(nil)

func NewExecutorRepository(poolManager *PoolManager) *ExecutorRepository {
	return &ExecutorRepository{poolManager: poolManager}
}

func (r *ExecutorRepository) ExecuteQuery(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	return r.query(ctx, database, query)
}

func (r *ExecutorRepository) ExecuteCommand(ctx context.Context, database domain.Database, command string) (*ports.DatabaseCommandResult, error) {
	db, err := r.poolManager.Get(ctx, database)
	if err != nil {
		return nil, err
	}
	result := db.WithContext(ctx).Exec(command)
	if result.Error != nil {
		return nil, fmt.Errorf("execute PostgreSQL command: %w", result.Error)
	}
	return &ports.DatabaseCommandResult{RowsAffected: result.RowsAffected}, nil
}

func (r *ExecutorRepository) Explain(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	return r.query(ctx, database, "EXPLAIN "+query)
}

func (r *ExecutorRepository) ExplainAnalyze(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	return r.query(ctx, database, "EXPLAIN (ANALYZE, BUFFERS) "+query)
}

func (r *ExecutorRepository) GetDatabaseMetadata(ctx context.Context, database domain.Database) (*ports.DatabaseMetadata, error) {
	tables, err := r.inspectTables(ctx, database)
	if err != nil {
		return nil, err
	}

	columns, err := r.inspectColumns(ctx, database)
	if err != nil {
		return nil, err
	}

	indexes, err := r.inspectIndexes(ctx, database)
	if err != nil {
		return nil, err
	}

	return &ports.DatabaseMetadata{
		Tables:  tables,
		Columns: columns,
		Indexes: indexes,
	}, nil
}

func (r *ExecutorRepository) inspectTables(ctx context.Context, database domain.Database) ([]ports.DatabaseTableMetadata, error) {
	query := `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema', 'wbd_migration')
		ORDER BY table_name`

	result, err := r.query(ctx, database, query)
	if err != nil {
		return nil, err
	}

	tables := make([]ports.DatabaseTableMetadata, 0, len(result.Rows))
	for _, row := range result.Rows {
		tables = append(tables, ports.DatabaseTableMetadata{
			Schema: stringValue(row["table_schema"]),
			Name:   stringValue(row["table_name"]),
		})
	}
	return tables, nil
}

func (r *ExecutorRepository) inspectColumns(ctx context.Context, database domain.Database) ([]ports.DatabaseColumnMetadata, error) {
	query := `
		SELECT table_schema, table_name, column_name, data_type,
		       (is_nullable = 'YES') AS is_nullable
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'wbd_migration')
		ORDER BY table_name, ordinal_position`

	result, err := r.query(ctx, database, query)
	if err != nil {
		return nil, err
	}

	columns := make([]ports.DatabaseColumnMetadata, 0, len(result.Rows))
	for _, row := range result.Rows {
		columns = append(columns, ports.DatabaseColumnMetadata{
			Schema:     stringValue(row["table_schema"]),
			Table:      stringValue(row["table_name"]),
			Name:       stringValue(row["column_name"]),
			DataType:   stringValue(row["data_type"]),
			IsNullable: boolValue(row["is_nullable"]),
		})
	}
	return columns, nil
}

func (r *ExecutorRepository) inspectIndexes(ctx context.Context, database domain.Database) ([]ports.DatabaseIndexMetadata, error) {
	query := `
		SELECT schemaname AS table_schema, tablename AS table_name, indexname AS index_name,
		       (indexdef ILIKE '%UNIQUE INDEX%') AS is_unique,
		       (indexname LIKE '%_pkey') AS is_primary
		FROM pg_indexes
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema', 'wbd_migration')
		ORDER BY tablename, indexname`

	result, err := r.query(ctx, database, query)
	if err != nil {
		return nil, err
	}

	indexes := make([]ports.DatabaseIndexMetadata, 0, len(result.Rows))
	for _, row := range result.Rows {
		indexes = append(indexes, ports.DatabaseIndexMetadata{
			Schema:    stringValue(row["table_schema"]),
			Table:     stringValue(row["table_name"]),
			Name:      stringValue(row["index_name"]),
			IsUnique:  boolValue(row["is_unique"]),
			IsPrimary: boolValue(row["is_primary"]),
		})
	}
	return indexes, nil
}

func (r *ExecutorRepository) query(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	db, err := r.poolManager.Get(ctx, database)
	if err != nil {
		return nil, err
	}

	tx := db.WithContext(ctx).Begin(&sql.TxOptions{ReadOnly: true})
	if tx.Error != nil {
		tx = db.WithContext(ctx).Begin()
	}
	if tx.Error != nil {
		return nil, fmt.Errorf("begin transaction: %w", tx.Error)
	}
	defer logger.ReadTransactionStarted(ctx, database.ID, func() error { return tx.Rollback().Error })()

	rows, err := tx.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL database: %w", err)
	}
	defer rows.Close()

	return scanQueryResult(rows)
}

func scanQueryResult(rows *sql.Rows) (*ports.DatabaseQueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		rowMap, err := scanSingleRow(rows, columns)
		if err != nil {
			return nil, err
		}
		resultRows = append(resultRows, rowMap)
	}

	return &ports.DatabaseQueryResult{
		Columns: columns,
		Rows:    resultRows,
	}, nil
}

func scanSingleRow(rows *sql.Rows, columns []string) (map[string]any, error) {
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	if err := rows.Scan(pointers...); err != nil {
		return nil, err
	}

	rowMap := make(map[string]any, len(columns))
	for i, col := range columns {
		if b, ok := values[i].([]byte); ok {
			rowMap[col] = string(b)
		} else {
			rowMap[col] = values[i]
		}
	}
	return rowMap, nil
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case sql.NullString:
		if v.Valid {
			return v.String
		}
		return ""
	default:
		return ""
	}
}

func boolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int64:
		return v == 1
	case int:
		return v == 1
	case []byte:
		return string(v) == "1"
	case string:
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
		return v == "1"
	default:
		return false
	}
}
