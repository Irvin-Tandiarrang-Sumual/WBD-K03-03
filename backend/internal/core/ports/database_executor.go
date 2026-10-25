package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type DatabaseQueryResult struct {
	Columns []string
	Rows    []map[string]any
}

type DatabaseCommandResult struct {
	RowsAffected int64
}

type DatabaseTableMetadata struct {
	Schema string
	Name   string
}

type DatabaseColumnMetadata struct {
	Schema     string
	Table      string
	Name       string
	DataType   string
	IsNullable bool
}

type DatabaseIndexMetadata struct {
	Schema    string
	Table     string
	Name      string
	IsUnique  bool
	IsPrimary bool
}

type DatabaseMetadata struct {
	Tables  []DatabaseTableMetadata
	Columns []DatabaseColumnMetadata
	Indexes []DatabaseIndexMetadata
}

type DatabaseExecutorService interface {
	ExecuteDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string) (*DatabaseQueryResult, error)
	ExplainDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string, analyze bool) (*DatabaseQueryResult, error)
}

type DatabaseExecutor interface {
	ExecuteQuery(ctx context.Context, database domain.Database, query string, args ...any) (*DatabaseQueryResult, error)
	ExecuteCommand(ctx context.Context, database domain.Database, command string, args ...any) (*DatabaseCommandResult, error)
	ExplainQuery(ctx context.Context, database domain.Database, query string, analyze bool, args ...any) (*DatabaseQueryResult, error)
	GetDatabaseMetadata(ctx context.Context, database domain.Database) (*DatabaseMetadata, error)
}
