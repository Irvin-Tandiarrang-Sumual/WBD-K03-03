//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestIntegrationDatabaseViewerReadsAllowedTargetDatabase(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewOnly)
	ctx := context.Background()

	if err := stack.db.Exec(`DROP TABLE IF EXISTS integration_viewer_items`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stack.db.Exec(`DROP TABLE IF EXISTS integration_viewer_items`).Error
	})
	if err := stack.db.Exec(`CREATE TABLE integration_viewer_items (id integer PRIMARY KEY, name text NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := stack.db.Exec(`INSERT INTO integration_viewer_items (id, name) VALUES (1, 'alpha'), (2, 'beta')`).Error; err != nil {
		t.Fatal(err)
	}

	databases, total, err := stack.databaseService.GetDatabasesByUserID(ctx, stack.user.ID, pagination.NewPaginationParams(1, 10))
	if err != nil {
		t.Fatalf("GetDatabasesByUserID() error=%v", err)
	}
	if total != 1 || len(databases) != 1 || databases[0].ID != stack.target.ID || databases[0].Password != "" {
		t.Fatalf("unexpected accessible databases: total=%d databases=%+v", total, databases)
	}

	result, err := stack.executorService.ExecuteDatabaseQuery(
		ctx,
		stack.target.ID,
		stack.user.ID,
		"SELECT id, name FROM integration_viewer_items ORDER BY id",
	)
	if err != nil {
		t.Fatalf("ExecuteDatabaseQuery() error=%v", err)
	}
	if len(result.Columns) != 2 || len(result.Rows) != 2 {
		t.Fatalf("unexpected result=%+v", result)
	}
	if fmt.Sprint(result.Rows[0]["id"]) != "1" || fmt.Sprint(result.Rows[0]["name"]) != "alpha" {
		t.Fatalf("unexpected first row=%v", result.Rows[0])
	}

	explain, err := stack.executorService.ExplainDatabaseQuery(
		ctx,
		stack.target.ID,
		stack.user.ID,
		"SELECT * FROM integration_viewer_items",
		false,
	)
	if err != nil {
		t.Fatalf("ExplainDatabaseQuery() error=%v", err)
	}
	if len(explain.Rows) == 0 {
		t.Fatal("EXPLAIN returned no rows")
	}
}

func TestIntegrationDatabaseViewerRejectsMutationQuery(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewOnly)
	_, err := stack.executorService.ExecuteDatabaseQuery(
		context.Background(),
		stack.target.ID,
		stack.user.ID,
		"DELETE FROM users",
	)
	if !errors.Is(err, domain.ErrInvalidDatabaseQuery) {
		t.Fatalf("error=%v, want ErrInvalidDatabaseQuery", err)
	}
}

func TestIntegrationViewOnlyUserCannotCreateMigration(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewOnly)
	_, err := stack.migrationService.CreateMigration(
		context.Background(),
		stack.target.ID,
		stack.user.ID,
		"CREATE TABLE integration_denied_table (id integer)",
	)
	if !errors.Is(err, domain.ErrDatabaseAccessDenied) {
		t.Fatalf("error=%v, want ErrDatabaseAccessDenied", err)
	}

	var count int64
	if err := stack.db.Raw(`SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public' AND tablename = 'integration_denied_table'`).Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("denied migration unexpectedly changed target database")
	}
}
