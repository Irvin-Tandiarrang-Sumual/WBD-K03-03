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

func TestMySQLTarget_GivenUserWithReadOnlyAccess_WhenExecuteSelectQuery_ThenReturnRowsSuccessfully(t *testing.T) {
	// Arrange
	targetConfig := requireTargetMySQL(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	// Register Target MySQL in platform DB
	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target mysql: %v", err)
	}

	// Create Group with AccessViewOnly
	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "mysql-analysts-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatalf("failed to create view-only group: %v", err)
	}

	user := createRegularUser(t, stack, "mysql_analyst_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Execute read-only SELECT
	queryReq := handlers.DatabaseQueryRequest{
		Query: "SELECT 1 AS col_val, 'mysql_test' AS name_val;",
	}
	urlPath := fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, urlPath, queryReq, sessionCookie, csrfToken)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on SELECT query in MySQL target, got %d: %s", rec.Code, rec.Body.String())
	}

	res := parseResponseBody[*ports.DatabaseQueryResult](t, rec)
	if len(res.Data.Rows) != 1 {
		t.Fatalf("expected 1 row from MySQL, got %d", len(res.Data.Rows))
	}
}

func TestMySQLTarget_GivenUserWithReadOnlyAccess_WhenExecuteMutatingCommand_ThenRejectWithForbidden(t *testing.T) {
	// Arrange
	targetConfig := requireTargetMySQL(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target mysql: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "mysql-readonly-enforcement",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "mysql_readonly_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Attempting mutating command
	queryReq := handlers.DatabaseQueryRequest{
		Query: "CREATE TABLE unauthorized_mysql_tbl (id INT);",
	}
	urlPath := fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, urlPath, queryReq, sessionCookie, csrfToken)

	// Assert - Rejected
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 403 Forbidden or 400 on mutating query for read-only user in MySQL, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMySQLTarget_GivenUserWithReadWriteAccess_WhenExecuteExplain_ThenReturnValidExecutionPlan(t *testing.T) {
	// Arrange
	targetConfig := requireTargetMySQL(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target mysql: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "mysql-devs-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "mysql_dev_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - EXPLAIN on MySQL
	explainReq := handlers.DatabaseQueryRequest{Query: "SELECT 1;"}
	explainURL := fmt.Sprintf("/api/v1/me/databases/%d/explain", createdDB.ID)
	recExplain := executeRequest(stack.engine, http.MethodPost, explainURL, explainReq, sessionCookie, csrfToken)

	// Assert
	if recExplain.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on EXPLAIN in MySQL target, got %d: %s", recExplain.Code, recExplain.Body.String())
	}
}

func TestMySQLTarget_GivenUserWithReadWriteAccess_WhenApplySQLUpMigration_ThenSchemaUpdatedAndSnapshotRecorded(t *testing.T) {
	// Arrange
	targetConfig := requireTargetMySQL(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target mysql: %v", err)
	}

	group, err := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "mysql-migrators-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "mysql_migrator_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Act - Apply SQL Up Migration on MySQL dialect
	sqlUp := `CREATE TABLE mysql_products (
		id INT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price DECIMAL(10,2) NOT NULL
	);`

	migrationReq := handlers.CreateMigrationRequest{SqlUp: sqlUp}
	migrationURL := fmt.Sprintf("/api/v1/me/databases/%d/migrations", createdDB.ID)
	rec := executeRequest(stack.engine, http.MethodPost, migrationURL, migrationReq, sessionCookie, csrfToken)

	// Assert
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on migration in MySQL, got %d: %s", rec.Code, rec.Body.String())
	}

	res := parseResponseBody[handlers.MigrationResponse](t, rec)
	if res.Data.Status != "success" {
		t.Fatalf("expected migration status success, got %q", res.Data.Status)
	}

	// Verify table physically exists in MySQL container
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT id, name, price FROM mysql_products;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 OK verifying created table in MySQL, got %d: %s", recVerify.Code, recVerify.Body.String())
	}
}

func TestMySQLTarget_GivenAppliedMigration_WhenRollbackTriggered_ThenCompensatingSQLDownRestoresPreviousSchema(t *testing.T) {
	// Arrange
	targetConfig := requireTargetMySQL(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	createdDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target mysql: %v", err)
	}

	leader := createMigrationLeader(t, stack, "mysql_rollback_leader")
	group, err := stack.groupRepo.Save(ctx, domain.Group{
		LeaderID: leader.ID,
		Name:     "mysql-rollback-group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: createdDB.ID, AccessType: domain.AccessViewEdit},
		},
	})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	user := createRegularUser(t, stack, "mysql_rollback_user", "password123", []uint{group.ID})
	sessionCookie, csrfToken := loginUser(t, stack.engine, user.Username, "password123")

	// Step 1: Apply baseline migration 1
	rec1 := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/migrations", createdDB.ID),
		handlers.CreateMigrationRequest{SqlUp: "CREATE TABLE mysql_base_tbl (id INT PRIMARY KEY);"},
		sessionCookie, csrfToken)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("failed to create initial baseline migration in MySQL: %s", rec1.Body.String())
	}

	// Step 2: Apply second migration to be rolled back
	rec2 := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/migrations", createdDB.ID),
		handlers.CreateMigrationRequest{SqlUp: "CREATE TABLE mysql_rollback_tbl (id INT PRIMARY KEY, title VARCHAR(50));"},
		sessionCookie, csrfToken)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("failed to create second migration in MySQL: %s", rec2.Body.String())
	}

	// Act
	base := fmt.Sprintf("/api/v1/me/databases/%d", createdDB.ID)
	proposal := executeRequest(stack.engine, http.MethodPost, base+"/rollback-draft", nil, sessionCookie, csrfToken)
	if proposal.Code != http.StatusCreated {
		t.Fatalf("propose: %s", proposal.Body.String())
	}
	draft := parseResponseBody[handlers.MigrationDraftResponse](t, proposal).Data
	path := fmt.Sprintf("%s/migration-drafts/%d", base, draft.ID)
	blocked := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)
	if blocked.Code != http.StatusConflict {
		t.Fatalf("expected unapproved execution to fail: %s", blocked.Body.String())
	}
	leaderCookie, leaderToken := loginUser(t, stack.engine, leader.Username, "password123")
	vote := executeRequest(stack.engine, http.MethodPost, path+"/vote", map[string]bool{"approve": true}, leaderCookie, leaderToken)
	if vote.Code != http.StatusNoContent {
		t.Fatalf("vote: %s", vote.Body.String())
	}
	recRollback := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)

	// Assert - Rollback execution success
	if recRollback.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on rollback in MySQL, got %d: %s", recRollback.Code, recRollback.Body.String())
	}

	// Verify table mysql_rollback_tbl is dropped in MySQL container
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT * FROM mysql_rollback_tbl;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code == http.StatusOK {
		t.Fatalf("expected query to fail because table should be dropped by rollback, but got 200 OK")
	}
}
