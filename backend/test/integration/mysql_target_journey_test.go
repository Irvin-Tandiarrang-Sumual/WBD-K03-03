//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestMySQLTarget_CreateMissingDatabase_GrantsAccessOnlyToRequestedDatabase(t *testing.T) {
	// Arrange
	ctx := context.Background()
	target := requireTargetMySQL(t)
	target.Name = "grant_scope_test"
	pools := driver.NewDatabasePoolManager()
	t.Cleanup(func() { _ = pools.CloseAll() })
	executor := driver.NewDatabaseDriver(pools)

	// Act
	err := driver.NewDatabaseConnectionTester().Test(ctx, target)
	if err != nil {
		t.Fatalf("provision MySQL target: %v", err)
	}
	grants, err := executor.ExecuteQuery(ctx, target, "SHOW GRANTS FOR CURRENT_USER")

	// Assert
	if err != nil {
		t.Fatalf("read grants: %v", err)
	}
	hasDatabaseGrant := false
	for _, row := range grants.Rows {
		for _, value := range row {
			grant := fmt.Sprint(value)
			if strings.Contains(grant, "ON *.*") && !strings.HasPrefix(grant, "GRANT USAGE ") {
				t.Errorf("unexpected server-wide grant: %s", grant)
			}
			if strings.Contains(grant, "ON `grant\\_scope\\_test`.*") {
				hasDatabaseGrant = true
			}
		}
	}
	if !hasDatabaseGrant {
		t.Fatalf("expected a literal grant for the requested database, got %v", grants.Rows)
	}
}

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
		LeaderID: createMigrationLeader(t, stack, "mysql-analysts-group-leader").ID,
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
		LeaderID: createMigrationLeader(t, stack, "mysql-readonly-enforcement-leader").ID,
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
		LeaderID: createMigrationLeader(t, stack, "mysql-devs-group-leader").ID,
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
		LeaderID: createMigrationLeader(t, stack, "mysql-migrators-group-leader").ID,
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

	leader := signIn(t, stack, "mysql-migrators-group-leader")
	migration := applyTestMigration(t, stack, createdDB.ID, testSession{sessionCookie, csrfToken},
		sqlUp, "DROP TABLE mysql_products", leader)

	// Assert
	if migration.Status != domain.MigrationStatusSuccess || migration.Version == "" || len(migration.Snapshot) == 0 {
		t.Fatalf("expected versioned migration with a snapshot, got %+v", migration)
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

	author := testSession{sessionCookie, csrfToken}
	leaderCookie, leaderToken := loginUser(t, stack.engine, leader.Username, "password123")
	voters := []testSession{{leaderCookie, leaderToken}}

	// Step 1: Apply baseline migration 1
	applyTestMigration(t, stack, createdDB.ID, author,
		"CREATE TABLE mysql_base_tbl (id INT PRIMARY KEY);", "DROP TABLE mysql_base_tbl", voters...)

	// Step 2: Apply second migration to be rolled back
	applyTestMigration(t, stack, createdDB.ID, author,
		"CREATE TABLE mysql_rollback_tbl (id INT PRIMARY KEY, title VARCHAR(50));", "DROP TABLE mysql_rollback_tbl", voters...)

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
	vote := executeRequest(stack.engine, http.MethodPost, path+"/vote", map[string]bool{"approve": true}, leaderCookie, leaderToken)
	if vote.Code != http.StatusNoContent {
		t.Fatalf("vote: %s", vote.Body.String())
	}
	recRollback := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, sessionCookie, csrfToken)

	// Assert - Rollback execution success
	if recRollback.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted on rollback in MySQL, got %d: %s", recRollback.Code, recRollback.Body.String())
	}

	waitForAppliedMigration(t, stack, createdDB.ID, draft.ID, author)

	// Verify table mysql_rollback_tbl is dropped in MySQL container
	verifyReq := handlers.DatabaseQueryRequest{Query: "SELECT * FROM mysql_rollback_tbl;"}
	recVerify := executeRequest(stack.engine, http.MethodPost, fmt.Sprintf("/api/v1/me/databases/%d/query", createdDB.ID), verifyReq, sessionCookie, csrfToken)
	if recVerify.Code == http.StatusOK {
		t.Fatalf("expected query to fail because table should be dropped by rollback, but got 200 OK")
	}
}
