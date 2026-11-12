package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type DatabaseExecutorRepository struct {
	ports.DatabaseExecutorRepository
	ExecuteQueryFunc   func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error)
	ExecuteCommandFunc func(context.Context, domain.Database, string) (*ports.DatabaseCommandResult, error)
	ExplainFunc        func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error)
	ExplainAnalyzeFunc func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error)
}

func (f *DatabaseExecutorRepository) ExecuteQuery(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	if f.ExecuteQueryFunc != nil {
		return f.ExecuteQueryFunc(ctx, database, query)
	}
	return nil, nil
}

func (f *DatabaseExecutorRepository) ExecuteCommand(ctx context.Context, database domain.Database, command string) (*ports.DatabaseCommandResult, error) {
	if f.ExecuteCommandFunc != nil {
		return f.ExecuteCommandFunc(ctx, database, command)
	}
	return nil, nil
}

func (f *DatabaseExecutorRepository) Explain(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	if f.ExplainFunc != nil {
		return f.ExplainFunc(ctx, database, query)
	}
	return nil, nil
}

func (f *DatabaseExecutorRepository) ExplainAnalyze(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	if f.ExplainAnalyzeFunc != nil {
		return f.ExplainAnalyzeFunc(ctx, database, query)
	}
	return nil, nil
}

type DatabaseDriver struct {
	ports.DatabaseDriver
	ExecuteQueryFunc        func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error)
	ExecuteCommandFunc      func(context.Context, domain.Database, string) (*ports.DatabaseCommandResult, error)
	ExplainFunc             func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error)
	ExplainAnalyzeFunc      func(context.Context, domain.Database, string) (*ports.DatabaseQueryResult, error)
	GetDatabaseMetadataFunc func(context.Context, domain.Database) (*ports.DatabaseMetadata, error)
	GenerateSQLDownFunc     func(context.Context, domain.Database, string) (string, error)
}

func (f *DatabaseDriver) ExecuteQuery(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	if f.ExecuteQueryFunc != nil {
		return f.ExecuteQueryFunc(ctx, database, query)
	}
	return nil, nil
}

func (f *DatabaseDriver) ExecuteCommand(ctx context.Context, database domain.Database, command string) (*ports.DatabaseCommandResult, error) {
	if f.ExecuteCommandFunc != nil {
		return f.ExecuteCommandFunc(ctx, database, command)
	}
	return nil, nil
}

func (f *DatabaseDriver) Explain(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	if f.ExplainFunc != nil {
		return f.ExplainFunc(ctx, database, query)
	}
	return nil, nil
}

func (f *DatabaseDriver) ExplainAnalyze(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
	if f.ExplainAnalyzeFunc != nil {
		return f.ExplainAnalyzeFunc(ctx, database, query)
	}
	return nil, nil
}

func (f *DatabaseDriver) GetDatabaseMetadata(ctx context.Context, database domain.Database) (*ports.DatabaseMetadata, error) {
	if f.GetDatabaseMetadataFunc != nil {
		return f.GetDatabaseMetadataFunc(ctx, database)
	}
	return nil, nil
}

func (f *DatabaseDriver) GenerateSQLDown(ctx context.Context, database domain.Database, sqlUp string) (string, error) {
	if f.GenerateSQLDownFunc != nil {
		return f.GenerateSQLDownFunc(ctx, database, sqlUp)
	}
	return "", nil
}
