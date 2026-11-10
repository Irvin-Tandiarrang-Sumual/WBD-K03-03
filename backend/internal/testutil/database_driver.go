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
