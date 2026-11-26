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
	RowsAffected int64 `json:"rowsAffected"`
}

type DatabaseTableMetadata struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type DatabaseColumnMetadata struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Name       string `json:"name"`
	DataType   string `json:"dataType"`
	IsNullable bool   `json:"isNullable"`
}

type DatabaseIndexMetadata struct {
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Name      string `json:"name"`
	IsUnique  bool   `json:"isUnique"`
	IsPrimary bool   `json:"isPrimary"`
}

type DatabaseMetadata struct {
	Tables  []DatabaseTableMetadata  `json:"tables"`
	Columns []DatabaseColumnMetadata `json:"columns"`
	Indexes []DatabaseIndexMetadata  `json:"indexes"`
}

type DatabaseDriverService interface {
	ExecuteDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string) (*DatabaseQueryResult, error)
	ExplainDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string) (*DatabaseQueryResult, error)
	ExplainAnalyzeDatabaseQuery(ctx context.Context, databaseID uint, userID uint, query string) (*DatabaseQueryResult, error)
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

type BatchExecution struct {
	Key string
	Table string
	Column string
	Statement string
	Size int
	Resume bool
	SourceKey string
	Reverse bool
}

type BatchReceipt struct {
	Number int
	Cursor string
	Rows int64
}

type BatchExecutor interface {
	PrepareBatch(ctx context.Context, database domain.Database, batch BatchExecution) (int64, error)
	ExecuteBatchChunk(ctx context.Context, database domain.Database, batch BatchExecution, number int, revert bool) (*BatchReceipt, error)
}

type MigrationValidator interface {
	ValidateMigrationSQL(ctx context.Context, database domain.Database, script string) error
}

type DatabaseCredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type DatabaseConnectionTester interface {
	Test(ctx context.Context, database domain.Database) error
}

type DatabasePoolInvalidator interface {
	Invalidate(databaseID uint) error
}
