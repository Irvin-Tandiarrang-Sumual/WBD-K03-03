package database_executor

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

type databaseServiceFake struct {
	ports.DatabaseService
	database *domain.Database
	err      error
}

func (f *databaseServiceFake) GetDatabaseByIDAndUserID(context.Context, uint, uint) (*domain.Database, error) {
	return f.database, f.err
}

type executorFake struct {
	ports.DatabaseExecutor
	queryResult *ports.DatabaseQueryResult
	explained   bool
	query       string
	database    domain.Database
	analyze     bool
}

func (f *executorFake) ExecuteQuery(_ context.Context, database domain.Database, query string, _ ...any) (*ports.DatabaseQueryResult, error) {
	f.database = database
	f.query = query
	return f.queryResult, nil
}

func (f *executorFake) ExplainQuery(_ context.Context, database domain.Database, query string, analyze bool, _ ...any) (*ports.DatabaseQueryResult, error) {
	f.database = database
	f.query = query
	f.analyze = analyze
	f.explained = true
	return f.queryResult, nil
}

func TestExecuteDatabaseQueryChecksAccessAndDelegates(t *testing.T) {
	executor := &executorFake{queryResult: &ports.DatabaseQueryResult{Columns: []string{"id"}}}
	service := NewDatabaseExecutorService(
		&databaseServiceFake{database: &domain.Database{ID: 5}},
		&testutil.DatabaseRepository{
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 5, Password: "secret"}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewOnly, nil
			},
		},
	)

	result, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")
	if err != nil {
		t.Fatalf("ExecuteDatabaseQuery() error = %v", err)
	}
	if result != executor.queryResult || executor.query != "SELECT 1" || executor.database.Password != "secret" {
		t.Fatalf("unexpected delegation: result=%v query=%q database=%+v", result, executor.query, executor.database)
	}
}

func TestExecuteDatabaseQueryRejectsViewOnlyMismatch(t *testing.T) {
	service := NewDatabaseExecutorService(
		&databaseServiceFake{database: &domain.Database{ID: 5}},
		&testutil.DatabaseRepository{},
		&executorFake{},
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessType(""), nil
			},
		},
	)

	_, err := service.ExecuteDatabaseQuery(context.Background(), 5, 9, "SELECT 1")
	if !errors.Is(err, domain.ErrDatabaseAccessDenied) {
		t.Fatalf("error = %v, want access denied", err)
	}
}

func TestExplainDatabaseQueryPassesAnalyzeFlag(t *testing.T) {
	executor := &executorFake{queryResult: &ports.DatabaseQueryResult{}}
	service := NewDatabaseExecutorService(
		&databaseServiceFake{database: &domain.Database{ID: 5}},
		&testutil.DatabaseRepository{
			FindByIdFunc: func(context.Context, uint) (*domain.Database, error) {
				return &domain.Database{ID: 5}, nil
			},
		},
		executor,
		&testutil.GroupService{
			EffectiveDatabaseAccessFunc: func(context.Context, uint, uint) (domain.AccessType, error) {
				return domain.AccessViewEdit, nil
			},
		},
	)

	_, err := service.ExplainDatabaseQuery(context.Background(), 5, 9, "WITH x AS (SELECT 1) SELECT * FROM x", true)
	if err != nil {
		t.Fatalf("ExplainDatabaseQuery() error = %v", err)
	}
	if !executor.explained || !executor.analyze {
		t.Fatalf("explain delegation lost analyze flag: %+v", executor)
	}
}

func TestValidateSelectQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{name: "select query", query: "SELECT * FROM users", wantErr: false},
		{name: "with query", query: "WITH users AS (SELECT 1) SELECT * FROM users", wantErr: false},
		{name: "case insensitive", query: "  select id from users", wantErr: false},
		{name: "empty query", query: "   ", wantErr: true},
		{name: "mutation query", query: "UPDATE users SET active = true", wantErr: true},
		{name: "multiple statements", query: "SELECT 1; SELECT 2", wantErr: true},
		{name: "select without separator", query: "selection", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSelectQuery(test.query)
			if test.wantErr && err != domain.ErrInvalidDatabaseQuery {
				t.Fatalf("got error %v, want %v", err, domain.ErrInvalidDatabaseQuery)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
