//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestPostgresTarget_GivenUserWithReadOnlyAccess_WhenExecuteSelectQuery_ThenReturnRowsSuccessfully(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	// Register Target PostgreSQL in platform DB
	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	// Create Group with AccessViewOnly
	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-analysts-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatalf("failed to create view-only group: %v", err)
	}

	// Create user & login
	user := createRegularUser(t, stack, "pg_analyst_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Execute read-only SELECT
	queryReq := handlers.DatabaseQueryRequest{
		Query: "SELECT 1 AS col_val, 'pg_test' AS name_val;",
	}
	urlPath := fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, urlPath, queryReq, sessionCookie, csrfToken)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on SELECT query, got %d: %s", rec.Code, rec.Body.String())
	}

	res := parseResponseBody[*ports.DatabaseQueryResult](t, rec)
	if len(res.Data.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(res.Data.Rows))
	}
}

func TestPostgresTarget_GivenUserWithReadOnlyAccess_WhenExecuteMutatingCommand_ThenRejectWithForbidden(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-readonly-enforcement",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_readonly_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Attempting to execute mutating DDL/DML
	queryReq := handlers.DatabaseQueryRequest{
		Query: "CREATE TABLE unauthorized_tbl (id INT);",
	}
	urlPath := fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, urlPath, queryReq, sessionCookie, csrfToken)

	// Assert - AccessViewOnly forbids mutations
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 403 Forbidden or 400 on mutating query for read-only user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostgresTarget_GivenUserWithReadWriteAccess_WhenExecuteExplainAndExplainAnalyze_ThenReturnValidExecutionPlan(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-devs-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_dev_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act 1 - EXPLAIN
	explainReq := handlers.DatabaseQueryRequest{Query: "SELECT 1;"}
	explainURL := fmt.Sprintf("/api/v1/me/databases/%d/explain", createdDB.ID)
	recExplain := executeRequest(stack.engine, http.MethodPost, explainURL, explainReq, sessionCookie, csrfToken)

	// Assert 1
	if recExplain.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on EXPLAIN, got %d: %s", recExplain.Code, recExplain.Body.String())
	}

	// Act 2 - EXPLAIN ANALYZE
	explainAnalyzeURL := fmt.Sprintf("/api/v1/me/databases/%d/explain-analyze", createdDB.ID)
	recAnalyze := executeRequest(stack.engine, http.MethodPost, explainAnalyzeURL, explainReq, sessionCookie, csrfToken)

	// Assert 2
	if recAnalyze.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on EXPLAIN ANALYZE, got %d: %s", recAnalyze.Code, recAnalyze.Body.String())
	}
}

func TestPostgresTarget_GivenUserWithReadWriteAccess_WhenApplySQLUpMigration_ThenSchemaUpdatedAndSnapshotRecorded(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-migrators-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_migrator_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Apply SQL Up Migration
	sqlUp := `CREATE TABLE pg_products (
		id INT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price NUMERIC(10,2) NOT NULL
	);
	INSERT INTO pg_products (id, name, price) VALUES (1, 'Widget', 19.99);`

	migrationReq := handlers.CreateMigrationRequest{SqlUp: sqlUp}
	migrationURL := fmt.Sprintf("/api/v1/me/databases/%d/migrations", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, migrationURL, migrationReq, sessionCookie, csrfToken)

	// Assert - Migration creation success
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on migration, got %d: %s", rec.Code, rec.Body.String())
	}

	res := parseResponseBody[handlers.MigrationResponse](t, rec)
	if res.Data.Status != "success" {
		t.Fatalf("expected migration status success, got %q", res.Data.Status)
	}

	// Verify table physically exists in Target PostgreSQL
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT name, price FROM pg_products WHERE id = 1;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 OK verifying created table, got %d: %s", recVerify.Code, recVerify.Body.String())
	}
}

func TestPostgresTarget_GivenAppliedMigration_WhenRollbackTriggered_ThenCompensatingSQLDownRestoresPreviousSchema(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target postgres: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "pg-rollback-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "pg_rollback_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Step 1: Apply initial baseline migration 1
	rec1 := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/migrations", createdDB.ID),
		handlers.CreateMigrationRequest{SqlUp: "CREATE TABLE pg_base_tbl (id INT PRIMARY KEY);"},
		sessionCookie, csrfToken)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("failed to create initial baseline migration: %s", rec1.Body.String())
	}
	res1 := parseResponseBody[handlers.MigrationResponse](t, rec1)

	// Step 2: Apply second migration to be rolled back
	rec2 := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/migrations", createdDB.ID),
		handlers.CreateMigrationRequest{SqlUp: "CREATE TABLE pg_rollback_tbl (id INT PRIMARY KEY, title VARCHAR(50));"},
		sessionCookie, csrfToken)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("failed to create second migration: %s", rec2.Body.String())
	}

	// Act - Trigger Rollback targeting baseline migration 1
	rollbackReq := handlers.RollbackMigrationRequest{
		DatabaseID:        createdDB.ID,
		TargetMigrationID: res1.Data.ID,
	}
	recRollback := executeRequest(stack.engine, http.MethodPost, "/api/v1/me/databases/migrations/rollback", rollbackReq, sessionCookie, csrfToken)

	// Assert - Rollback execution success
	if recRollback.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on rollback, got %d: %s", recRollback.Code, recRollback.Body.String())
	}

	// Verify table pg_rollback_tbl is dropped in PostgreSQL target
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT * FROM pg_rollback_tbl;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code == http.StatusOK {
		t.Fatalf("expected query to fail because table should be dropped by rollback, but got 200 OK")
	}
}
