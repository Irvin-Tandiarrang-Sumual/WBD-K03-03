//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestIntegrationMigrationCreateHistoryCompareAndRollback(t *testing.T) {
	stack := newIntegrationStack(t, domain.AccessViewEdit)
	ctx := context.Background()

	if err := stack.db.Exec(`DROP TABLE IF EXISTS integration_migration_items`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stack.db.Exec(`DROP TABLE IF EXISTS integration_migration_items`).Error
	})

	first, err := stack.migrationService.CreateMigration(
		ctx,
		stack.target.ID,
		stack.user.ID,
		"CREATE TABLE integration_migration_items (id integer PRIMARY KEY, name text NOT NULL)",
	)
	if err != nil {
		t.Fatalf("CreateMigration(first) error=%v", err)
	}
	if first.Version != "1" || first.Status != "success" || first.SqlDown != "DROP TABLE integration_migration_items" {
		t.Fatalf("unexpected first migration=%+v", first)
	}
	if !stack.db.Migrator().HasTable("integration_migration_items") {
		t.Fatal("first migration did not create target table")
	}

	second, err := stack.migrationService.CreateMigration(
		ctx,
		stack.target.ID,
		stack.user.ID,
		"ALTER TABLE integration_migration_items ADD COLUMN note text",
	)
	if err != nil {
		t.Fatalf("CreateMigration(second) error=%v", err)
	}
	if second.Version != "2" || second.SqlDown != "ALTER TABLE integration_migration_items DROP COLUMN note" {
		t.Fatalf("unexpected second migration=%+v", second)
	}
	if !stack.db.Migrator().HasColumn("integration_migration_items", "note") {
		t.Fatal("second migration did not add note column")
	}

	history, total, err := stack.migrationService.GetMigrationsByDatabaseID(
		ctx,
		stack.target.ID,
		pagination.NewPaginationParams(1, 10),
	)
	if err != nil {
		t.Fatalf("GetMigrationsByDatabaseID() error=%v", err)
	}
	if total != 2 || len(history) != 2 || history[0].Version != "1" || history[1].Version != "2" {
		t.Fatalf("unexpected migration history total=%d history=%+v", total, history)
	}

	comparison, err := stack.migrationService.CompareMigrations(ctx, stack.target.ID, stack.user.ID, first.ID, second.ID)
	if err != nil {
		t.Fatalf("CompareMigrations() error=%v", err)
	}
	if comparison.SQLUpEqual || comparison.SQLDownEqual {
		t.Fatalf("different migrations unexpectedly reported equal: %+v", comparison)
	}

	rolledBack, err := stack.migrationService.RollbackMigrations(ctx, stack.target.ID, stack.user.ID, first.ID)
	if err != nil {
		t.Fatalf("RollbackMigrations() error=%v", err)
	}
	if len(rolledBack) != 2 {
		t.Fatalf("rollback result length=%d, want 2; result=%+v", len(rolledBack), rolledBack)
	}
	if rolledBack[0].ID != second.ID {
		t.Fatalf("first rollback entry=%+v, want migration 2", rolledBack[0])
	}
	rollbackRecord := rolledBack[1]
	if rollbackRecord.Version != "3" || rollbackRecord.SqlUp != second.SqlDown || rollbackRecord.SqlDown != second.SqlUp {
		t.Fatalf("unexpected rollback record=%+v", rollbackRecord)
	}
	if stack.db.Migrator().HasColumn("integration_migration_items", "note") {
		t.Fatal("rollback did not remove note column")
	}

	history, total, err = stack.migrationService.GetMigrationsByDatabaseID(ctx, stack.target.ID, pagination.NewPaginationParams(1, 10))
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(history) != 3 || history[2].Version != "3" {
		t.Fatalf("history after rollback total=%d history=%+v", total, history)
	}

	latest, err := stack.migrationRepo.FindLatestByDatabaseID(ctx, stack.target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != rollbackRecord.ID || latest.Version != "3" {
		t.Fatalf("latest=%+v rollback=%+v", latest, rollbackRecord)
	}
}
