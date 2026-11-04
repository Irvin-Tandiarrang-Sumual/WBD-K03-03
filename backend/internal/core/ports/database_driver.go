package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type DatabaseQueryResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

type DatabaseCommandResult struct {
	RowsAffected int64 `json:"rows_affected"`
}

type DatabaseTableMetadata struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type DatabaseColumnMetadata struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	IsNullable bool   `json:"is_nullable"`
}

type DatabaseIndexMetadata struct {
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Name      string `json:"name"`
	IsUnique  bool   `json:"is_unique"`
	IsPrimary bool   `json:"is_primary"`
}

type DatabaseMetadata struct {
	Tables  []DatabaseTableMetadata  `json:"tables"`
	Columns []DatabaseColumnMetadata `json:"columns"`
	Indexes []DatabaseIndexMetadata  `json:"indexes"`
}

type DatabaseDriverService interface {
	ExecuteDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string) (*DatabaseQueryResult, error)
	ExplainDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string, analyze bool) (*DatabaseQueryResult, error)
}

type DatabaseExecutorRepository interface {
	ExecuteQuery(ctx context.Context, database domain.Database, query string) (*DatabaseQueryResult, error)
	ExecuteCommand(ctx context.Context, database domain.Database, command string) (*DatabaseCommandResult, error)
	Explain(ctx context.Context, database domain.Database, query string) (*DatabaseQueryResult, error)
	ExplainAnalyze(ctx context.Context, database domain.Database, query string) (*DatabaseQueryResult, error)
}

type SchemaInspectorRepository interface {
	GetDatabaseMetadata(ctx context.Context, database domain.Database) (*DatabaseMetadata, error)
}

type SQLDialect interface {
	GenerateSQLDown(ctx context.Context, database domain.Database, sqlUp string) (string, error)
}

type DatabaseDriver interface {
	DatabaseExecutorRepository
	SchemaInspectorRepository
	SQLDialect
}
