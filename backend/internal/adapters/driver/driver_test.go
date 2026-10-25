package driver

import (
	"context"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestDatabaseExecutorForType(t *testing.T) {
	manager := NewDatabasePoolManager()
	t.Cleanup(func() { _ = manager.CloseAll() })
	executor := NewDatabaseExecutor(manager)

	if got, err := databaseExecutorForType(executor, "postgresql"); err != nil || got != executor.postgresqlExecutor {
		t.Fatalf("postgresql executor=%T err=%v", got, err)
	}
	if got, err := databaseExecutorForType(executor, "mysql"); err != nil || got != executor.mysqlExecutor {
		t.Fatalf("mysql executor=%T err=%v", got, err)
	}
	if _, err := databaseExecutorForType(executor, "sqlite"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported type error=%v", err)
	}
}

func TestConnectionTesterRejectsUnsupportedType(t *testing.T) {
	tester := NewDatabaseConnectionTester()
	err := tester.Test(context.Background(), domain.Database{Type: "sqlite"})
	if err == nil || !strings.Contains(err.Error(), "unsupported database type") {
		t.Fatalf("error=%v", err)
	}
}

func TestDatabaseExecutorRejectsUnsupportedTypeBeforeConnecting(t *testing.T) {
	manager := NewDatabasePoolManager()
	t.Cleanup(func() { _ = manager.CloseAll() })
	executor := NewDatabaseExecutor(manager)
	database := domain.Database{Type: "sqlite"}

	if _, err := executor.ExecuteQuery(context.Background(), database, "SELECT 1"); err == nil {
		t.Fatal("ExecuteQuery must reject unsupported type")
	}
	if _, err := executor.ExecuteCommand(context.Background(), database, "CREATE TABLE x(id int)"); err == nil {
		t.Fatal("ExecuteCommand must reject unsupported type")
	}
	if _, err := executor.ExplainQuery(context.Background(), database, "SELECT 1", false); err == nil {
		t.Fatal("ExplainQuery must reject unsupported type")
	}
	if _, err := executor.GetDatabaseMetadata(context.Background(), database); err == nil {
		t.Fatal("GetDatabaseMetadata must reject unsupported type")
	}
}
