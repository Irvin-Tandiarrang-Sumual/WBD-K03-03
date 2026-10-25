package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type DatabaseExecutor struct {
	poolManager *PoolManager
}

var _ ports.DatabaseExecutor = (*DatabaseExecutor)(nil)

func NewDatabaseExecutor(poolManager *PoolManager) *DatabaseExecutor {
	return &DatabaseExecutor{poolManager: poolManager}
}

func (e *DatabaseExecutor) ExecuteQuery(ctx context.Context, database domain.Database, query string, args ...any) (*ports.DatabaseQueryResult, error) {
	return e.query(ctx, database, query, args...)
}

func (e *DatabaseExecutor) ExecuteCommand(ctx context.Context, database domain.Database, command string, args ...any) (*ports.DatabaseCommandResult, error) {
	db, err := e.poolManager.Get(ctx, database)
	if err != nil {
		return nil, err
	}
	result := db.WithContext(ctx).Exec(command, args...)
	if result.Error != nil {
		return nil, fmt.Errorf("execute PostgreSQL command: %w", result.Error)
	}
	return &ports.DatabaseCommandResult{RowsAffected: result.RowsAffected}, nil
}

func (e *DatabaseExecutor) ExplainQuery(ctx context.Context, database domain.Database, query string, analyze bool, args ...any) (*ports.DatabaseQueryResult, error) {
	prefix := "EXPLAIN "
	if analyze {
		prefix = "EXPLAIN (ANALYZE, BUFFERS) "
	}
	return e.query(ctx, database, prefix+query, args...)
}

func (e *DatabaseExecutor) GetDatabaseMetadata(ctx context.Context, database domain.Database) (*ports.DatabaseMetadata, error) {
	tables, err := e.query(ctx, database, `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name`)
	if err != nil {
		return nil, err
	}
	columns, err := e.query(ctx, database, `
		SELECT table_schema, table_name, column_name, data_type,
		       (is_nullable = 'YES') AS is_nullable
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	indexes, err := e.query(ctx, database, `
		SELECT schemaname, tablename, indexname, indexdef LIKE '%UNIQUE INDEX%' AS is_unique,
		       indexdef LIKE '% PRIMARY KEY%' AS is_primary
		FROM pg_indexes
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, tablename, indexname`)
	if err != nil {
		return nil, err
	}
	return &ports.DatabaseMetadata{
		Tables:  mapTables(tables),
		Columns: mapColumns(columns),
		Indexes: mapIndexes(indexes),
	}, nil
}

func (e *DatabaseExecutor) query(ctx context.Context, database domain.Database, query string, args ...any) (*ports.DatabaseQueryResult, error) {
	db, err := e.poolManager.Get(ctx, database)
	if err != nil {
		return nil, err
	}
	rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL database: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

func scanRows(rows *sql.Rows) (*ports.DatabaseQueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := &ports.DatabaseQueryResult{Columns: columns, Rows: make([]map[string]any, 0)}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for index, column := range columns {
			row[column] = normalizeValue(values[index])
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeValue(value any) any {
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return value
}

func mapTables(result *ports.DatabaseQueryResult) []ports.DatabaseTableMetadata {
	tables := make([]ports.DatabaseTableMetadata, 0, len(result.Rows))
	for _, row := range result.Rows {
		tables = append(tables, ports.DatabaseTableMetadata{
			Schema: stringValue(row["table_schema"]),
			Name:   stringValue(row["table_name"]),
		})
	}
	return tables
}

func mapColumns(result *ports.DatabaseQueryResult) []ports.DatabaseColumnMetadata {
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
	return columns
}

func mapIndexes(result *ports.DatabaseQueryResult) []ports.DatabaseIndexMetadata {
	indexes := make([]ports.DatabaseIndexMetadata, 0, len(result.Rows))
	for _, row := range result.Rows {
		indexes = append(indexes, ports.DatabaseIndexMetadata{
			Schema:    stringValue(row["schemaname"]),
			Table:     stringValue(row["tablename"]),
			Name:      stringValue(row["indexname"]),
			IsUnique:  boolValue(row["is_unique"]),
			IsPrimary: boolValue(row["is_primary"]),
		})
	}
	return indexes
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func boolValue(value any) bool {
	if value, ok := value.(bool); ok {
		return value
	}
	return false
}
