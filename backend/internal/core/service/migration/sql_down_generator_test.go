package migration

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type generatorExecutor struct {
	result *ports.DatabaseQueryResult
	err    error
	query  string
}

func (f *generatorExecutor) ExecuteQuery(
	_ context.Context,
	_ domain.Database,
	query string,
	_ ...any,
) (*ports.DatabaseQueryResult, error) {
	f.query = query
	return f.result, f.err
}

func (f *generatorExecutor) ExecuteCommand(context.Context, domain.Database, string, ...any) (*ports.DatabaseCommandResult, error) {
	return nil, nil
}

func (f *generatorExecutor) ExplainQuery(context.Context, domain.Database, string, bool, ...any) (*ports.DatabaseQueryResult, error) {
	return nil, nil
}

func (f *generatorExecutor) GetDatabaseMetadata(context.Context, domain.Database) (*ports.DatabaseMetadata, error) {
	return nil, nil
}

func TestGenerateSQLDownForDDLAndInsert(t *testing.T) {
	database := domain.Database{Type: "postgresql"}
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "create table",
			sql:  "CREATE TABLE users (id integer);",
			want: "DROP TABLE users",
		},
		{
			name: "create index",
			sql:  "CREATE UNIQUE INDEX users_email_idx ON users (email)",
			want: "DROP INDEX users_email_idx",
		},
		{
			name: "add column",
			sql:  "ALTER TABLE users ADD COLUMN active boolean",
			want: "ALTER TABLE users DROP COLUMN active",
		},
		{
			name: "insert",
			sql:  "INSERT INTO users (id, name) VALUES (1, 'Ada')",
			want: "DELETE FROM users WHERE id = 1 AND name = 'Ada'",
		},
		{
			name: "multiple statements reversed in sql_down",
			sql:  "CREATE TABLE users (id integer, name VARCHAR(50)); ALTER TABLE users ADD COLUMN email VARCHAR(100); INSERT INTO users (id, name, email) VALUES (1, 'Ada;Grace', 'ada@example.com');",
			want: "DELETE FROM users WHERE id = 1 AND name = 'Ada;Grace' AND email = 'ada@example.com';\nALTER TABLE users DROP COLUMN email;\nDROP TABLE users",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := generateSQLDown(context.Background(), &generatorExecutor{}, database, test.sql)
			if err != nil {
				t.Fatalf("generateSQLDown returned error: %v", err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestGenerateSQLDownForMySQLIndex(t *testing.T) {
	got, err := generateSQLDown(
		context.Background(),
		&generatorExecutor{},
		domain.Database{Type: "mysql"},
		"CREATE INDEX users_email_idx ON users (email)",
	)
	if err != nil {
		t.Fatalf("generateSQLDown returned error: %v", err)
	}
	if got != "DROP INDEX users_email_idx ON users" {
		t.Fatalf("got %q, want MySQL drop-index syntax", got)
	}
}

func TestGenerateSQLDownForDeleteAndUpdateUsesSnapshotQuery(t *testing.T) {
	executor := &generatorExecutor{
		result: &ports.DatabaseQueryResult{
			Columns: []string{"id", "name", "active"},
			Rows: []map[string]any{
				{"id": int64(1), "name": "Ada", "active": true},
			},
		},
	}
	database := domain.Database{Type: "postgresql"}

	deleteDown, err := generateSQLDown(
		context.Background(),
		executor,
		database,
		"DELETE FROM users WHERE id = 1",
	)
	if err != nil {
		t.Fatalf("delete generation returned error: %v", err)
	}
	if deleteDown != "INSERT INTO users (id, name, active) VALUES (1, 'Ada', TRUE)" {
		t.Fatalf("unexpected delete inverse: %q", deleteDown)
	}
	if executor.query != "SELECT * FROM users WHERE id = 1" {
		t.Fatalf("unexpected snapshot query: %q", executor.query)
	}

	updateDown, err := generateSQLDown(
		context.Background(),
		executor,
		database,
		"UPDATE users SET active = false WHERE id = 1",
	)
	if err != nil {
		t.Fatalf("update generation returned error: %v", err)
	}
	if updateDown == "" {
		t.Fatal("expected update inverse SQL")
	}
}

func TestGenerateSQLDownRejectsUnsupportedQueriesAndPropagatesSnapshotError(t *testing.T) {
	unsupported, err := generateSQLDown(
		context.Background(),
		&generatorExecutor{},
		domain.Database{},
		"SELECT * FROM users",
	)
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || unsupported != "" {
		t.Fatalf("unexpected select result: sql=%q err=%v", unsupported, err)
	}

	expected := errors.New("snapshot failed")
	_, err = generateSQLDown(
		context.Background(),
		&generatorExecutor{err: expected},
		domain.Database{},
		"DELETE FROM users WHERE id = 1",
	)
	if !errors.Is(err, expected) {
		t.Fatalf("got error %v, want %v", err, expected)
	}
}
